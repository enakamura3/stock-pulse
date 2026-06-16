package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
	
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
)

type PortfolioService interface {
	GetPortfolios(ctx context.Context, userID string) ([]portfolio.Portfolio, error)
	GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*portfolio.Portfolio, []portfolio.Position, error)
	AddTransaction(ctx context.Context, userID string, tx *portfolio.Transaction) (*portfolio.Transaction, error)
	GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]portfolio.CalculatedDividend, error)
	GetPortfolioTransactions(ctx context.Context, portfolioID, userID string) ([]portfolio.Transaction, error)
}

type MarketService interface {
	GetQuote(ctx context.Context, ticker string) (*market.Quote, error)
}

type FixedIncomeService interface {
	GetPortfolioPositions(ctx context.Context, portfolioID string) ([]fixedincome.Position, error)
}

type Handlers struct {
	svc          Service
	portfolioSvc PortfolioService
	marketSvc    MarketService
	fiSvc        FixedIncomeService
}

func NewHandlers(svc Service, pSvc PortfolioService, mSvc MarketService, fiSvc FixedIncomeService) *Handlers {
	return &Handlers{
		svc:          svc,
		portfolioSvc: pSvc,
		marketSvc:    mSvc,
		fiSvc:        fiSvc,
	}
}

func (h *Handlers) Register(bot *telebot.Bot) {
	bot.Handle("/start", h.HandleStart)
	bot.Handle("/menu", h.HandleMenu)
	
	// Callback dos Inline Keyboards estáticos
	bot.Handle("\fbtn_resumo", h.HandlePortfolioSummary)
	bot.Handle("\fbtn_proventos", h.HandleDividends)
	bot.Handle("\fbtn_history", h.HandleHistory)
	bot.Handle("\fbtn_renda_fixa", h.HandleFixedIncome)
	bot.Handle("\fbtn_divs_year", h.HandleDividendsByYear)
	bot.Handle("\fbtn_divs_month", h.HandleDividendsByMonth)
	bot.Handle("\fbtn_operacao", h.HandleLaunchOperation)
	bot.Handle("\fbtn_change_portfolio", h.HandleChangePortfolio)

	bot.Handle("\fbtn_new_asset", h.HandleNewAsset)
	bot.Handle("\fbtn_buy", h.HandleSetTypeBuy)
	bot.Handle("\fbtn_sell", h.HandleSetTypeSell)

	// Intercepta todos os callbacks para capturar a seleção dinâmica de ticker e portfólio
	bot.Handle(telebot.OnCallback, h.HandleDynamicCallback)

	// Intercepta todas as mensagens de texto para a máquina de estados
	bot.Handle(telebot.OnText, h.HandleText)
}

func (h *Handlers) HandleStart(c telebot.Context) error {
	args := c.Args()
	if len(args) > 0 {
		token := args[0]
		err := h.svc.LinkAccountWithToken(context.Background(), token, c.Chat().ID)
		if err != nil {
			if strings.Contains(err.Error(), "inválido ou expirado") {
				return c.Send("❌ O link de vinculação é inválido ou expirou. Gere um novo no Stock Pulse.")
			}
			slog.Error("Erro ao vincular conta telegram", "error", err, "chat_id", c.Chat().ID)
			return c.Send("❌ Ocorreu um erro interno ao vincular sua conta. Tente novamente.")
		}
		return c.Send("✅ Conta vinculada com sucesso! Bem-vindo ao Stock Pulse.\n\nEnvie /menu para ver as opções.")
	}

	return c.Send("Bem-vindo ao bot do Stock Pulse! Para usar este bot, vá até as configurações no sistema web e clique em 'Vincular Telegram'.")
}

