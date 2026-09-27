package telegram

import (
	"log/slog"
	"math"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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
		{Text: "resumo", Description: "Resumo da carteira ativa"},
		{Text: "ativos", Description: "Listar ativos em carteira"},
		{Text: "rendafixa", Description: "Consultar posições de renda fixa"},
		{Text: "operacao", Description: "Lançar nova operação"},
		{Text: "alertas", Description: "Gerenciar alertas de preço"},
		{Text: "watchlist", Description: "Listar e gerenciar favoritos"},
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

	menu := &telebot.ReplyMarkup{}
	btnQuote := menu.Data("📈 Ver Cotação", "btn_quote_"+ticker)
	btnNewOp := menu.Data("➕ Nova Operação", "btn_operacao")
	btnAlerts := menu.Data("🔔 Meus Alertas", "btn_alerts")
	menu.Inline(menu.Row(btnQuote, btnNewOp), menu.Row(btnAlerts))

	_, err := r.bot.Send(&telebot.Chat{ID: chatID}, msg, telebot.ModeMarkdown, menu)
	if err != nil && isBlockedByUser(err) {
		slog.Warn("Usuário bloqueou o bot do Telegram ao receber alerta", "chatID", chatID, "error", err)
	}
	return err
}

func (r *BotRunner) SendDividendPaymentAlert(chatID int64, userName, portfolioName string, items []portfolio.DividendNotificationItem) error {
	if r == nil || r.bot == nil || len(items) == 0 {
		return nil
	}

	p := message.NewPrinter(language.Portuguese)
	escapedUserName := escapeMarkdown(userName)
	escapedPortfolioName := escapeMarkdown(portfolioName)

	msg := "💰 *PROVENTO PAGO: Caiu na conta!* 💰\n\n"
	msg += "Olá, *" + escapedUserName + "*!\n"
	msg += "Hoje é data de pagamento de proventos na sua carteira *" + escapedPortfolioName + "*:\n\n"

	totalCredited := 0.0
	currency := "BRL"
	for _, it := range items {
		if it.Currency != "" {
			currency = it.Currency
		}
		totalCredited += it.NetAmount

		escapedTicker := escapeMarkdown(it.Ticker)
		escapedType := escapeMarkdown(it.Type)
		perShareStr := formatFinancialPrice(p, it.PerShareAmount)
		totalItemStr := formatFinancialPrice(p, it.NetAmount)
		qtyStr := p.Sprintf("%.2f", it.Quantity)
		if math.Abs(it.Quantity-math.Round(it.Quantity)) < 1e-6 {
			qtyStr = p.Sprintf("%.0f", it.Quantity)
		}

		msg += "• *" + escapedTicker + "* (" + escapedType + ")\n"
		msg += "  ▫ Valor por cota: " + currency + " " + perShareStr + "\n"
		msg += "  ▫ Quantidade: " + qtyStr + " cotas\n"
		msg += "  ▫ Total creditado: *" + currency + " " + totalItemStr + "*\n\n"
	}

	if len(items) > 1 {
		msg += "💵 *Total Geral Creditado Hoje:* *" + currency + " " + formatFinancialPrice(p, totalCredited) + "*\n\n"
	}
	msg += "Acesse o *Stock Pulse* para conferir seu extrato atualizado."

	menu := &telebot.ReplyMarkup{}
	var buttons []telebot.Btn
	if len(items) == 1 {
		buttons = append(buttons, menu.Data("📈 Ver Cotação", "btn_quote_"+items[0].Ticker))
	}
	buttons = append(buttons, menu.Data("📅 Minha Agenda", "btn_agenda"))
	buttons = append(buttons, menu.Data("💼 Menu Principal", "btn_menu"))

	menu.Inline(menu.Row(buttons...))

	_, err := r.bot.Send(&telebot.Chat{ID: chatID}, msg, telebot.ModeMarkdown, menu)
	if err != nil && isBlockedByUser(err) {
		slog.Warn("Usuário bloqueou o bot do Telegram ao receber proventos", "chatID", chatID, "error", err)
	}
	return err
}

func rateLimitMiddleware() telebot.MiddlewareFunc {
	return RateLimitMiddleware()
}
