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

	// Adiciona os comandos no Menu dinâmico nativo do Telegram (Botão "Menu" ao lado da caixa de texto)
	_ = b.SetCommands([]telebot.Command{
		{Text: "menu", Description: "Abrir o menu principal"},
		{Text: "cotacao", Description: "Consultar cotação de ativo"},
		{Text: "agenda", Description: "Agenda de proventos (30 dias)"},
		{Text: "analise", Description: "Análise fundamentalista de ativo"},
		{Text: "desfazer", Description: "Desfazer última operação lançada"},
		{Text: "help", Description: "Exibir comandos e ajuda"},
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

	escapedUserName := escapeMarkdown(userName)
	escapedTicker := escapeMarkdown(ticker)
	escapedAssetName := escapeMarkdown(assetName)

	msg := "🚨 *ALERTA DE PREÇO DISPARADO* 🚨\n\n"
	msg += "Olá, *" + escapedUserName + "*!\n"
	msg += "Seu alerta para o ativo *" + escapedTicker + "* (" + escapedAssetName + ") foi atingido.\n\n"
	msg += "📊 *Preço Atual:* " + currency + " " + formatFinancialPrice(nil, currentVal) + "\n"
	msg += "🎯 *Seu Alvo (" + condStr + "):* " + currency + " " + formatFinancialPrice(nil, targetVal) + "\n\n"
	msg += "Acesse o *Stock Pulse* para mais detalhes."

	_, err := r.bot.Send(&telebot.Chat{ID: chatID}, msg, telebot.ModeMarkdown)
	if err != nil && isBlockedByUser(err) {
		slog.Warn("Usuário bloqueou o bot do Telegram ao receber alerta", "chatID", chatID, "error", err)
	}
	return err
}

func rateLimitMiddleware() telebot.MiddlewareFunc {
	return RateLimitMiddleware()
}