func (h *Handlers) HandleMenu(c telebot.Context) error {
	// Verifica se a conta está vinculada
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada. Gere um link no painel do Stock Pulse.")
	}
	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	menu := &telebot.ReplyMarkup{}
	btnResumo := menu.Data("📊 Resumo da Carteira", "btn_resumo")
	btnProventos := menu.Data("💸 Ver Proventos", "btn_proventos")
	btnHistory := menu.Data("📜 Histórico", "btn_history")
	btnRendaFixa := menu.Data("🏛️ Renda Fixa", "btn_renda_fixa")
	btnOperacao := menu.Data("💵 Lançar Operação", "btn_operacao")

	rows := []telebot.Row{
		menu.Row(btnResumo),
		menu.Row(btnProventos),
		menu.Row(btnHistory),
		menu.Row(btnRendaFixa),
		menu.Row(btnOperacao),
	}

	if len(portfolios) > 1 {
		btnTrocarCarteira := menu.Data("🔄 Trocar Carteira", "btn_change_portfolio")
		rows = append(rows, menu.Row(btnTrocarCarteira))
	}

	menu.Inline(rows...)

	_, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	return c.Send(fmt.Sprintf("🏢 *Carteira Ativa:* %s\nEscolha uma opção:", portfolioName), telebot.ModeMarkdown, menu)
}

func (h *Handlers) resolveActivePortfolio(ctx context.Context, chatID int64, portfolios []portfolio.Portfolio) (string, string) {
	if len(portfolios) == 0 {
		return "", ""
	}
	
	activeID, err := h.svc.GetActivePortfolio(ctx, chatID)
	if err == nil && activeID != "" {
		for _, p := range portfolios {
			if p.ID == activeID {
				return p.ID, p.Name
			}
		}
	}
	// Fallback para a primeira
	return portfolios[0].ID, portfolios[0].Name
}

