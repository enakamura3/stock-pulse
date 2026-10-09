package telegram

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/redis/go-redis/v9"
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
	b.Use(TimeoutMiddleware(DefaultHandlerTimeout))

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

// StartWithLeaderElection inicia o bot gerenciado por eleição de liderança no cluster Redis.
func (r *BotRunner) StartWithLeaderElection(ctx context.Context, rdb *redis.Client, cfg ...LeaderElectorConfig) error {
	if r == nil || r.bot == nil {
		return nil
	}
	var c LeaderElectorConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	elector := NewLeaderElector(r, rdb, c)
	return elector.Run(ctx)
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

func (r *BotRunner) SendDailyDigest(
	chatID int64,
	userName, portfolioName, currency string,
	totalValue, dailyChange, dailyChangePercent, returnPercent float64,
	topGainers, topLosers []DailyMover,
	todayDividends, upcomingDividends []portfolio.CalculatedDividend,
) error {
	if r == nil || r.bot == nil {
		return nil
	}

	p := message.NewPrinter(language.Portuguese)
	escapedUserName := escapeMarkdown(userName)
	escapedPortfolioName := escapeMarkdown(portfolioName)

	msg := "🌆 *FECHAMENTO DE MERCADO: Seu Resumo Diário* 🌆\n\n"
	msg += "Boa noite, *" + escapedUserName + "*!\n"
	msg += "Aqui está o resumo de fechamento da sua carteira *" + escapedPortfolioName + "*:\n\n"

	msg += "💼 *Patrimônio:* " + currency + " " + formatFinancialPrice(p, totalValue) + "\n"

	badge := "⚪ "
	dailyPercentStr := p.Sprintf("%.2f%%", 0.0)
	if dailyChange > 1e-6 {
		badge = "🟢 +"
		dailyPercentStr = p.Sprintf("+%.2f%%", dailyChangePercent)
	} else if dailyChange < -1e-6 {
		badge = "🔴 -"
		dailyPercentStr = p.Sprintf("%.2f%%", dailyChangePercent)
	}
	msg += "📊 *Resultado de Hoje:* " + badge + currency + " " + formatFinancialPrice(p, math.Abs(dailyChange)) + " (" + dailyPercentStr + ")\n"

	retBadge := "⚪ "
	if returnPercent > 1e-6 {
		retBadge = "🟢 +"
	} else if returnPercent < -1e-6 {
		retBadge = "🔴 "
	}
	msg += "📈 *Rentabilidade Geral:* " + retBadge + p.Sprintf("%.2f%%", returnPercent) + "\n\n"

	// Destaques de mercado (Maiores Altas e Maiores Baixas)
	if len(topGainers) > 0 {
		msg += "🚀 *Maiores Altas:*\n"
		for _, g := range topGainers {
			msg += formatDailyMover(p, g, currency, true)
		}
		msg += "\n"
	}

	if len(topLosers) > 0 {
		msg += "🔻 *Maiores Baixas:*\n"
		for _, l := range topLosers {
			msg += formatDailyMover(p, l, currency, false)
		}
		msg += "\n"
	}

	// Proventos creditados hoje
	if len(todayDividends) > 0 {
		msg += "💰 *Proventos creditados hoje:*\n"
		for _, d := range todayDividends {
			msg += "  • *" + escapeMarkdown(d.Ticker) + "* (" + escapeMarkdown(d.Type) + "): " + d.Currency + " " + formatFinancialPrice(p, d.NetAmount) + "\n"
		}
		msg += "\n"
	}

	// Próximos proventos (7 dias)
	if len(upcomingDividends) > 0 {
		msg += "📅 *Próximos Proventos (7 dias):*\n"
		for _, d := range upcomingDividends {
			msg += "  • *" + escapeMarkdown(d.Ticker) + "*: " + d.PaymentDate.Format("02/01") + " — " + d.Currency + " " + formatFinancialPrice(p, d.NetAmount) + "\n"
		}
		msg += "\n"
	}

	msg += "Acompanhe seus investimentos em tempo real no *Stock Pulse*."

	menu := &telebot.ReplyMarkup{}
	btnResumo := menu.Data("💼 Ver Carteira", "btn_resumo")
	btnAgenda := menu.Data("📅 Minha Agenda", "btn_agenda")
	btnFavs := menu.Data("⭐ Meus Favoritos", "btn_watchlist")

	menu.Inline(menu.Row(btnResumo, btnAgenda), menu.Row(btnFavs))

	_, err := r.bot.Send(&telebot.Chat{ID: chatID}, msg, telebot.ModeMarkdown, menu)
	if err != nil && isBlockedByUser(err) {
		slog.Warn("Usuário bloqueou o bot do Telegram ao receber daily digest", "chatID", chatID, "error", err)
	}
	return err
}

func formatDailyMover(p *message.Printer, m DailyMover, defaultCurrency string, isGainer bool) string {
	if p == nil {
		p = message.NewPrinter(language.Portuguese)
	}
	sign := ""
	if isGainer {
		sign = "+"
	}
	line := "  • *" + escapeMarkdown(m.Ticker) + "*: " + p.Sprintf(sign+"%.2f%%", m.DailyChangePercent) + "\n"
	if m.CurrentPrice > 1e-6 {
		curr := m.Currency
		if curr == "" {
			curr = defaultCurrency
		}
		if m.OpenPrice > 1e-6 {
			line += "    Abert. " + curr + " " + formatFinancialPrice(p, m.OpenPrice) + " → Atual " + curr + " " + formatFinancialPrice(p, m.CurrentPrice) + "\n"
		} else {
			line += "    Atual " + curr + " " + formatFinancialPrice(p, m.CurrentPrice) + "\n"
		}
	}
	return line
}

func rateLimitMiddleware() telebot.MiddlewareFunc {
	return RateLimitMiddleware()
}
