package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleCancelOperation(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()
	_ = h.svc.ClearConversationState(ctx, c.Chat().ID)
	return h.sendOrEditMenu(c)
}

func (h *Handlers) HandleOperationBack(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()
	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil {
		return h.sendOrEditMenu(c)
	}

	switch state.Step {
	case "EXPECT_TYPE":
		return h.HandleLaunchOperation(c)

	case "EXPECT_QTY":
		return h.handleSelectedTicker(c, state.Ticker)

	case "EXPECT_PRICE":
		return h.handleSetType(c, state.Type)

	case "EXPECT_DATE":
		state.UnitPrice = 0
		state.ExecutedAt = ""
		state.Fee = 0
		state.Step = "EXPECT_PRICE"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)
		return h.askPrice(c, state, c.Callback() != nil)

	case "EXPECT_FEE":
		state.ExecutedAt = ""
		state.Fee = 0
		state.Step = "EXPECT_DATE"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)
		return h.askDate(c, state, c.Callback() != nil)

	case "ALERT_EXPECT_COND":
		return h.HandleAlertCreate(c)

	case "ALERT_EXPECT_PRICE":
		state.Step = "ALERT_EXPECT_COND"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

		condMenu := &telebot.ReplyMarkup{}
		btnAbove := condMenu.Data("🟢 Acima de (>=)", "btn_alert_cond_above")
		btnBelow := condMenu.Data("🔴 Abaixo de (<=)", "btn_alert_cond_below")
		btnBack := condMenu.Data("⬅️ Voltar", "btn_op_back")
		btnCancelOp := condMenu.Data("❌ Cancelar", "btn_cancel_op")
		condMenu.Inline(
			condMenu.Row(btnAbove, btnBelow),
			condMenu.Row(btnBack, btnCancelOp),
		)

		text := fmt.Sprintf("🔔 *Alerta para %s*\n\nDisparar quando a cotação estiver:", escapeMarkdown(cleanTickerForDisplay(state.Ticker)))
		if c.Callback() != nil {
			return c.Edit(text, telebot.ModeMarkdown, condMenu)
		}
		return c.Send(text, telebot.ModeMarkdown, condMenu)

	default:
		return h.sendOrEditMenu(c)
	}
}

func formatOperationBreadcrumb(state *ConversationState, stepNum int) string {
	if state == nil {
		return ""
	}
	p := message.NewPrinter(language.BrazilianPortuguese)
	header := "🧾 *Nova Operação*"
	if state.PortfolioName != "" {
		header += fmt.Sprintf(" (%s)", escapeMarkdown(state.PortfolioName))
	}
	lines := []string{header}

	var line1 []string
	if state.Ticker != "" {
		line1 = append(line1, fmt.Sprintf("Ativo: `%s`", escapeMarkdown(cleanTickerForDisplay(state.Ticker))))
	}
	if stepNum >= 2 && state.Type != "" {
		tipoLabel := "🟢 Compra"
		if state.Type == "SELL" {
			tipoLabel = "🔴 Venda"
		}
		line1 = append(line1, "Tipo: "+tipoLabel)
	}
	if stepNum >= 3 && state.Quantity > 1e-6 {
		line1 = append(line1, fmt.Sprintf("Qtd: `%s`", formatQuantity(state.Quantity)))
	}
	if len(line1) > 0 {
		lines = append(lines, "• "+strings.Join(line1, " | "))
	}

	currSymbol := getTickerDefaultCurrency(state.Ticker)
	var line2 []string
	if stepNum >= 4 && state.UnitPrice > 1e-6 {
		subtotal := state.Quantity * state.UnitPrice
		line2 = append(line2, p.Sprintf("Preço: `%s %s` _(Subtotal: %s %s)_",
			currSymbol, formatUnitPrice(p, state.UnitPrice), currSymbol, formatFinancialPrice(p, subtotal)))
	}
	if stepNum >= 5 && state.ExecutedAt != "" {
		line2 = append(line2, fmt.Sprintf("Data: `%s`", state.ExecutedAt))
	}
	if len(line2) > 0 {
		lines = append(lines, "• "+strings.Join(line2, " | "))
	}

	return strings.Join(lines, "\n") + "\n\n"
}