func (h *Handlers) HandlePortfolioSummary(c telebot.Context) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	_, positions, err := h.portfolioSvc.GetPortfolioDetails(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch portfolio for telegram bot", "error", err, "user_id", userIDStr)
		return c.Send("❌ Ocorreu um erro ao buscar sua carteira.")
	}

	var totalValue, totalCost, totalDailyChange float64
	for _, pos := range positions {
		totalValue += pos.CurrentValue
		totalCost += pos.TotalCost
		// Simplificação: Assumimos que o CurrentValue embute a taxa de câmbio. 
		rate := 1.0
		if pos.CurrentPrice > 0 && pos.Quantity > 0 {
			rate = pos.CurrentValue / (pos.CurrentPrice * pos.Quantity)
		}
		totalDailyChange += pos.DailyChange * pos.Quantity * rate
	}

	// Integra Renda Fixa no Resumo
	var totalFIValue float64
	var nearMaturity []fixedincome.Position
	if h.fiSvc != nil {
		fiPos, err := h.fiSvc.GetPortfolioPositions(context.Background(), portfolioID)
		if err == nil {
			for _, pos := range fiPos {
				totalFIValue += pos.NetValue
				totalValue += pos.NetValue
				totalCost += pos.TotalInvested
				
				if pos.DaysToMaturity <= 30 && !pos.IsMatured {
					nearMaturity = append(nearMaturity, pos)
				}
			}
		}
	}

	totalProfitLoss := totalValue - totalCost
	totalReturnPercent := 0.0
	if totalCost > 0 {
		totalReturnPercent = (totalProfitLoss / totalCost) * 100
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📊 *Resumo: %s*\n\n", portfolioName)
	msg += p.Sprintf("💰 Valor Total: *R$ %.2f*\n", totalValue)
	
	var variacaoDiaria string
	if totalDailyChange >= 0 {
		variacaoDiaria = p.Sprintf("🟢 +R$ %.2f", totalDailyChange)
	} else {
		variacaoDiaria = p.Sprintf("🔴 R$ %.2f", totalDailyChange)
	}
	msg += p.Sprintf("📈 Variação Diária: *%s*\n", variacaoDiaria)
	
	var lucroPrejuizo string
	if totalProfitLoss >= 0 {
		lucroPrejuizo = p.Sprintf("🟢 +R$ %.2f (%.2f%%)", totalProfitLoss, totalReturnPercent)
	} else {
		lucroPrejuizo = p.Sprintf("🔴 R$ %.2f (%.2f%%)", totalProfitLoss, totalReturnPercent)
	}
	msg += p.Sprintf("⚖️ Lucro/Prejuízo Total: %s\n", lucroPrejuizo)
	
	if len(nearMaturity) > 0 {
		msg += "\n⚠️ *Vencimentos Próximos (Renda Fixa)*\n"
		for _, pos := range nearMaturity {
			msg += p.Sprintf("• `%s` (%s): Vence em %d dias\n", pos.Asset.Institution, pos.Asset.Type, pos.DaysToMaturity)
		}
	}

	// Clonar posições para ordenar sem afetar a original
	sortedPos := make([]portfolio.Position, 0, len(positions))
	for _, pos := range positions {
		if pos.DailyChangePercent != 0 {
			sortedPos = append(sortedPos, pos)
		}
	}

	sort.Slice(sortedPos, func(i, j int) bool {
		return sortedPos[i].DailyChangePercent > sortedPos[j].DailyChangePercent
	})

	var risers []portfolio.Position
	var fallers []portfolio.Position

	for _, pos := range sortedPos {
		if pos.DailyChangePercent > 0 {
			risers = append(risers, pos)
		}
	}
	for i := len(sortedPos) - 1; i >= 0; i-- {
		if sortedPos[i].DailyChangePercent < 0 {
			fallers = append(fallers, sortedPos[i])
		}
	}

	if len(risers) > 0 {
		msg += p.Sprintf("\n🚀 *Maiores Altas do Dia*\n")
		limit := 5
		if len(risers) < 5 {
			limit = len(risers)
		}
		for i := 0; i < limit; i++ {
			msg += p.Sprintf("• `%s`: +%.2f%%\n", risers[i].Ticker, risers[i].DailyChangePercent)
		}
	}

	if len(fallers) > 0 {
		msg += p.Sprintf("\n📉 *Maiores Baixas do Dia*\n")
		limit := 5
		if len(fallers) < 5 {
			limit = len(fallers)
		}
		for i := 0; i < limit; i++ {
			msg += p.Sprintf("• `%s`: %.2f%%\n", fallers[i].Ticker, fallers[i].DailyChangePercent)
		}
	}

	if len(sortedPos) > 0 {
		msg += p.Sprintf("\n📋 *Resumo Completo (Ativos)*\n")
		for _, pos := range sortedPos {
			var symbol string
			if pos.DailyChangePercent > 0 {
				symbol = "🟢"
			} else if pos.DailyChangePercent < 0 {
				symbol = "🔴"
			} else {
				symbol = "⚪"
			}
			rate := 1.0
			if pos.CurrentPrice > 0 && pos.Quantity > 0 {
				rate = pos.CurrentValue / (pos.CurrentPrice * pos.Quantity)
			}
			varBRL := pos.DailyChange * pos.Quantity * rate

			msg += p.Sprintf("%s `%s`: %+.2f%% (R$ %+.2f)\n", symbol, pos.Ticker, pos.DailyChangePercent, varBRL)
		}
	}

	// Acknowledge the callback to remove the loading state on the button
	c.Respond()
	
	return c.Send(msg, telebot.ModeMarkdown)
}

func getCurrencySymbol(code string) string {
	switch code {
	case "USD":
		return "US$"
	case "EUR":
		return "€"
	case "BRL":
		return "R$"
	default:
		return "R$"
	}
}

func abbreviateDividendType(t string) string {
	switch strings.ToUpper(t) {
	case "DIVIDENDO", "DIVIDENDOS", "DIV":
		return "DIV"
	case "JUROS SOBRE CAPITAL PRÓPRIO", "JUROS SOBRE CAPITAL PROPRIO", "JCP":
		return "JCP"
	case "RENDIMENTO", "RENDIMENTOS", "REND":
		return "REND"
	case "AMORTIZAÇÃO", "AMORTIZACAO":
		return "AMORT"
	default:
		if len(t) > 4 {
			return strings.ToUpper(t[:4])
		}
		return strings.ToUpper(t)
	}
}

