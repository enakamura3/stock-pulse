package telegram

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/telebot.v3"
)

type BotRunner struct {
	bot      *telebot.Bot
	handlers *Handlers
}

func NewBotRunner(token string, handlers *Handlers) (*BotRunner, error) {
	if token == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN não configurado. Bot do Telegram não será iniciado.")
		return nil, nil
	}

	return NewBotRunnerWithSettings(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}, handlers)
}

func NewBotRunnerWithSettings(pref telebot.Settings, handlers *Handlers) (*BotRunner, error) {
	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	b.Use(rateLimitMiddleware())

	handlers.Register(b)

	// Adiciona o Menu dinâmico nativo do Telegram (Botão "Menu" ao lado da caixa de texto)
	_ = b.SetCommands([]telebot.Command{
		{Text: "menu", Description: "Abrir o menu principal"},
	})

	return &BotRunner{
		bot:      b,
		handlers: handlers,
	}, nil
}

func (r *BotRunner) Start() {
	if r == nil || r.bot == nil {
		return
	}
	slog.Info("Iniciando Bot do Telegram em background...")
	r.bot.Start()
}

func (r *BotRunner) Stop() {
	if r == nil || r.bot == nil {
		return
	}
	slog.Info("Parando Bot do Telegram...")
	r.bot.Stop()
}

func (r *BotRunner) GetUsername() string {
	if r == nil || r.bot == nil || r.bot.Me == nil {
		return ""
	}
	return r.bot.Me.Username
}

func (r *BotRunner) SendAlertMessage(chatID int64, userName, ticker, assetName string, currentVal, targetVal float64, condition, currency string) error {
	if r == nil || r.bot == nil {
		return nil // Bot is disabled
	}

	condStr := "acima de"
	if condition == "BELOW" {
		condStr = "abaixo de"
	}

	msg := "🚨 *ALERTA DE PREÇO DISPARADO* 🚨\n\n"
	msg += "Olá, *" + userName + "*!\n"
	msg += "Seu alerta para o ativo *" + ticker + "* (" + assetName + ") foi atingido.\n\n"
	msg += "📊 *Preço Atual:* " + currency + " " + fmt.Sprintf("%.2f", currentVal) + "\n"
	msg += "🎯 *Seu Alvo (" + condStr + "):* " + currency + " " + fmt.Sprintf("%.2f", targetVal) + "\n\n"
	msg += "Acesse o *Stock Pulse* para mais detalhes."

	_, err := r.bot.Send(&telebot.Chat{ID: chatID}, msg, telebot.ModeMarkdown)
	return err
}

func rateLimitMiddleware() telebot.MiddlewareFunc {
	var mu sync.Mutex
	limiters := make(map[int64]*rate.Limiter)

	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}

			mu.Lock()
			l, exists := limiters[sender.ID]
			if !exists {
				// Permite 1 mensagem por segundo com burst de até 3 mensagens simultâneas
				l = rate.NewLimiter(rate.Every(time.Second), 3)
				limiters[sender.ID] = l
			}
			mu.Unlock()

			if !l.Allow() {
				slog.Warn("Rate limit exceeded for user", "userID", sender.ID, "username", sender.Username)
				return c.Send("⚠️ Você está enviando mensagens muito rápido. Por favor, aguarde um momento antes de enviar a próxima.")
			}

			return next(c)
		}
	}
}
