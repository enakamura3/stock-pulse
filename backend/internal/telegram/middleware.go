package telegram

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/telebot.v3"
)

type userLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitMiddleware cria um middleware telebot que limita requisições por usuário (sender.ID).
// Permite 1 mensagem por segundo com burst de até 3 mensagens, com expiração de limitadores inativos após 10 minutos.
func RateLimitMiddleware() telebot.MiddlewareFunc {
	return NewRateLimitMiddleware(rate.Every(time.Second), 3, 10*time.Minute)
}

// NewRateLimitMiddleware permite customizar a taxa, burst e TTL de expiração dos limitadores de taxa por usuário.
func NewRateLimitMiddleware(r rate.Limit, burst int, ttl time.Duration) telebot.MiddlewareFunc {
	var mu sync.Mutex
	limiters := make(map[int64]*userLimiter)

	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}

			now := time.Now()

			mu.Lock()
			// Limpeza oportuna de limitadores inativos caso o mapa acumule mais de 100 usuários
			if len(limiters) > 100 {
				for id, ul := range limiters {
					if now.Sub(ul.lastSeen) > ttl {
						delete(limiters, id)
					}
				}
			}

			ul, exists := limiters[sender.ID]
			if !exists {
				ul = &userLimiter{
					limiter: rate.NewLimiter(r, burst),
				}
				limiters[sender.ID] = ul
			}
			ul.lastSeen = now
			l := ul.limiter
			mu.Unlock()

			if !l.Allow() {
				slog.Warn("Rate limit exceeded for Telegram user", "userID", sender.ID, "username", sender.Username)
				if c.Callback() != nil {
					_ = c.Respond(&telebot.CallbackResponse{
						Text:      "⚠️ Muitas requisições. Aguarde um momento antes de tentar novamente.",
						ShowAlert: true,
					})
					return nil
				}
				return c.Send("⚠️ Você está enviando mensagens muito rápido. Por favor, aguarde um momento antes de enviar a próxima.")
			}

			return next(c)
		}
	}
}

func (h *Handlers) AuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		// Ignore /start as it is used for linking
		if c.Message() != nil && strings.HasPrefix(c.Text(), "/start") {
			return next(c)
		}

		userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
		if err != nil {
			if c.Callback() != nil {
				c.Respond()
			}
			return c.Send("⚠️ Sua conta não está vinculada. Gere um link no painel do Stock Pulse.")
		}

		c.Set("user_id", userID.String())
		return next(c)
	}
}