func (h *Handlers) HandleLaunchOperation(c telebot.Context) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)

	_, positions, err := h.portfolioSvc.GetPortfolioDetails(context.Background(), portfolioID, userIDStr)
	if err != nil {
		return c.Send("❌ Ocorreu um erro ao buscar seus ativos.")
	}

	// Criar estado de conversa no Redis
	err = h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
		Step:        "EXPECT_TICKER",
		PortfolioID: portfolioID,
	})
	if err != nil {
		return c.Send("❌ Erro interno ao iniciar operação.")
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	// Adiciona botões para os ativos que o usuário já possui
	for _, pos := range positions {
		btn := menu.Data(fmt.Sprintf("%s (%s)", pos.Ticker, pos.Name), "btn_ticker_"+pos.Ticker)
		rows = append(rows, menu.Row(btn))
	}

	btnNew := menu.Data("➕ Novo Ativo", "btn_new_asset")
	rows = append(rows, menu.Row(btnNew))

	menu.Inline(rows...)

	c.Respond()
	return c.Send(fmt.Sprintf("🏢 *Carteira Ativa:* %s\nPara qual ativo deseja lançar a operação?", portfolioName), telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleDynamicCallback(c telebot.Context) error {
	data := c.Callback().Data
	data = strings.TrimPrefix(data, "\f")
	
	if strings.HasPrefix(data, "btn_ticker_") {
		ticker := strings.TrimPrefix(data, "btn_ticker_")
		return h.handleSelectedTicker(c, ticker)
	}

	if strings.HasPrefix(data, "btn_sel_port_") {
		portfolioID := strings.TrimPrefix(data, "btn_sel_port_")
		return h.handleSelectedPortfolio(c, portfolioID)
	}

	return nil
}

func (h *Handlers) HandleChangePortfolio(c telebot.Context) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, p := range portfolios {
		btn := menu.Data(fmt.Sprintf("📂 %s", p.Name), "btn_sel_port_"+p.ID)
		rows = append(rows, menu.Row(btn))
	}

	menu.Inline(rows...)
	c.Respond()
	return c.Send("Qual carteira você deseja definir como Ativa?", menu)
}

func (h *Handlers) handleSelectedPortfolio(c telebot.Context, portfolioID string) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil {
		return c.Send("❌ Erro ao buscar carteiras.")
	}

	var pName string
	for _, p := range portfolios {
		if p.ID == portfolioID {
			pName = p.Name
			break
		}
	}

	if pName == "" {
		return c.Send("❌ Carteira inválida.")
	}

	err = h.svc.SetActivePortfolio(context.Background(), c.Chat().ID, portfolioID)
	if err != nil {
		slog.Error("Failed to set active portfolio", "error", err)
		return c.Send("❌ Erro interno ao salvar carteira ativa.")
	}

	c.Respond()
	return c.Send(fmt.Sprintf("✅ Carteira *%s* selecionada como ativa!\n\nEnvie /menu para continuar.", pName), telebot.ModeMarkdown)
}

func (h *Handlers) handleSelectedTicker(c telebot.Context, ticker string) error {
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return c.Send("⚠️ Nenhuma operação em andamento. Envie /menu e clique em Lançar Operação.")
	}

	state.Ticker = ticker
	state.Step = "EXPECT_TYPE"
	_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

	menu := &telebot.ReplyMarkup{}
	btnBuy := menu.Data("🟢 Compra", "btn_buy")
	btnSell := menu.Data("🔴 Venda", "btn_sell")
	menu.Inline(menu.Row(btnBuy, btnSell))

	c.Respond()
	return c.Send(fmt.Sprintf("Operação para *%s*.\n\nÉ uma operação de *Compra* ou *Venda*?", ticker), telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleNewAsset(c telebot.Context) error {
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return c.Send("⚠️ Nenhuma operação em andamento. Envie /menu.")
	}

	c.Respond()
	return c.Send("Qual o código do ativo? (ex: AAPL, PETR4.SA)")
}

func (h *Handlers) HandleSetTypeBuy(c telebot.Context) error {
	return h.handleSetType(c, "BUY")
}

func (h *Handlers) HandleSetTypeSell(c telebot.Context) error {
	return h.handleSetType(c, "SELL")
}

