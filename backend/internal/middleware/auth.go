package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

// AuthRequired é o middleware real de validação de token JWT.
func AuthRequired(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Busca pelo cookie access_token
			cookie, err := r.Cookie("access_token")
			if err != nil || cookie == nil {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão ausente. Faça login novamente.")
				return
			}

			// Realiza o parse e a validação do JWT
			token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
				// Valida se o algoritmo de assinatura é o correto (HMAC SHA-256)
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("algoritmo de assinatura inesperado: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão inválida ou expirada. Refaça o login.")
				return
			}

			// Extrai as claims e injeta o user_id no contexto da requisição
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || claims["user_id"] == nil {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Erro ao processar as credenciais.")
				return
			}

			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, "ID de usuário inválido nas credenciais.")
				return
			}

			// Injeta o UserID no contexto da requisição
			ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IsOriginAllowed verifica se a origem da requisição é permitida segundo as regras de negócio e ambiente.
func IsOriginAllowed(origin, configuredURL, env, reqHost string) bool {
	if origin == "" {
		return false
	}
	// 1. Suporta correspondência exata ou URLs separadas por vírgula em FRONTEND_URL
	if configuredURL != "" {
		for _, u := range strings.Split(configuredURL, ",") {
			if strings.TrimSpace(u) == origin {
				return true
			}
		}
	}
	// 2. Em produção, apenas URLs explicitamente cadastradas são permitidas
	if env == "production" {
		return false
	}
	// 3. Em desenvolvimento, se FRONTEND_URL não estiver configurado, permite localhost:3000 por padrão
	if configuredURL == "" && origin == "http://localhost:3000" {
		return true
	}
	// 4. Em desenvolvimento, se a origem tiver o mesmo hostname do servidor acessado (ex: IP de rede local como 192.168.x.x)
	originHost := extractHost(origin)
	reqHostname := reqHost
	if h, _, err := net.SplitHostPort(reqHost); err == nil {
		reqHostname = h
	}
	if originHost != "" && reqHostname != "" && reqHostname != "localhost" && reqHostname != "127.0.0.1" && originHost == reqHostname {
		return true
	}
	return false
}

func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// CORS configura as permissões de compartilhamento de recursos entre origens de forma extremamente segura.
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := IsOriginAllowed(origin, config.Envs.FrontendURL, config.Envs.Env, r.Host)

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Idempotency-Key, X-Admin-Key")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}

			// Responde imediatamente a requisições de preflight do browser (OPTIONS)
			if r.Method == http.MethodOptions {
				if allowed {
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
