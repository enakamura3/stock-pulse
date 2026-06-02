package telegram

import (
	"log/slog"
	"time"

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

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

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