func (h *Handlers) handleSetType(c telebot.Context, txType string) error {
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return c.Send("⚠️ Nenhuma operação em andamento. Envie /menu.")
	}

	state.Type = txType
	state.Step = "EXPECT_QTY"
	_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

	c.Respond()
	return c.Send("Qual a quantidade negociada? (ex: 10, 0.5)")
}

func (h *Handlers) HandleText(c telebot.Context) error {
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		// Se não estiver no meio de um wizard, responde com o menu principal para facilitar a usabilidade
		return h.HandleMenu(c)
	}

	text := strings.TrimSpace(c.Text())

	switch state.Step {
	case "EXPECT_TICKER":
		ticker := strings.ToUpper(text)
		// Validação Anti-Erro no Yahoo Finance / Mercado
		_, err := h.marketSvc.GetQuote(context.Background(), ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique se há erros de digitação e envie o código novamente:")
		}
		
		return h.handleSelectedTicker(c, ticker)

	case "EXPECT_QTY":
		// Permite uso de vírgula para floats (BR)
		text = strings.ReplaceAll(text, ",", ".")
		var qty float64
		if _, err := fmt.Sscanf(text, "%f", &qty); err != nil || qty <= 0 {
			return c.Send("⚠️ Quantidade inválida. Por favor, envie apenas o número (ex: 10):")
		}

		state.Quantity = qty
		state.Step = "EXPECT_PRICE"
		_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

		return c.Send("Qual o preço unitário da transação? (ex: 15.50)")

	case "EXPECT_PRICE":
		text = strings.ReplaceAll(text, ",", ".")
		var price float64
		if _, err := fmt.Sscanf(text, "%f", &price); err != nil || price <= 0 {
			return c.Send("⚠️ Preço inválido. Por favor, envie apenas o número (ex: 15.50):")
		}

		userID, _ := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)

		// Executar Transação
		tx := &portfolio.Transaction{
			PortfolioID:  state.PortfolioID,
			Ticker:       state.Ticker,
			Type:         state.Type,
			Quantity:     state.Quantity,
			UnitPrice:    price,
			TotalCost:    state.Quantity * price,
			ExchangeRate: 1.0,
			ExecutedAt:   time.Now(),
		}

		_, err = h.portfolioSvc.AddTransaction(context.Background(), userID.String(), tx)
		if err != nil {
			slog.Error("Erro ao lançar transação via telegram", "error", err)
			return c.Send("❌ Ocorreu um erro ao salvar a transação. Tente novamente mais tarde.")
		}

		// Limpa o estado
		_ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)

		tipoStr := "COMPRA"
		if state.Type == "SELL" {
			tipoStr = "VENDA"
		}
		
		p := message.NewPrinter(language.BrazilianPortuguese)
		successMsg := p.Sprintf("✅ *Operação Lançada com Sucesso!*\n\nAtivo: %s\nTipo: %s\nQuantidade: %.4f\nPreço Unitário: %.2f\nTotal: %.2f",
			state.Ticker, tipoStr, state.Quantity, price, tx.TotalCost)

		return c.Send(successMsg, telebot.ModeMarkdown)
	}

	return nil
}

