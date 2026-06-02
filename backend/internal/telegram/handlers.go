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

type Handlers struct {
	svc          Service
	portfolioSvc PortfolioService
	marketSvc    MarketService
}

func NewHandlers(svc Service, pSvc PortfolioService, mSvc MarketService) *Handlers {
	return &Handlers{
		svc:          svc,
		portfolioSvc: pSvc,
		marketSvc:    mSvc,
	}
}

func (h *Handlers) Register(bot *telebot.Bot) {
	bot.Handle("/start", h.HandleStart)
	bot.Handle("/menu", h.HandleMenu)
	
	// Callback dos Inline Keyboards estáticos
	bot.Handle("\fbtn_resumo", h.HandlePortfolioSummary)
	bot.Handle("\fbtn_proventos", h.HandleDividends)
	bot.Handle("\fbtn_history", h.HandleHistory)
	bot.Handle("\fbtn_divs_year", h.HandleDividendsByYear)
	bot.Handle("\fbtn_divs_month", h.HandleDividendsByMonth)
	bot.Handle("\fbtn_operacao", h.HandleLaunchOperation)

	bot.Handle("\fbtn_new_asset", h.HandleNewAsset)
	bot.Handle("\fbtn_buy", h.HandleSetTypeBuy)
	bot.Handle("\fbtn_sell", h.HandleSetTypeSell)

	// Intercepta todos os callbacks para capturar a seleção dinâmica de ticker
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
	_, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada. Gere um link no painel do Stock Pulse.")
	}

	menu := &telebot.ReplyMarkup{}
	btnResumo := menu.Data("📊 Resumo da Carteira", "btn_resumo")
	btnProventos := menu.Data("💸 Ver Proventos", "btn_proventos")
	btnHistory := menu.Data("📜 Histórico", "btn_history")
	btnOperacao := menu.Data("💵 Lançar Operação", "btn_operacao")

	menu.Inline(
		menu.Row(btnResumo),
		menu.Row(btnProventos),
		menu.Row(btnHistory),
		menu.Row(btnOperacao),
	)

	return c.Send("Escolha uma opção:", menu)
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

	portfolioID := portfolios[0].ID
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
		// O DailyChange é na moeda original. Pra simplificar, vamos assumir que pos.DailyChange * pos.Quantity * taxa não temos a taxa exposta aqui na struct Position facilmente, mas podemos inferir ou apenas não somar.
		// Vamos estimar baseando em CurrentValue / CurrentPrice para a taxa
		rate := 1.0
		if pos.CurrentPrice > 0 && pos.Quantity > 0 {
			rate = pos.CurrentValue / (pos.CurrentPrice * pos.Quantity)
		}
		totalDailyChange += pos.DailyChange * pos.Quantity * rate
	}

	totalProfitLoss := totalValue - totalCost
	totalReturnPercent := 0.0
	if totalCost > 0 {
		totalReturnPercent = (totalProfitLoss / totalCost) * 100
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📊 *Resumo da sua Carteira*\n\n")
	msg += p.Sprintf("💰 Valor Total: *%.2f BRL*\n", totalValue)
	
	var variacaoDiaria string
	if totalDailyChange >= 0 {
		variacaoDiaria = p.Sprintf("🟢 +%.2f BRL", totalDailyChange)
	} else {
		variacaoDiaria = p.Sprintf("🔴 %.2f BRL", totalDailyChange)
	}
	msg += p.Sprintf("📈 Variação Diária: *%s*\n", variacaoDiaria)
	
	var lucroPrejuizo string
	if totalProfitLoss >= 0 {
		lucroPrejuizo = p.Sprintf("🟢 +%.2f BRL (%.2f%%)", totalProfitLoss, totalReturnPercent)
	} else {
		lucroPrejuizo = p.Sprintf("🔴 %.2f BRL (%.2f%%)", totalProfitLoss, totalReturnPercent)
	}
	msg += p.Sprintf("⚖️ Lucro/Prejuízo Total: %s\n", lucroPrejuizo)

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

			msg += p.Sprintf("%s `%s`: %+.2f%% (%+.2f BRL)\n", symbol, pos.Ticker, pos.DailyChangePercent, varBRL)
		}
	}

	// Acknowledge the callback to remove the loading state on the button
	c.Respond()
	
	return c.Send(msg, telebot.ModeMarkdown)
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
	portfolioID := portfolios[0].ID

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
	return c.Send("Para qual ativo deseja lançar a operação?", menu)
}

