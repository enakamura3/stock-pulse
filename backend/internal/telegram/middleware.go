package telegram

import (
	"context"
	"strings"

	"gopkg.in/telebot.v3"
)

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