func (h *Handlers) HandleDividends(c telebot.Context) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	divs, err := h.portfolioSvc.GetPortfolioDividends(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch dividends for telegram bot", "error", err, "user_id", userIDStr)
		return c.Send("❌ Ocorreu um erro ao buscar os proventos da sua carteira.")
	}

	var totalPaidMonth, totalFutureMonth float64
	now := time.Now()
	currentMonth := now.Month()
	currentYear := now.Year()
	
	for _, d := range divs {
		if d.PaymentDate.Year() == currentYear && d.PaymentDate.Month() == currentMonth {
			if d.PaymentDate.After(now) {
				totalFutureMonth += d.NetAmount
			} else {
				totalPaidMonth += d.NetAmount
			}
		}
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("💸 *Proventos: %s*\n\n", portfolioName)
	msg += p.Sprintf("✅ *Recebidos (Mês Atual):* R$ %.2f\n", totalPaidMonth)
	msg += p.Sprintf("⏳ *A Receber (Mês Atual):* R$ %.2f\n", totalFutureMonth)
	
	if len(divs) > 0 {
		var pastDivs []portfolio.CalculatedDividend
		var futureDivs []portfolio.CalculatedDividend
		for _, d := range divs {
			if !d.PaymentDate.After(now) {
				pastDivs = append(pastDivs, d)
			} else {
				futureDivs = append(futureDivs, d)
			}
		}

		if len(pastDivs) > 0 {
			msg += "\n📋 *Últimos Pagamentos*\n"
			
			// Ordernar pastDivs por data de pagamento decrescente
			sort.Slice(pastDivs, func(i, j int) bool {
				return pastDivs[i].PaymentDate.After(pastDivs[j].PaymentDate)
			})
			
			limit := 3
			if len(pastDivs) < 3 {
				limit = len(pastDivs)
			}
			
			sym := getCurrencySymbol("BRL")
			for i := 0; i < limit; i++ {
				d := pastDivs[i]
				tipoStr := "Div"
				if d.Type != "" {
					tipoStr = d.Type
				}
				msg += p.Sprintf("✅ `%s`: %s %.2f • %s • %s\n", d.Ticker, sym, d.NetAmount, d.PaymentDate.Format("2006-01-02"), abbreviateDividendType(tipoStr))
			}
		}

		if len(futureDivs) > 0 {
			msg += "\n📅 *Próximos Pagamentos*\n"
			
			// Ordernar futureDivs por data de pagamento crescente (mais próximos primeiro)
			sort.Slice(futureDivs, func(i, j int) bool {
				return futureDivs[i].PaymentDate.Before(futureDivs[j].PaymentDate)
			})
			
			limit := 3
			if len(futureDivs) < 3 {
				limit = len(futureDivs)
			}
			
			sym := getCurrencySymbol("BRL")
			for i := 0; i < limit; i++ {
				d := futureDivs[i]
				tipoStr := "Div"
				if d.Type != "" {
					tipoStr = d.Type
				}
				msg += p.Sprintf("⏳ `%s`: %s %.2f • %s • %s\n", d.Ticker, sym, d.NetAmount, d.PaymentDate.Format("2006-01-02"), abbreviateDividendType(tipoStr))
			}
		}
	} else {
		msg += "\nNenhum provento registrado na sua carteira ainda."
	}

	menu := &telebot.ReplyMarkup{}
	btnAno := menu.Data("📅 Agrupar por Ano", "btn_divs_year")
	btnMes := menu.Data("📆 Agrupar por Mês", "btn_divs_month")
	if len(divs) > 0 {
		menu.Inline(menu.Row(btnAno, btnMes))
	}

	c.Respond()
	if len(divs) > 0 {
		return c.Send(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown)
}

func (h *Handlers) fetchDividends(c telebot.Context) ([]portfolio.CalculatedDividend, string, error) {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return nil, "", fmt.Errorf("conta não vinculada")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return nil, "", fmt.Errorf("nenhuma carteira")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	divs, err := h.portfolioSvc.GetPortfolioDividends(context.Background(), portfolioID, userIDStr)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao buscar")
	}
	return divs, portfolioName, nil
}

