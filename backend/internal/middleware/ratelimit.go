package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/httputils"
	"github.com/redis/go-redis/v9"
)

// rateLimitLuaScript incrementa a contagem de acessos de forma atômica e define o TTL caso não exista.
var rateLimitLuaScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
local ttl = redis.call('TTL', KEYS[1])
if ttl < 0 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
    ttl = tonumber(ARGV[1])
end
return {current, ttl}
`)

// RateLimiterConfig define os parâmetros de limitação de requisições.
type RateLimiterConfig struct {
	// Limit é o número máximo de requisições permitidas dentro da Window.
	Limit int
	// Window é a janela temporal de rate limiting (ex: 1 * time.Minute).
	Window time.Duration
	// KeyPrefix é o prefixo da chave no Redis (ex: "rate_limit:auth:login").
	KeyPrefix string
	// KeyExtractor extrai o identificador da requisição (ex: IP ou userID). Se nil, usa ClientIP(r).
	KeyExtractor func(r *http.Request) string
}

// ClientIP extrai o IP real do cliente a partir dos cabeçalhos HTTP ou do RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		ip := strings.TrimSpace(xrip)
		if ip != "" {
			return ip
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}

// RateLimit cria um middleware de limitação de requisições baseado em Redis.
func RateLimit(rdb redis.Cmdable, cfg RateLimiterConfig) func(http.Handler) http.Handler {
	if rdb == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	if cfg.Limit <= 0 {
		cfg.Limit = 60
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "rate_limit"
	}
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = ClientIP
	}

	ttlSeconds := int(cfg.Window.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := cfg.KeyExtractor(r)
			if identifier == "" {
				identifier = "unknown"
			}

			key := fmt.Sprintf("%s:%s", cfg.KeyPrefix, identifier)
			ctx := r.Context()

			res, err := rateLimitLuaScript.Run(ctx, rdb, []string{key}, ttlSeconds).Result()
			if err != nil {
				slog.Warn("Rate limiter Redis script error, failing open", "key", key, "error", err)
				next.ServeHTTP(w, r)
				return
			}

			vals, ok := res.([]interface{})
			if !ok || len(vals) < 2 {
				slog.Warn("Rate limiter invalid Redis response format, failing open", "key", key)
				next.ServeHTTP(w, r)
				return
			}

			current, _ := vals[0].(int64)
			ttl, _ := vals[1].(int64)
			if ttl <= 0 {
				ttl = int64(ttlSeconds)
			}

			remaining := int64(cfg.Limit) - current
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(ttl, 10))

			if current > int64(cfg.Limit) {
				w.Header().Set("Retry-After", strconv.FormatInt(ttl, 10))
				httputils.RespondWithError(w, http.StatusTooManyRequests, "Muitas requisições. Tente novamente mais tarde.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitAuthLogin limita tentativas de login a 10 requisições por minuto por IP.
func RateLimitAuthLogin(rdb redis.Cmdable) func(http.Handler) http.Handler {
	return RateLimit(rdb, RateLimiterConfig{
		Limit:     10,
		Window:    time.Minute,
		KeyPrefix: "rate_limit:auth:login",
	})
}

// RateLimitAuthRegister limita cadastros a 5 requisições por minuto por IP.
func RateLimitAuthRegister(rdb redis.Cmdable) func(http.Handler) http.Handler {
	return RateLimit(rdb, RateLimiterConfig{
		Limit:     5,
		Window:    time.Minute,
		KeyPrefix: "rate_limit:auth:register",
	})
}

// RateLimitAuthRefresh limita renovação de tokens a 30 requisições por minuto por IP.
func RateLimitAuthRefresh(rdb redis.Cmdable) func(http.Handler) http.Handler {
	return RateLimit(rdb, RateLimiterConfig{
		Limit:     30,
		Window:    time.Minute,
		KeyPrefix: "rate_limit:auth:refresh",
	})
}

// RateLimitTelegramLink limita a geração de tokens de vinculação do Telegram a 10 requisições por minuto por IP.
func RateLimitTelegramLink(rdb redis.Cmdable) func(http.Handler) http.Handler {
	return RateLimit(rdb, RateLimiterConfig{
		Limit:     10,
		Window:    time.Minute,
		KeyPrefix: "rate_limit:telegram:link",
	})
}