func (h *Handlers) HandleDynamicCallback(c telebot.Context) error {
	data := c.Callback().Data
	// data vem no formato "\fbtn_ticker_AAPL"
	data = strings.TrimPrefix(data, "\f")
	
	if strings.HasPrefix(data, "btn_ticker_") {
		ticker := strings.TrimPrefix(data, "btn_ticker_")
		return h.handleSelectedTicker(c, ticker)
	}

	// Como a gente registrou OnCallback globalmente, temos que dar Fallback
	// mas os botões estáticos têm preferência de rota do telebot, então não caem aqui a menos que falhe
	return nil
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

	portfolioID := portfolios[0].ID
	divs, err := h.portfolioSvc.GetPortfolioDividends(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch dividends for telegram bot", "error", err, "user_id", userIDStr)
		return c.Send("❌ Ocorreu um erro ao buscar os proventos da sua carteira.")
	}

	var totalPaid, totalFuture float64
	now := time.Now()
	
	for _, d := range divs {
		if d.PaymentDate.After(now) {
			totalFuture += d.NetAmount
		} else {
			totalPaid += d.NetAmount
		}
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("💸 *Resumo de Proventos*\n\n")
	msg += p.Sprintf("✅ *Recebidos:* %.2f BRL\n", totalPaid)
	msg += p.Sprintf("⏳ *A Receber:* %.2f BRL\n", totalFuture)
	
	if len(divs) > 0 {
		msg += "\n📋 *Últimos Pagamentos*\n"
		
		// Ordernar divs por data de pagamento decrescente
		sort.Slice(divs, func(i, j int) bool {
			return divs[i].PaymentDate.After(divs[j].PaymentDate)
		})
		
		limit := 5
		if len(divs) < 5 {
			limit = len(divs)
		}
		
		for i := 0; i < limit; i++ {
			d := divs[i]
			status := "✅"
			if d.PaymentDate.After(now) {
				status = "⏳"
			}
			msg += p.Sprintf("%s `%s`: %.2f BRL (%s)\n", status, d.Ticker, d.NetAmount, d.PaymentDate.Format("02/01/2006"))
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

func (h *Handlers) fetchDividends(c telebot.Context) ([]portfolio.CalculatedDividend, error) {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return nil, fmt.Errorf("conta não vinculada")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return nil, fmt.Errorf("nenhuma carteira")
	}

	portfolioID := portfolios[0].ID
	divs, err := h.portfolioSvc.GetPortfolioDividends(context.Background(), portfolioID, userIDStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar")
	}
	return divs, nil
}

func (h *Handlers) HandleDividendsByYear(c telebot.Context) error {
	divs, err := h.fetchDividends(c)
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
	msg := p.Sprintf("📅 *Proventos por Ano*\n\n")
	for _, y := range years {
		msg += p.Sprintf("• *%d*: %.2f BRL\n", y, grouped[y])
	}

	c.Respond()
	return c.Send(msg, telebot.ModeMarkdown)
}

func (h *Handlers) HandleDividendsByMonth(c telebot.Context) error {
	divs, err := h.fetchDividends(c)
	if err != nil {
		return c.Send("❌ Erro ao buscar proventos.")
	}

	grouped := make(map[string]map[string]float64)
	for _, d := range divs {
		key := d.PaymentDate.Format("2006-01") // Sortable key YYYY-MM
		if grouped[key] == nil {
			grouped[key] = make(map[string]float64)
		}
		grouped[key][d.Ticker] += d.NetAmount
	}

	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📆 *Proventos por Mês*\n\n")
	for _, k := range keys {
		// Format back to MM/YYYY for display
		parts := strings.Split(k, "-")
		display := fmt.Sprintf("%s/%s", parts[1], parts[0])
		
		var totalMonth float64
		for _, amt := range grouped[k] {
			totalMonth += amt
		}
		
		msg += p.Sprintf("• *%s*: %.2f BRL\n", display, totalMonth)
		
		// Sort the tickers inside the month to be deterministic
		tickers := make([]string, 0, len(grouped[k]))
		for t := range grouped[k] {
			tickers = append(tickers, t)
		}
		sort.Strings(tickers)
		
		for _, t := range tickers {
			msg += p.Sprintf("   ↳ `%s`: %.2f BRL\n", t, grouped[k][t])
		}
		msg += "\n"
	}

	c.Respond()
	return c.Send(msg, telebot.ModeMarkdown)
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
	portfolioID := portfolios[0].ID

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
	msg := p.Sprintf("📜 *Histórico de Operações*\n_Página %d_\n\n", page+1)

	for _, tx := range pageTxs {
		tipoStr := "🟢 C"
		if tx.Type == "SELL" {
			tipoStr = "🔴 V"
		}
		
		msg += p.Sprintf("%s | `%s`\n", tipoStr, tx.Ticker)
		msg += p.Sprintf("Data: %s\n", tx.ExecutedAt.Format("02/01/2006"))
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