func parseOperationDate(text string) (time.Time, error) {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)
	if lower == "hoje" || lower == "today" {
		return time.Now(), nil
	}
	if lower == "ontem" || lower == "yesterday" {
		return time.Now().AddDate(0, 0, -1), nil
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "02-01-2006"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func parseQuickOperationArgs(args []string) (string, string, float64, float64, string, float64, string) {
	var opType string
	switch strings.ToUpper(strings.TrimSpace(args[0])) {
	case "C", "COMPRA", "BUY":
		opType = "BUY"
	case "V", "VENDA", "SELL":
		opType = "SELL"
	default:
		return "", "", 0, 0, "", 0, "⚠️ Tipo inválido. Use `C` (Compra) ou `V` (Venda).\nEx: `/operacao C PETR4 100 36,50`"
	}

	ticker := strings.ToUpper(strings.TrimSpace(args[1]))

	var qty float64
	if _, err := fmt.Sscanf(strings.ReplaceAll(strings.TrimSpace(args[2]), ",", "."), "%f", &qty); err != nil || qty <= 1e-6 {
		return "", "", 0, 0, "", 0, "⚠️ Quantidade inválida. Informe um número maior que zero."
	}

	var price float64
	if _, err := fmt.Sscanf(strings.ReplaceAll(strings.TrimSpace(args[3]), ",", "."), "%f", &price); err != nil || price <= 1e-6 {
		return "", "", 0, 0, "", 0, "⚠️ Preço unitário inválido. Informe um número maior que zero."
	}

	dateStr := time.Now().Format("2006-01-02")
	if len(args) >= 5 {
		parsedDate, err := parseOperationDate(args[4])
		if err != nil {
			return "", "", 0, 0, "", 0, "⚠️ Data inválida. Use o formato `AAAA-MM-DD` (ex: `2026-03-15`)."
		}
		dateStr = parsedDate.Format("2006-01-02")
	}

	fee := 0.0
	if len(args) == 6 {
		if _, err := fmt.Sscanf(strings.ReplaceAll(strings.TrimSpace(args[5]), ",", "."), "%f", &fee); err != nil || fee < -1e-6 {
			return "", "", 0, 0, "", 0, "⚠️ Taxa inválida. Informe um valor positivo ou zero (ex: `4,50`)."
		}
		if fee < 1e-6 {
			fee = 0
		}
	}

	return opType, ticker, qty, price, dateStr, fee, ""
}

func (h *Handlers) HandleLaunchOperation(c telebot.Context) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		return h.sendOrEdit(c, "⚠️ Nenhuma carteira encontrada na sua conta.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(ctx, c.Chat().ID, portfolios)

	if c.Callback() == nil {
		if args := c.Args(); len(args) > 0 {
			return h.handleLaunchOperationArgs(ctx, c, userIDStr, portfolioID, portfolioName, args)
		}
	}

	_, positions, err := h.portfolioSvc.GetPortfolioDetails(ctx, portfolioID, userIDStr)
	if err != nil {
		return h.sendOrEdit(c, "❌ Ocorreu um erro ao buscar seus ativos.")
	}

	err = h.svc.SetConversationState(ctx, c.Chat().ID, ConversationState{
		Step:          "EXPECT_TICKER",
		PortfolioID:   portfolioID,
		PortfolioName: portfolioName,
	})
	if err != nil {
		return h.sendOrEdit(c, "❌ Erro interno ao iniciar operação.")
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	displayPositions := make([]portfolio.Position, len(positions))
	copy(displayPositions, positions)
	if len(displayPositions) > 12 {
		sort.Slice(displayPositions, func(i, j int) bool {
			return displayPositions[i].CurrentValue > displayPositions[j].CurrentValue
		})
		displayPositions = displayPositions[:12]
	}

	var assetBtns []telebot.Btn
	for _, pos := range displayPositions {
		tickerClean := cleanTickerForDisplay(pos.Ticker)
		label := tickerClean
		if pos.Name != "" && pos.Name != pos.Ticker && pos.Name != tickerClean {
			shortName := pos.Name
			if len(shortName) > 12 {
				shortName = shortName[:10] + ".."
			}
			label = fmt.Sprintf("%s (%s)", tickerClean, shortName)
		}
		btn := menu.Data(label, "btn_ticker_"+pos.Ticker)
		assetBtns = append(assetBtns, btn)
	}

	for i := 0; i < len(assetBtns); i += 2 {
		if i+1 < len(assetBtns) {
			rows = append(rows, menu.Row(assetBtns[i], assetBtns[i+1]))
		} else {
			rows = append(rows, menu.Row(assetBtns[i]))
		}
	}

	btnNew := menu.Data("➕ Novo Ativo", "btn_new_asset")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	rows = append(rows, menu.Row(btnNew, btnCancel))

	menu.Inline(rows...)

	return h.sendOrEdit(c, fmt.Sprintf("🏢 *Carteira Ativa:* %s\n\nEscolha um ativo abaixo ou *digite o ticker* no chat (ex: `PETR4`, `AAPL`):", escapeMarkdown(portfolioName)), telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleLaunchOperationArgs(ctx context.Context, c telebot.Context, userIDStr, portfolioID, portfolioName string, args []string) error {
	if len(args) == 1 {
		ticker := strings.ToUpper(strings.TrimSpace(args[0]))
		quote, err := h.getQuoteWithFallback(ctx, ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique o código e tente novamente (ex: `/operacao PETR4`).", telebot.ModeMarkdown)
		}
		resolvedTicker := ticker
		if quote != nil && quote.Ticker != "" {
			resolvedTicker = quote.Ticker
		}

		state := ConversationState{
			Step:          "EXPECT_TYPE",
			PortfolioID:   portfolioID,
			PortfolioName: portfolioName,
			Ticker:        resolvedTicker,
		}
		if err := h.svc.SetConversationState(ctx, c.Chat().ID, state); err != nil {
			return c.Send("❌ Erro interno ao iniciar operação.")
		}

		menu := &telebot.ReplyMarkup{}
		btnBuy := menu.Data("🟢 Compra", "btn_buy")
		btnSell := menu.Data("🔴 Venda", "btn_sell")
		btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
		btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
		menu.Inline(
			menu.Row(btnBuy, btnSell),
			menu.Row(btnBack, btnCancel),
		)

		text := formatOperationBreadcrumb(&state, 1) +
			fmt.Sprintf("📌 *Passo 1/5* — Você selecionou *%s*.\nQual o tipo da operação?", escapeMarkdown(cleanTickerForDisplay(resolvedTicker)))
		return h.sendOrEdit(c, text, telebot.ModeMarkdown, menu)
	}

	if len(args) < 4 || len(args) > 6 {
		usage := "⚠️ *Formato inválido.*\n\nUse:\n" +
			"• `/operacao` — assistente passo a passo\n" +
			"• `/operacao PETR4` — iniciar direto no ativo\n" +
			"• `/operacao C PETR4 100 36,50` — lançamento rápido\n" +
			"• `/operacao V PETR4 50 38,00 2026-03-15 4,50` — com data (`AAAA-MM-DD`) e taxa"
		return c.Send(usage, telebot.ModeMarkdown)
	}

	opType, ticker, qty, price, dateStr, fee, errMsg := parseQuickOperationArgs(args)
	if errMsg != "" {
		return c.Send(errMsg, telebot.ModeMarkdown)
	}

	quote, err := h.getQuoteWithFallback(ctx, ticker)
	if err != nil {
		return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique o código e tente novamente.", telebot.ModeMarkdown)
	}
	resolvedTicker := ticker
	if quote != nil && quote.Ticker != "" {
		resolvedTicker = quote.Ticker
	}

	if opType == "SELL" {
		pos, _ := h.getCustodyPosition(ctx, portfolioID, userIDStr, resolvedTicker)
		custodyQty := 0.0
		if pos != nil {
			custodyQty = pos.Quantity
		}
		if pos == nil || (qty-pos.Quantity) > 1e-6 {
			p := message.NewPrinter(language.BrazilianPortuguese)
			return c.Send(p.Sprintf("⚠️ *Quantidade indisponível para venda!*\n\nVocê possui apenas *%.4f* cotas de *%s* em custódia.", custodyQty, escapeMarkdown(cleanTickerForDisplay(resolvedTicker))), telebot.ModeMarkdown)
		}
	}

	state := ConversationState{
		Step:          "CONFIRM_OP",
		PortfolioID:   portfolioID,
		PortfolioName: portfolioName,
		Ticker:        resolvedTicker,
		Type:          opType,
		Quantity:      qty,
		UnitPrice:     price,
		ExecutedAt:    dateStr,
		Fee:           fee,
	}
	if err := h.svc.SetConversationState(ctx, c.Chat().ID, state); err != nil {
		return c.Send("❌ Erro interno ao iniciar operação.")
	}

	currSymbol := getTickerDefaultCurrency(resolvedTicker)
	if quote != nil && quote.Currency != "" {
		currSymbol = getCurrencySymbol(quote.Currency)
	}

	tipoStr := "🟢 COMPRA"
	totalCost := (qty * price) + fee
	if opType == "SELL" {
		tipoStr = "🔴 VENDA"
		totalCost = (qty * price) - fee
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("🧾 *Confirmar Operação*\n🏢 Carteira: *%s*\n\n• Ativo: `%s`\n• Tipo: %s\n• Quantidade: %s\n• Preço Unitário: %s %s\n• Data: `%s`\n• Taxas: %s %s\n• Total: *%s %s*\n\nDeseja confirmar o lançamento?",
		escapeMarkdown(portfolioName), escapeMarkdown(cleanTickerForDisplay(resolvedTicker)), tipoStr, formatQuantity(qty),
		currSymbol, formatUnitPrice(p, price), dateStr, currSymbol, formatFinancialPrice(p, fee), currSymbol, formatFinancialPrice(p, totalCost))

	menu := &telebot.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Confirmar Operação", "btn_op_confirm")
	btnEdit := menu.Data("✏️ Alterar Data/Taxa", "btn_op_edit_date")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(
		menu.Row(btnConfirm),
		menu.Row(btnEdit, btnCancel),
	)

	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleOperationConfirm(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "CONFIRM_OP" {
		return h.sendOrEdit(c, "⚠️ Nenhuma operação pendente de confirmação.")
	}

	return h.finalizeTransaction(c, state, state.Fee, c.Callback() != nil)
}

func (h *Handlers) HandleOperationEditDate(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "CONFIRM_OP" {
		return h.sendOrEdit(c, "⚠️ Nenhuma operação pendente de confirmação.")
	}

	state.ExecutedAt = ""
	state.Fee = 0
	state.Step = "EXPECT_DATE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askDate(c, state, c.Callback() != nil)
}

func (h *Handlers) HandleDynamicCallback(c telebot.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}

	data := cb.Data
	data = strings.TrimPrefix(data, "\f")

	if strings.HasPrefix(data, "btn_op_quick_") {
		ticker := strings.TrimPrefix(data, "btn_op_quick_")
		return h.handleQuickOperation(c, ticker)
	}

	if strings.HasPrefix(data, "btn_ticker_") {
		ticker := strings.TrimPrefix(data, "btn_ticker_")
		return h.handleSelectedTicker(c, ticker)
	}

	if strings.HasPrefix(data, "btn_sel_port_") {
		portfolioID := strings.TrimPrefix(data, "btn_sel_port_")
		return h.handleSelectedPortfolio(c, portfolioID)
	}

	if strings.HasPrefix(data, "btn_qty_") {
		qtyStr := strings.TrimPrefix(data, "btn_qty_")
		return h.handleSelectedQty(c, qtyStr)
	}

	if strings.HasPrefix(data, "btn_price_cur_") {
		priceStr := strings.TrimPrefix(data, "btn_price_cur_")
		return h.handleSelectedPrice(c, priceStr)
	}

	if strings.HasPrefix(data, "txda_") {
		payload := strings.TrimPrefix(data, "txda_")
		return h.handleAskDeleteTransaction(c, payload)
	}

	if strings.HasPrefix(data, "txdc_") {
		payload := strings.TrimPrefix(data, "txdc_")
		txID, returnData := parseTxDeleteCallback(payload)
		return h.handleDeleteTransactionWithReturn(c, txID, returnData)
	}

	if strings.HasPrefix(data, "btn_tx_del_") {
		txID := strings.TrimPrefix(data, "btn_tx_del_")
		return h.handleDeleteTransaction(c, txID)
	}

	if strings.HasPrefix(data, "btn_analise_") {
		ticker := strings.TrimPrefix(data, "btn_analise_")
		return h.renderAnalysis(c, ticker)
	}

	if strings.HasPrefix(data, "btn_quote_") {
		ticker := strings.TrimPrefix(data, "btn_quote_")
		return h.renderQuote(c, ticker)
	}

	if strings.HasPrefix(data, "btn_alert_create_") {
		ticker := strings.TrimPrefix(data, "btn_alert_create_")
		return h.handleQuickAlertCreate(c, ticker)
	}

	if strings.HasPrefix(data, "btn_alert_toggle_") {
		alertID := strings.TrimPrefix(data, "btn_alert_toggle_")
		return h.handleAlertToggle(c, alertID)
	}

	if strings.HasPrefix(data, "btn_alert_del_") {
		alertID := strings.TrimPrefix(data, "btn_alert_del_")
		return h.handleAlertDelete(c, alertID)
	}

	if strings.HasPrefix(data, "btn_alert_cond_") {
		condition := strings.TrimPrefix(data, "btn_alert_cond_")
		return h.handleAlertCondition(c, strings.ToUpper(condition))
	}

	if strings.HasPrefix(data, "btn_alert_price_") {
		priceStr := strings.TrimPrefix(data, "btn_alert_price_")
		return h.handleSelectedAlertPrice(c, priceStr)
	}

	if strings.HasPrefix(data, "btn_wl_tog_q_") {
		ticker := strings.TrimPrefix(data, "btn_wl_tog_q_")
		return h.handleQuickWatchlistToggle(c, "q", ticker)
	}

	if strings.HasPrefix(data, "btn_wl_tog_a_") {
		ticker := strings.TrimPrefix(data, "btn_wl_tog_a_")
		return h.handleQuickWatchlistToggle(c, "a", ticker)
	}

	if strings.HasPrefix(data, "btn_wl_add_") {
		ticker := strings.TrimPrefix(data, "btn_wl_add_")
		return h.handleQuickWatchlistAdd(c, ticker)
	}

	if strings.HasPrefix(data, "btn_wl_del_") {
		payload := strings.TrimPrefix(data, "btn_wl_del_")
		return h.handleWatchlistDelete(c, payload)
	}

	return nil
}

func (h *Handlers) handleQuickOperation(c telebot.Context, ticker string) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}
	ctx, cancel := h.getContext(c)
	defer cancel()

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		return h.sendOrEdit(c, "⚠️ Nenhuma carteira encontrada na sua conta.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(ctx, c.Chat().ID, portfolios)

	state := ConversationState{
		PortfolioID:   portfolioID,
		PortfolioName: portfolioName,
		Ticker:        ticker,
		Step:          "EXPECT_TYPE",
	}
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, state)

	menu := &telebot.ReplyMarkup{}
	btnBuy := menu.Data("🟢 Compra", "btn_buy")
	btnSell := menu.Data("🔴 Venda", "btn_sell")
	btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(
		menu.Row(btnBuy, btnSell),
		menu.Row(btnBack, btnCancel),
	)

	text := formatOperationBreadcrumb(&state, 1) +
		fmt.Sprintf("📌 *Passo 1/5* — Você selecionou *%s*.\nQual o tipo da operação?", escapeMarkdown(cleanTickerForDisplay(ticker)))
	return h.sendOrEdit(c, text, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleSelectedTicker(c telebot.Context, ticker string) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil {
		return c.Send("⚠️ Nenhuma operação em andamento. Envie /menu e clique em Lançar Operação.")
	}

	state.Ticker = ticker
	state.Type = ""
	state.Quantity = 0
	state.UnitPrice = 0
	state.ExecutedAt = ""
	state.Fee = 0
	state.Step = "EXPECT_TYPE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	menu := &telebot.ReplyMarkup{}
	btnBuy := menu.Data("🟢 Compra", "btn_buy")
	btnSell := menu.Data("🔴 Venda", "btn_sell")
	btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")

	menu.Inline(
		menu.Row(btnBuy, btnSell),
		menu.Row(btnBack, btnCancel),
	)

	text := formatOperationBreadcrumb(state, 1) +
		fmt.Sprintf("📌 *Passo 1/5* — Você selecionou *%s*.\nQual o tipo da operação?", escapeMarkdown(cleanTickerForDisplay(ticker)))
	return h.sendOrEdit(c, text, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleNewAsset(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnBack, btnCancel))

	return c.Edit("Qual o código do ativo? (ex: AAPL, PETR4)", menu)
}

func (h *Handlers) HandleSetTypeBuy(c telebot.Context) error {
	return h.handleSetType(c, "BUY")
}

func (h *Handlers) HandleSetTypeSell(c telebot.Context) error {
	return h.handleSetType(c, "SELL")
}

func (h *Handlers) handleSetType(c telebot.Context, opType string) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	state.Type = opType
	state.Quantity = 0
	state.UnitPrice = 0
	state.ExecutedAt = ""
	state.Fee = 0
	state.Step = "EXPECT_QTY"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")

	if opType == "SELL" {
		pos, _ := h.getCustodyPosition(ctx, state.PortfolioID, userIDStr, state.Ticker)
		if pos == nil || pos.Quantity <= 1e-6 {
			menu.Inline(menu.Row(btnBack, btnCancel))
			return c.Edit(fmt.Sprintf("⚠️ *Você não possui %s em custódia* nesta carteira para realizar uma venda.\n\nEscolha outro ativo ou selecione Compra.", escapeMarkdown(cleanTickerForDisplay(state.Ticker))), telebot.ModeMarkdown, menu)
		}

		p := message.NewPrinter(language.BrazilianPortuguese)
		currSymbol := getTickerDefaultCurrency(state.Ticker)
		btnAll := menu.Data(p.Sprintf("📦 Vender Tudo (%.0f)", pos.Quantity), "btn_qty_all")
		btnHalf := menu.Data("½ Metade", "btn_qty_half")

		menu.Inline(
			menu.Row(btnAll, btnHalf),
			menu.Row(btnBack, btnCancel),
		)

		text := formatOperationBreadcrumb(state, 2) +
			p.Sprintf("📊 *Posição em Custódia:* %.4f cotas (PM: %s %s)\n\n📦 *Passo 2/5* — Qual a quantidade que deseja vender? (Escolha abaixo ou digite o valor no chat):",
				pos.Quantity, currSymbol, formatUnitPrice(p, pos.AveragePrice))
		return c.Edit(text, telebot.ModeMarkdown, menu)
	}

	b1 := menu.Data("1", "btn_qty_1")
	b5 := menu.Data("5", "btn_qty_5")
	b10 := menu.Data("10", "btn_qty_10")
	b50 := menu.Data("50", "btn_qty_50")
	b100 := menu.Data("100", "btn_qty_100")

	menu.Inline(
		menu.Row(b1, b5, b10),
		menu.Row(b50, b100),
		menu.Row(btnBack, btnCancel),
	)

	text := formatOperationBreadcrumb(state, 2) + "📦 *Passo 2/5* — Qual a quantidade negociada? (Escolha abaixo ou digite o valor no chat):"
	return c.Edit(text, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleQtyAll(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "EXPECT_QTY" {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	pos, err := h.getCustodyPosition(ctx, state.PortfolioID, userIDStr, state.Ticker)
	if err != nil || pos == nil || pos.Quantity <= 1e-6 {
		return c.Edit(fmt.Sprintf("⚠️ Você não possui custódia de %s para vender.", escapeMarkdown(cleanTickerForDisplay(state.Ticker))))
	}

	state.Quantity = pos.Quantity
	state.Step = "EXPECT_PRICE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askPrice(c, state, true)
}

func (h *Handlers) HandleQtyHalf(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "EXPECT_QTY" {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	pos, err := h.getCustodyPosition(ctx, state.PortfolioID, userIDStr, state.Ticker)
	if err != nil || pos == nil || pos.Quantity <= 1e-6 {
		return c.Edit(fmt.Sprintf("⚠️ Você não possui custódia de %s para vender.", escapeMarkdown(cleanTickerForDisplay(state.Ticker))))
	}

	state.Quantity = pos.Quantity / 2.0
	state.Step = "EXPECT_PRICE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askPrice(c, state, true)
}

func (h *Handlers) handleSelectedQty(c telebot.Context, qtyStr string) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	var qty float64
	fmt.Sscanf(qtyStr, "%f", &qty)
	if qty <= 1e-6 {
		return c.Edit("⚠️ Quantidade inválida.")
	}

	if state.Type == "SELL" {
		userIDStr, err := h.getUserID(c)
		if err != nil {
			return err
		}
		pos, _ := h.getCustodyPosition(ctx, state.PortfolioID, userIDStr, state.Ticker)
		custodyQty := 0.0
		if pos != nil {
			custodyQty = pos.Quantity
		}
		if pos == nil || (qty-pos.Quantity) > 1e-6 {
			menu := &telebot.ReplyMarkup{}
			btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
			btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
			menu.Inline(menu.Row(btnBack, btnCancel))
			p := message.NewPrinter(language.BrazilianPortuguese)
			return c.Edit(p.Sprintf("⚠️ *Quantidade indisponível para venda!*\n\nVocê possui apenas *%.4f* cotas de *%s* em custódia.\nPor favor, informe uma quantidade menor ou igual:", custodyQty, escapeMarkdown(cleanTickerForDisplay(state.Ticker))), telebot.ModeMarkdown, menu)
		}
	}

	state.Quantity = qty
	state.Step = "EXPECT_PRICE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askPrice(c, state, true)
}

func (h *Handlers) askPrice(c telebot.Context, state *ConversationState, isCallback bool) error {
	ctx, cancel := h.getContext(c)
	defer cancel()

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	currSymbol := getTickerDefaultCurrency(state.Ticker)
	quote, err := h.getQuoteWithFallback(ctx, state.Ticker)
	if err == nil && quote != nil && quote.Price > 1e-6 {
		p := message.NewPrinter(language.BrazilianPortuguese)
		if quote.Currency != "" {
			currSymbol = getCurrencySymbol(quote.Currency)
		}
		btnCurPrice := menu.Data(p.Sprintf("💵 Usar Preço Atual (%s %.2f)", currSymbol, quote.Price), fmt.Sprintf("btn_price_cur_%.2f", quote.Price))
		rows = append(rows, menu.Row(btnCurPrice))
	}

	btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	rows = append(rows, menu.Row(btnBack, btnCancel))
	menu.Inline(rows...)

	text := formatOperationBreadcrumb(state, 3) + "💵 *Passo 3/5* — Qual o preço unitário da transação? (ex: 15.50)"
	if isCallback {
		return c.Edit(text, telebot.ModeMarkdown, menu)
	}
	return c.Send(text, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleSelectedPrice(c telebot.Context, priceStr string) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "EXPECT_PRICE" {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	var price float64
	if _, err := fmt.Sscanf(priceStr, "%f", &price); err != nil || price <= 1e-6 {
		return c.Edit("⚠️ Preço inválido.")
	}

	state.UnitPrice = price
	state.Step = "EXPECT_DATE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askDate(c, state, true)
}

func (h *Handlers) askDate(c telebot.Context, state *ConversationState, isCallback bool) error {
	dateMenu := &telebot.ReplyMarkup{}
	btnToday := dateMenu.Data("📅 Hoje", "btn_op_date_today")
	btnYesterday := dateMenu.Data("⏮️ Ontem", "btn_op_date_yesterday")
	btnBack := dateMenu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := dateMenu.Data("❌ Cancelar", "btn_cancel_op")
	dateMenu.Inline(
		dateMenu.Row(btnToday, btnYesterday),
		dateMenu.Row(btnBack, btnCancel),
	)

	text := formatOperationBreadcrumb(state, 4) +
		"📅 *Passo 4/5 — Data da Operação*\n\nClique em *Hoje*, *Ontem* ou digite a data no formato `AAAA-MM-DD` (ex: `2026-03-15`):"
	if isCallback {
		return c.Edit(text, telebot.ModeMarkdown, dateMenu)
	}
	return c.Send(text, telebot.ModeMarkdown, dateMenu)
}

func (h *Handlers) HandleDateToday(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "EXPECT_DATE" {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	state.ExecutedAt = time.Now().Format("2006-01-02")
	state.Step = "EXPECT_FEE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askFee(c, state, true)
}

func (h *Handlers) HandleDateYesterday(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "EXPECT_DATE" {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	state.ExecutedAt = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	state.Step = "EXPECT_FEE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	return h.askFee(c, state, true)
}

func (h *Handlers) HandleFeeZero(c telebot.Context) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "EXPECT_FEE" {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	state.Fee = 0
	return h.finalizeTransaction(c, state, 0, true)
}

func (h *Handlers) askFee(c telebot.Context, state *ConversationState, isCallback bool) error {
	currSymbol := getTickerDefaultCurrency(state.Ticker)

	feeMenu := &telebot.ReplyMarkup{}
	btnNoFee := feeMenu.Data("0️⃣ Sem Taxas", "btn_op_fee_zero")
	btnBack := feeMenu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := feeMenu.Data("❌ Cancelar", "btn_cancel_op")
	feeMenu.Inline(
		feeMenu.Row(btnNoFee),
		feeMenu.Row(btnBack, btnCancel),
	)

	text := formatOperationBreadcrumb(state, 5) +
		fmt.Sprintf("💵 *Passo 5/5 — Taxas / Corretagem*\n\nInforme o valor total de taxas em %s (ex: `4.50`) ou clique em *Sem Taxas*:", currSymbol)
	if isCallback {
		return c.Edit(text, telebot.ModeMarkdown, feeMenu)
	}
	return c.Send(text, telebot.ModeMarkdown, feeMenu)
}

func (h *Handlers) finalizeTransaction(c telebot.Context, state *ConversationState, fee float64, isCallback bool) error {
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	executedAt := time.Now()
	if state.ExecutedAt != "" {
		if parsed, parseErr := time.Parse("2006-01-02", state.ExecutedAt); parseErr == nil {
			executedAt = parsed
		}
	}

	var totalCost float64
	if state.Type == "SELL" {
		totalCost = (state.Quantity * state.UnitPrice) - fee
	} else {
		totalCost = (state.Quantity * state.UnitPrice) + fee
	}

	tx := &portfolio.Transaction{
		PortfolioID:  state.PortfolioID,
		Ticker:       state.Ticker,
		Type:         state.Type,
		Quantity:     state.Quantity,
		UnitPrice:    state.UnitPrice,
		TotalCost:    totalCost,
		Fee:          fee,
		ExchangeRate: 0,
		ExecutedAt:   executedAt,
	}

	savedTx, err := h.portfolioSvc.AddTransaction(ctx, userIDStr, tx)
	if err != nil {
		slog.Error("Erro ao lançar transação via telegram", "error", err)
		errMsg := "❌ Ocorreu um erro ao salvar a transação. Tente novamente mais tarde."
		if isCallback {
			return c.Edit(errMsg)
		}
		return c.Send(errMsg)
	}

	_ = h.svc.ClearConversationState(ctx, c.Chat().ID)

	currSymbol := getTickerDefaultCurrency(state.Ticker)
	if savedTx != nil && savedTx.Currency != "" {
		currSymbol = getCurrencySymbol(savedTx.Currency)
	}

	tipoStr := "COMPRA"
	if state.Type == "SELL" {
		tipoStr = "VENDA"
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	successMsg := p.Sprintf("✅ *Operação Lançada com Sucesso!*\n\n• Ativo: `%s`\n• Tipo: %s\n• Quantidade: %.4f\n• Preço Unitário: %s %s\n• Taxas: %s %.2f\n• Data: %s\n• Total: %s %.2f",
		cleanTickerForDisplay(state.Ticker), tipoStr, state.Quantity, currSymbol, formatUnitPrice(p, state.UnitPrice), currSymbol, fee, executedAt.Format("02/01/2006"), currSymbol, totalCost)

	if pos, _ := h.getCustodyPosition(ctx, state.PortfolioID, userIDStr, state.Ticker); pos != nil && pos.Quantity > 1e-6 {
		avgPrice := pos.AveragePrice
		if avgPrice <= 1e-6 && pos.TotalCost > 1e-6 {
			avgPrice = pos.TotalCost / pos.Quantity
		}
		if avgPrice > 1e-6 {
			successMsg += p.Sprintf("\n\n📊 *Posição Atualizada:* %s cotas (PM: %s %s)",
				formatQuantity(pos.Quantity), currSymbol, formatUnitPrice(p, avgPrice))
		}
	}

	successMenu := &telebot.ReplyMarkup{}
	var rows []telebot.Row
	if savedTx != nil && savedTx.ID != "" {
		btnUndo := successMenu.Data("🗑️ Desfazer Operação", "txda_"+savedTx.ID)
		rows = append(rows, successMenu.Row(btnUndo))
	}
	btnNewOp := successMenu.Data("➕ Nova Operação", "btn_operacao")
	btnMenu := successMenu.Data("🏠 Voltar ao Menu", "btn_menu")
	rows = append(rows, successMenu.Row(btnNewOp, btnMenu))
	successMenu.Inline(rows...)

	if isCallback {
		return c.Edit(successMsg, telebot.ModeMarkdown, successMenu)
	}
	return c.Send(successMsg, telebot.ModeMarkdown, successMenu)
}

func (h *Handlers) HandleUndoLastOperation(c telebot.Context) error {
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada. Crie uma carteira primeiro na plataforma web.")
	}

	portID, portName := h.resolveActivePortfolio(ctx, c.Chat().ID, portfolios)

	txs, err := h.portfolioSvc.GetPortfolioTransactions(ctx, portID, userIDStr)
	if err != nil || len(txs) == 0 {
		return c.Send("ℹ️ Nenhuma operação recente encontrada para desfazer nesta carteira.")
	}

	lastTx := txs[0]
	tipoStr := "COMPRA"
	if lastTx.Type == "SELL" {
		tipoStr = "VENDA"
	}

	currSymbol := "R$"
	if lastTx.Currency != "" {
		currSymbol = getCurrencySymbol(lastTx.Currency)
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("⚠️ *Confirmar cancelamento da última operação:*\n🏢 Carteira: *%s*\n\n• Ativo: `%s`\n• Tipo: %s\n• Quantidade: %.4f\n• Preço: %s %s\n• Taxas: %s %.2f\n• Data: %s\n• Total: %s %.2f\n\nDeseja realmente excluir esta operação?",
		escapeMarkdown(portName), cleanTickerForDisplay(lastTx.Ticker), tipoStr, lastTx.Quantity, currSymbol, formatUnitPrice(p, lastTx.UnitPrice), currSymbol, lastTx.Fee, lastTx.ExecutedAt.Format("02/01/2006"), currSymbol, lastTx.TotalCost)

	menu := &telebot.ReplyMarkup{}
	btnConfirm := menu.Data("🗑️ Sim, Desfazer", "btn_tx_del_"+lastTx.ID)
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnConfirm, btnCancel))

	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleAskDeleteTransaction(c telebot.Context, payload string) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Carteira não encontrada.")
	}

	portID, portName := h.resolveActivePortfolio(ctx, c.Chat().ID, portfolios)
	txID, returnData := parseTxDeleteCallback(payload)

	txs, err := h.portfolioSvc.GetPortfolioTransactions(ctx, portID, userIDStr)
	var targetTx *portfolio.Transaction
	if err == nil {
		for i := range txs {
			if txs[i].ID == txID {
				targetTx = &txs[i]
				break
			}
		}
	}

	if targetTx == nil {
		menu := &telebot.ReplyMarkup{}
		if returnData != "" {
			btnBack := menu.Data("📜 Voltar ao Histórico", "btn_history", returnData)
			btnMenu := menu.Data("🏠 Menu", "btn_menu")
			menu.Inline(menu.Row(btnBack, btnMenu))
		} else {
			btnMenu := menu.Data("🏠 Voltar ao Menu", "btn_menu")
			menu.Inline(menu.Row(btnMenu))
		}
		return c.Edit("⚠️ Operação não encontrada ou já foi removida.", menu)
	}

	tipoStr := "COMPRA"
	if targetTx.Type == "SELL" {
		tipoStr = "VENDA"
	}

	currSymbol := getTickerDefaultCurrency(targetTx.Ticker)
	if targetTx.Currency != "" {
		currSymbol = getCurrencySymbol(targetTx.Currency)
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("⚠️ *Confirmar exclusão de operação:*\n🏢 Carteira: *%s*\n\n• Ativo: `%s`\n• Tipo: %s\n• Quantidade: %s\n• Preço: %s %s\n• Taxas: %s %s\n• Data: %s\n• Total: %s %s\n\nDeseja realmente excluir esta operação?",
		escapeMarkdown(portName), cleanTickerForDisplay(targetTx.Ticker), tipoStr, formatQuantity(targetTx.Quantity),
		currSymbol, formatUnitPrice(p, targetTx.UnitPrice), currSymbol, formatFinancialPrice(p, targetTx.Fee),
		targetTx.ExecutedAt.Format("02/01/2006"), currSymbol, formatFinancialPrice(p, targetTx.TotalCost))

	menu := &telebot.ReplyMarkup{}
	if returnData != "" {
		btnConfirm := menu.Data("🗑️ Sim, Excluir", "txdc_"+payload)
		btnBack := menu.Data("⬅️ Voltar ao Histórico", "btn_history", returnData)
		menu.Inline(menu.Row(btnConfirm, btnBack))
	} else {
		btnConfirm := menu.Data("🗑️ Sim, Excluir", "btn_tx_del_"+txID)
		btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
		menu.Inline(menu.Row(btnConfirm, btnCancel))
	}

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleDeleteTransaction(c telebot.Context, txID string) error {
	return h.handleDeleteTransactionWithReturn(c, txID, "")
}

func (h *Handlers) handleDeleteTransactionWithReturn(c telebot.Context, txID, returnData string) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Carteira não encontrada.")
	}

	portID, _ := h.resolveActivePortfolio(ctx, c.Chat().ID, portfolios)

	err = h.portfolioSvc.DeleteTransaction(ctx, txID, portID, userIDStr)
	if err != nil {
		slog.Error("Erro ao excluir transação via telegram", "txID", txID, "error", err)
		return c.Edit("❌ Erro ao excluir a operação. Ela pode já ter sido removida.")
	}

	menu := &telebot.ReplyMarkup{}
	btnMenu := menu.Data("🏠 Voltar ao Menu", "btn_menu")
	if returnData != "" {
		btnHist := menu.Data("📜 Voltar ao Histórico", "btn_history", returnData)
		menu.Inline(menu.Row(btnHist, btnMenu))
	} else {
		btnNewOp := menu.Data("➕ Nova Operação", "btn_operacao")
		menu.Inline(menu.Row(btnNewOp, btnMenu))
	}

	return c.Edit("🗑️ *Operação desfeita com sucesso!*\nA transação foi removida da sua carteira.", telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleText(c telebot.Context) error {
	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil {
		if tickers := h.tryResolveFreeTextTickers(ctx, c, c.Text()); len(tickers) > 0 {
			if len(tickers) == 1 {
				return h.renderQuote(c, tickers[0])
			}
			return h.renderMultiQuote(c, tickers)
		}

		menu := &telebot.ReplyMarkup{}
		btnMenu := menu.Data("🏠 Abrir Menu Principal", "btn_menu")
		menu.Inline(menu.Row(btnMenu))
		msg := "🤖 *Olá!*\n\nNão entendi esse comando ou mensagem solta.\n\n" +
			"💡 *Você pode:*\n" +
			"• Digitar um ativo direto no chat (ex: `PETR4`, `BTC-USD`, `$AAPL`)\n" +
			"• Usar o botão abaixo para abrir o menu interativo\n" +
			"• Fazer análise fundamentalista: `/analise WEGE3`\n" +
			"• Ver todos os comandos disponíveis: `/help`"
		return c.Send(msg, telebot.ModeMarkdown, menu)
	}

	text := strings.TrimSpace(c.Text())
	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	switch state.Step {
	case "EXPECT_TICKER":
		ticker := strings.ToUpper(text)
		quote, err := h.getQuoteWithFallback(ctx, ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique se há erros de digitação e envie o código novamente:", menu)
		}
		resolvedTicker := ticker
		if quote != nil && quote.Ticker != "" {
			resolvedTicker = quote.Ticker
		}

		return h.handleSelectedTicker(c, resolvedTicker)

	case "EXPECT_QTY":
		text = strings.ReplaceAll(text, ",", ".")
		var qty float64
		if _, err := fmt.Sscanf(text, "%f", &qty); err != nil || qty <= 1e-6 {
			return c.Send("⚠️ Quantidade inválida. Por favor, envie apenas o número (ex: 10):", menu)
		}

		if state.Type == "SELL" {
			userIDStr, err := h.getUserID(c)
			if err != nil {
				return err
			}
			pos, _ := h.getCustodyPosition(ctx, state.PortfolioID, userIDStr, state.Ticker)
			custodyQty := 0.0
			if pos != nil {
				custodyQty = pos.Quantity
			}
			if pos == nil || (qty-pos.Quantity) > 1e-6 {
				p := message.NewPrinter(language.BrazilianPortuguese)
				return c.Send(p.Sprintf("⚠️ *Quantidade indisponível para venda!*\n\nVocê possui apenas *%.4f* cotas de *%s* em custódia.\nPor favor, informe uma quantidade menor ou igual:", custodyQty, escapeMarkdown(cleanTickerForDisplay(state.Ticker))), telebot.ModeMarkdown, menu)
			}
		}

		state.Quantity = qty
		state.Step = "EXPECT_PRICE"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

		return h.askPrice(c, state, false)

	case "EXPECT_PRICE":
		text = strings.ReplaceAll(text, ",", ".")
		var price float64
		if _, err := fmt.Sscanf(text, "%f", &price); err != nil || price <= 1e-6 {
			priceErrMenu := &telebot.ReplyMarkup{}
			btnBack := priceErrMenu.Data("⬅️ Voltar", "btn_op_back")
			btnCancelOp := priceErrMenu.Data("❌ Cancelar", "btn_cancel_op")
			priceErrMenu.Inline(priceErrMenu.Row(btnBack, btnCancelOp))
			return c.Send("⚠️ Preço inválido. Por favor, envie apenas o número (ex: 15.50):", priceErrMenu)
		}

		state.UnitPrice = price
		state.Step = "EXPECT_DATE"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

		return h.askDate(c, state, false)

	case "EXPECT_DATE":
		opDate, parseErr := parseOperationDate(text)
		if parseErr != nil {
			dateMenu := &telebot.ReplyMarkup{}
			btnToday := dateMenu.Data("📅 Hoje", "btn_op_date_today")
			btnYesterday := dateMenu.Data("⏮️ Ontem", "btn_op_date_yesterday")
			btnBack := dateMenu.Data("⬅️ Voltar", "btn_op_back")
			btnCancelOp := dateMenu.Data("❌ Cancelar", "btn_cancel_op")
			dateMenu.Inline(
				dateMenu.Row(btnToday, btnYesterday),
				dateMenu.Row(btnBack, btnCancelOp),
			)
			return c.Send("⚠️ Formato de data inválido. Use `AAAA-MM-DD` (ex: `2026-03-15`) ou clique em *Hoje* / *Ontem*:", telebot.ModeMarkdown, dateMenu)
		}

		state.ExecutedAt = opDate.Format("2006-01-02")
		state.Step = "EXPECT_FEE"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

		return h.askFee(c, state, false)

	case "EXPECT_FEE":
		textClean := strings.ReplaceAll(text, ",", ".")
		var fee float64
		if _, err := fmt.Sscanf(textClean, "%f", &fee); err != nil || fee < -1e-6 {
			feeMenu := &telebot.ReplyMarkup{}
			btnNoFee := feeMenu.Data("0️⃣ Sem Taxas", "btn_op_fee_zero")
			btnBack := feeMenu.Data("⬅️ Voltar", "btn_op_back")
			btnCancelOp := feeMenu.Data("❌ Cancelar", "btn_cancel_op")
			feeMenu.Inline(
				feeMenu.Row(btnNoFee),
				feeMenu.Row(btnBack, btnCancelOp),
			)
			return c.Send("⚠️ Valor de taxa inválido. Envie um número positivo (ex: 4.50) ou clique em *Sem Taxas*:", telebot.ModeMarkdown, feeMenu)
		}
		if fee < 1e-6 {
			fee = 0
		}

		state.Fee = fee
		return h.finalizeTransaction(c, state, fee, false)

	case "ALERT_EXPECT_TICKER":
		ticker := strings.ToUpper(text)
		quote, err := h.getQuoteWithFallback(ctx, ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique se há erros de digitação e envie o código novamente:", menu)
		}
		resolvedTicker := ticker
		if quote != nil && quote.Ticker != "" {
			resolvedTicker = quote.Ticker
		}

		state.Ticker = resolvedTicker
		state.Step = "ALERT_EXPECT_COND"
		_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

		condMenu := &telebot.ReplyMarkup{}
		btnAbove := condMenu.Data("🟢 Acima de (>=)", "btn_alert_cond_above")
		btnBelow := condMenu.Data("🔴 Abaixo de (<=)", "btn_alert_cond_below")
		btnBack := condMenu.Data("⬅️ Voltar", "btn_op_back")
		btnCancelOp := condMenu.Data("❌ Cancelar", "btn_cancel_op")
		condMenu.Inline(
			condMenu.Row(btnAbove, btnBelow),
			condMenu.Row(btnBack, btnCancelOp),
		)

		return c.Send(fmt.Sprintf("🔔 *Alerta para %s*\n\nDisparar quando a cotação estiver:", escapeMarkdown(cleanTickerForDisplay(resolvedTicker))), telebot.ModeMarkdown, condMenu)

	case "ALERT_EXPECT_PRICE":
		text = strings.ReplaceAll(text, ",", ".")
		var price float64
		if _, err := fmt.Sscanf(text, "%f", &price); err != nil || price <= 0 {
			priceErrMenu := &telebot.ReplyMarkup{}
			btnBack := priceErrMenu.Data("⬅️ Voltar", "btn_op_back")
			btnCancelOp := priceErrMenu.Data("❌ Cancelar", "btn_cancel_op")
			priceErrMenu.Inline(priceErrMenu.Row(btnBack, btnCancelOp))
			return c.Send("⚠️ Preço inválido. Por favor, envie apenas o número positivo (ex: 35.50):", priceErrMenu)
		}

		userIDStr, err := h.getUserID(c)
		if err != nil {
			return err
		}

		createdAlert, err := h.alertSvc.CreateAlert(ctx, userIDStr, state.Ticker, price, state.Type)
		if err != nil {
			slog.Error("Erro ao criar alerta via telegram", "error", err)
			return c.Send("❌ Ocorreu um erro ao salvar o alerta. Tente novamente mais tarde.", menu)
		}

		_ = h.svc.ClearConversationState(ctx, c.Chat().ID)

		condStr := "acima de"
		if createdAlert.Condition == "BELOW" {
			condStr = "abaixo de"
		}
		curr := createdAlert.Currency
		if curr == "" {
			curr = "BRL"
		}

		p := message.NewPrinter(language.BrazilianPortuguese)
		successMsg := p.Sprintf("✅ *Alerta Criado com Sucesso!*\n\n• Ativo: `%s`\n• Condição: Quando estiver %s %s %.2f",
			cleanTickerForDisplay(createdAlert.Ticker), condStr, getCurrencySymbol(curr), createdAlert.TargetPrice)

		successMenu := &telebot.ReplyMarkup{}
		btnAlertsList := successMenu.Data("🔔 Meus Alertas", "btn_alerts")
		btnMenuBack := successMenu.Data("🏠 Menu", "btn_menu")
		successMenu.Inline(successMenu.Row(btnAlertsList, btnMenuBack))

		return c.Send(successMsg, telebot.ModeMarkdown, successMenu)

	case "QUOTE_EXPECT_TICKER":
		fields := strings.FieldsFunc(text, func(r rune) bool {
			return r == ' ' || r == ',' || r == ';'
		})
		if len(fields) > 1 {
			var tickers []string
			for _, f := range fields {
				clean := strings.ToUpper(strings.TrimSpace(f))
				if clean != "" {
					tickers = append(tickers, clean)
				}
			}
			_ = h.svc.ClearConversationState(ctx, c.Chat().ID)
			return h.renderMultiQuote(c, tickers)
		}

		ticker := strings.ToUpper(text)
		quote, err := h.getQuoteWithFallback(ctx, ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique se há erros de digitação e envie o código novamente:", menu)
		}

		_ = h.svc.ClearConversationState(ctx, c.Chat().ID)

		resolvedTicker := ticker
		if quote != nil && quote.Ticker != "" {
			resolvedTicker = quote.Ticker
		}

		return h.sendQuoteCard(ctx, c, resolvedTicker, quote)

	case "ANALYSIS_EXPECT_TICKER":
		ticker := strings.ToUpper(text)
		_ = h.svc.ClearConversationState(ctx, c.Chat().ID)
		return h.renderAnalysis(c, ticker)

	case "WL_EXPECT_TICKER":
		return h.handleWatchlistAddTicker(c, text)

	default:
		_ = h.svc.ClearConversationState(ctx, c.Chat().ID)
		return c.Send("⚠️ Ação não reconhecida ou expirada. Envie /menu para navegar pelas opções.")
	}
}

func (h *Handlers) tryResolveFreeTextTickers(ctx context.Context, c telebot.Context, text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || strings.HasPrefix(trimmed, "/") {
		return nil
	}

	fields := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ' ' || r == ',' || r == ';'
	})
	if len(fields) == 0 || len(fields) > 10 {
		return nil
	}

	resolved := make([]string, len(fields))
	var pendingIdx []int

	for i, raw := range fields {
		tok := strings.ToUpper(strings.TrimSpace(raw))
		if clean, ok := parseExplicitDollarTicker(tok); ok {
			resolved[i] = clean
			continue
		}
		if isB3Ticker(tok) || isCryptoPairTicker(tok) {
			resolved[i] = tok
			continue
		}
		if isPlainTickerCandidate(tok) {
			resolved[i] = tok
			pendingIdx = append(pendingIdx, i)
			continue
		}
		return nil
	}

	if len(pendingIdx) > 0 {
		known := h.collectUserKnownTickers(ctx, c)
		for _, idx := range pendingIdx {
			if !known[cleanTickerForDisplay(resolved[idx])] {
				return nil
			}
		}
	}

	return resolved
}

func (h *Handlers) collectUserKnownTickers(ctx context.Context, c telebot.Context) map[string]bool {
	known := make(map[string]bool)
	if c == nil {
		return known
	}
	val := c.Get("user_id")
	userIDStr, ok := val.(string)
	if !ok || userIDStr == "" {
		return known
	}

	if h.portfolioSvc != nil {
		var chatID int64
		if c.Chat() != nil {
			chatID = c.Chat().ID
		}
		if portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr); err == nil && len(portfolios) > 0 {
			portID, _ := h.resolveActivePortfolio(ctx, chatID, portfolios)
			if _, positions, err := h.portfolioSvc.GetPortfolioDetails(ctx, portID, userIDStr); err == nil {
				for _, pos := range positions {
					known[cleanTickerForDisplay(pos.Ticker)] = true
				}
			}
		}
	}

	if h.watchlistSvc != nil {
		if lists, err := h.watchlistSvc.GetWatchlists(ctx, userIDStr); err == nil && len(lists) > 0 {
			if wl, err := h.watchlistSvc.GetWatchlist(ctx, lists[0].ID, userIDStr); err == nil && wl != nil {
				for _, item := range wl.Items {
					known[cleanTickerForDisplay(item.Ticker)] = true
				}
			}
		}
	}

	return known
}

func (h *Handlers) getCustodyPosition(ctx context.Context, portfolioID, userID, ticker string) (*portfolio.Position, error) {
	_, positions, err := h.portfolioSvc.GetPortfolioDetails(ctx, portfolioID, userID)
	if err != nil {
		return nil, err
	}
	cleanTicker := strings.ToUpper(strings.TrimSpace(ticker))
	for _, pos := range positions {
		posTicker := strings.ToUpper(strings.TrimSpace(pos.Ticker))
		if posTicker == cleanTicker ||
			(isB3Ticker(cleanTicker) && (posTicker == cleanTicker+".SA" || posTicker+".SA" == cleanTicker)) {
			p := pos
			return &p, nil
		}
	}
	return nil, nil
}

func (h *Handlers) handleQuickAlertCreate(c telebot.Context, ticker string) error {
	defer c.Respond()
	ctx, cancel := h.getContext(c)
	defer cancel()

	resolvedTicker := ticker
	quote, err := h.getQuoteWithFallback(ctx, ticker)
	if err == nil && quote != nil && quote.Ticker != "" {
		resolvedTicker = quote.Ticker
	}

	err = h.svc.SetConversationState(ctx, c.Chat().ID, ConversationState{
		Ticker: resolvedTicker,
		Step:   "ALERT_EXPECT_COND",
	})
	if err != nil {
		return c.Send("❌ Erro interno ao iniciar criação de alerta.")
	}

	condMenu := &telebot.ReplyMarkup{}
	btnAbove := condMenu.Data("🟢 Acima de (>=)", "btn_alert_cond_above")
	btnBelow := condMenu.Data("🔴 Abaixo de (<=)", "btn_alert_cond_below")
	btnBack := condMenu.Data("⬅️ Voltar", "btn_op_back")
	btnCancelOp := condMenu.Data("❌ Cancelar", "btn_cancel_op")
	condMenu.Inline(
		condMenu.Row(btnAbove, btnBelow),
		condMenu.Row(btnBack, btnCancelOp),
	)

	text := fmt.Sprintf("🔔 *Alerta para %s*\n\nDisparar quando a cotação estiver:", escapeMarkdown(cleanTickerForDisplay(resolvedTicker)))
	if c.Callback() != nil {
		return c.Edit(text, telebot.ModeMarkdown, condMenu)
	}
	return c.Send(text, telebot.ModeMarkdown, condMenu)
}

func (h *Handlers) handleQuickWatchlistAdd(c telebot.Context, ticker string) error {
	defer c.Respond()
	if h.watchlistSvc == nil {
		return c.Send("⚠️ Módulo de Favoritos não está ativo.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	lists, err := h.watchlistSvc.GetWatchlists(ctx, userIDStr)
	if err != nil || len(lists) == 0 {
		return c.Send("❌ Erro ao buscar lista de favoritos.")
	}

	resolvedTicker := strings.ToUpper(strings.TrimSpace(ticker))
	_, err = h.watchlistSvc.AddAssetToWatchlist(ctx, lists[0].ID, userIDStr, resolvedTicker)
	if err != nil && isB3Ticker(resolvedTicker) && !strings.HasSuffix(resolvedTicker, ".SA") {
		if _, errSA := h.watchlistSvc.AddAssetToWatchlist(ctx, lists[0].ID, userIDStr, resolvedTicker+".SA"); errSA == nil {
			resolvedTicker = resolvedTicker + ".SA"
			err = nil
		}
	}

	if err != nil {
		return c.Send(fmt.Sprintf("⚠️ Não foi possível adicionar *%s* aos Favoritos: %s", escapeMarkdown(cleanTickerForDisplay(resolvedTicker)), escapeMarkdown(err.Error())), telebot.ModeMarkdown)
	}

	menu := &telebot.ReplyMarkup{}
	btnWatchlist := menu.Data("⭐ Ver Favoritos", "btn_watchlist")
	btnMenu := menu.Data("🏠 Menu", "btn_menu")
	menu.Inline(menu.Row(btnWatchlist, btnMenu))

	msg := fmt.Sprintf("⭐ Ativo *`%s`* adicionado aos seus Favoritos com sucesso!", escapeMarkdown(cleanTickerForDisplay(resolvedTicker)))
	return c.Send(msg, telebot.ModeMarkdown, menu)
}