func (h *Handlers) HandleDividendsByYear(c telebot.Context) error {
	divs, portfolioName, err := h.fetchDividends(c)
	if err != nil {
		return c.Send("❌ Erro ao buscar proventos.")
	}

	grouped := make(map[int]float64)
	for _, d := range divs {
		grouped[d.PaymentDate.Year()] += d.NetAmount
	}

	years := make([]int, 0, len(grouped))
	for y := range grouped {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📅 *Proventos por Ano: %s*\n\n", portfolioName)
	for _, y := range years {
		if y <= 1 {
			msg += p.Sprintf("• *A Definir*: R$ %.2f\n", grouped[y])
		} else {
			msg += p.Sprintf("• *%s*: R$ %.2f\n", fmt.Sprint(y), grouped[y])
		}
	}

	c.Respond()
	return c.Send(msg, telebot.ModeMarkdown)
}

func (h *Handlers) HandleDividendsByMonth(c telebot.Context) error {
	divs, portfolioName, err := h.fetchDividends(c)
	if err != nil {
		return c.Send("❌ Erro ao buscar proventos.")
	}

	grouped := make(map[string][]portfolio.CalculatedDividend)
	for _, d := range divs {
		key := d.PaymentDate.Format("2006-01") // Sortable key YYYY-MM
		if d.PaymentDate.IsZero() || d.PaymentDate.Year() <= 1 {
			key = "0000-00"
		}
		grouped[key] = append(grouped[key], d)
	}

	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	pageStr := c.Data()
	page := 0
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}

	pageSize := 3
	start := page * pageSize
	if start >= len(keys) {
		start = len(keys)
	}
	end := start + pageSize
	if end > len(keys) {
		end = len(keys)
	}

	if len(keys) == 0 {
		c.Respond()
		return c.Send("📆 Nenhum provento encontrado.")
	}

	pageKeys := keys[start:end]

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📆 *Proventos por Mês: %s*\n_Página %d_\n\n", portfolioName, page+1)
	
	sym := getCurrencySymbol("BRL")
	for _, k := range pageKeys {
		// Format back to YYYY/MM for display
		display := ""
		if k == "0000-00" {
			display = "A Definir"
		} else {
			parts := strings.Split(k, "-")
			display = fmt.Sprintf("%s/%s", parts[0], parts[1])
		}
		
		type tSummary struct {
			amount float64
			dates  []string
			dType  string
		}
		summaryMap := make(map[string]*tSummary)
		
		var totalMonth float64
		for _, d := range grouped[k] {
			totalMonth += d.NetAmount
			
			dType := "Div"
			if d.Type != "" {
				dType = d.Type
			}
			
			mapKey := d.Ticker + "|" + dType
			if _, exists := summaryMap[mapKey]; !exists {
				summaryMap[mapKey] = &tSummary{dType: dType}
			}
			summaryMap[mapKey].amount += d.NetAmount
			
			dateStr := d.PaymentDate.Format("2006-01-02")
			if d.PaymentDate.IsZero() || d.PaymentDate.Year() <= 1 {
				dateStr = "-"
			}
			foundDate := false
			for _, existing := range summaryMap[mapKey].dates {
				if existing == dateStr {
					foundDate = true
					break
				}
			}
			if !foundDate {
				summaryMap[mapKey].dates = append(summaryMap[mapKey].dates, dateStr)
			}
		}
		
		msg += p.Sprintf("• *%s*: R$ %.2f\n", display, totalMonth)
		
		// Sort mapKeys alphabetically
		mapKeys := make([]string, 0, len(summaryMap))
		for mk := range summaryMap {
			mapKeys = append(mapKeys, mk)
		}
		sort.Strings(mapKeys)
		
		for _, mk := range mapKeys {
			sum := summaryMap[mk]
			ticker := strings.Split(mk, "|")[0]
			datesStr := strings.Join(sum.dates, ", ")
			msg += p.Sprintf("   ↳ `%s`: %s %.2f • %s • %s\n", ticker, sym, sum.amount, datesStr, abbreviateDividendType(sum.dType))
		}
		msg += "\n"
	}

	menu := &telebot.ReplyMarkup{}
	var btns []telebot.Btn
	
	if start > 0 {
		btns = append(btns, menu.Data("⬅️ Anterior", "btn_divs_month", fmt.Sprintf("%d", page-1)))
	}
	if end < len(keys) {
		btns = append(btns, menu.Data("Próxima ➡️", "btn_divs_month", fmt.Sprintf("%d", page+1)))
	}

	if len(btns) > 0 {
		menu.Inline(menu.Row(btns...))
	}

	c.Respond()
	if pageStr != "" && c.Message() != nil && c.Message().Text != "" {
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleHistory(c telebot.Context) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)

	txs, err := h.portfolioSvc.GetPortfolioTransactions(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch transactions for telegram bot", "error", err, "user_id", userIDStr)
		return c.Send("❌ Ocorreu um erro ao buscar o histórico.")
	}

	pageStr := c.Data()
	page := 0
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}

	pageSize := 10
	start := page * pageSize
	if start >= len(txs) {
		start = len(txs)
	}
	end := start + pageSize
	if end > len(txs) {
		end = len(txs)
	}

	if len(txs) == 0 {
		c.Respond()
		return c.Send("📜 Nenhuma operação encontrada na sua carteira.")
	}

	pageTxs := txs[start:end]

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📜 *Histórico: %s*\n_Página %d_\n\n", portfolioName, page+1)

	for _, tx := range pageTxs {
		tipoStr := "🟢 C"
		if tx.Type == "SELL" {
			tipoStr = "🔴 V"
		}
		
		msg += p.Sprintf("%s | `%s`\n", tipoStr, tx.Ticker)
		msg += p.Sprintf("Data: %s\n", tx.ExecutedAt.Format("2006-01-02"))
		msg += p.Sprintf("Qtd: %.4f | Preço: %.2f | Total: %.2f\n\n", tx.Quantity, tx.UnitPrice, tx.TotalCost)
	}

	menu := &telebot.ReplyMarkup{}
	var btns []telebot.Btn
	
	if start > 0 {
		btns = append(btns, menu.Data("⬅️ Anterior", "btn_history", fmt.Sprintf("%d", page-1)))
	}
	if end < len(txs) {
		btns = append(btns, menu.Data("Próxima ➡️", "btn_history", fmt.Sprintf("%d", page+1)))
	}

	if len(btns) > 0 {
		menu.Inline(menu.Row(btns...))
	}

	c.Respond()
	
	if pageStr != "" && c.Message() != nil && c.Message().Text != "" {
		// It's a pagination click, update the existing message
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleFixedIncome(c telebot.Context) error {
	if h.fiSvc == nil {
		return c.Send("⚠️ Módulo de Renda Fixa não está ativo.")
	}

	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada.")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	positions, err := h.fiSvc.GetPortfolioPositions(context.Background(), portfolioID)
	if err != nil {
		return c.Send("❌ Erro ao buscar posições de Renda Fixa.")
	}

	if len(positions) == 0 {
		c.Respond()
		return c.Send("🏛️ Você ainda não possui ativos de Renda Fixa cadastrados.")
	}

	var totalBruto, totalLiquido, totalCusto float64
	for _, pos := range positions {
		totalBruto += pos.GrossValue
		totalLiquido += pos.NetValue
		totalCusto += pos.TotalInvested
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("🏛️ *Renda Fixa: %s*\n\n", portfolioName)
	msg += p.Sprintf("💰 Valor Líquido: *R$ %.2f*\n", totalLiquido)
	msg += p.Sprintf("📈 Valor Bruto: R$ %.2f\n", totalBruto)
	
	lucro := totalLiquido - totalCusto
	lucroPct := 0.0
	if totalCusto > 0 {
		lucroPct = (lucro / totalCusto) * 100
	}
	msg += p.Sprintf("⚖️ Lucro Líquido: R$ %.2f (%.2f%%)\n\n", lucro, lucroPct)
	
	msg += "*Minhas Posições:*\n"
	for _, pos := range positions {
		status := ""
		if pos.IsMatured {
			status = " *(VENCIDO)*"
		} else if pos.DaysToMaturity <= 30 {
			status = " *(Vence logo!)*"
		}
		
		taxa := ""
		if pos.Asset.DebtType == "POS" {
			taxa = fmt.Sprintf("%.2f%% %s", pos.Asset.Rate, pos.Asset.Indexer)
		} else {
			taxa = fmt.Sprintf("%.2f%% a.a.", pos.Asset.Rate)
		}
		
		msg += p.Sprintf("• `%s %s` - %s\n", pos.Asset.Institution, pos.Asset.Type, taxa)
		msg += p.Sprintf("  Líquido: R$ %.2f (+%.2f%%)%s\n", pos.NetValue, pos.NetReturnPercent, status)
	}

	c.Respond()
	return c.Send(msg, telebot.ModeMarkdown)
}
