package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleCancelOperation(c telebot.Context) error {
	defer c.Respond()
	_ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)
	return h.sendOrEditMenu(c)
}

func (h *Handlers) HandleLaunchOperation(c telebot.Context) error {
	defer c.Respond()
	userIDStr := c.Get("user_id").(string)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Nenhuma carteira encontrada na sua conta.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)

	_, positions, err := h.portfolioSvc.GetPortfolioDetails(context.Background(), portfolioID, userIDStr)
	if err != nil {
		return c.Edit("❌ Ocorreu um erro ao buscar seus ativos.")
	}

	err = h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
		Step:        "EXPECT_TICKER",
		PortfolioID: portfolioID,
	})
	if err != nil {
		return c.Edit("❌ Erro interno ao iniciar operação.")
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, pos := range positions {
		btn := menu.Data(fmt.Sprintf("%s (%s)", pos.Ticker, pos.Name), "btn_ticker_"+pos.Ticker)
		rows = append(rows, menu.Row(btn))
	}

	btnNew := menu.Data("➕ Novo Ativo", "btn_new_asset")
	rows = append(rows, menu.Row(btnNew))

	btnCancel := menu.Data("❌ Cancelar Operação", "btn_cancel_op")
	rows = append(rows, menu.Row(btnCancel))

	menu.Inline(rows...)

	return c.Edit(fmt.Sprintf("🏢 *Carteira Ativa:* %s\nPara qual ativo deseja lançar a operação?", portfolioName), telebot.ModeMarkdown, menu)
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

	if strings.HasPrefix(data, "btn_qty_") {
		qtyStr := strings.TrimPrefix(data, "btn_qty_")
		return h.handleSelectedQty(c, qtyStr)
	}

	return nil
}

func (h *Handlers) handleSelectedTicker(c telebot.Context, ticker string) error {
	defer c.Respond()
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
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")

	menu.Inline(menu.Row(btnBuy, btnSell), menu.Row(btnCancel))

	msg := fmt.Sprintf("Operação para *%s*.\n\nÉ uma operação de *Compra* ou *Venda*?", ticker)
	if c.Callback() != nil {
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleNewAsset(c telebot.Context) error {
	defer c.Respond()
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	return c.Edit("Qual o código do ativo? (ex: AAPL, PETR4.SA)", menu)
}

func (h *Handlers) HandleSetTypeBuy(c telebot.Context) error {
	return h.handleSetType(c, "BUY")
}

func (h *Handlers) HandleSetTypeSell(c telebot.Context) error {
	return h.handleSetType(c, "SELL")
}

func (h *Handlers) handleSetType(c telebot.Context, txType string) error {
	defer c.Respond()
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	state.Type = txType
	state.Step = "EXPECT_QTY"
	_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

	menu := &telebot.ReplyMarkup{}
	btnQ1 := menu.Data("1", "btn_qty_1")
	btnQ10 := menu.Data("10", "btn_qty_10")
	btnQ100 := menu.Data("100", "btn_qty_100")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")

	menu.Inline(menu.Row(btnQ1, btnQ10, btnQ100), menu.Row(btnCancel))

	return c.Edit("Qual a quantidade negociada?\n\nEscolha abaixo ou digite o valor:", menu)
}

func (h *Handlers) handleSelectedQty(c telebot.Context, qtyStr string) error {
	defer c.Respond()
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return c.Edit("⚠️ Nenhuma operação em andamento.")
	}

	var qty float64
	fmt.Sscanf(qtyStr, "%f", &qty)

	state.Quantity = qty
	state.Step = "EXPECT_PRICE"
	_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	return c.Edit("Qual o preço unitário da transação? (ex: 15.50)", menu)
}

func (h *Handlers) HandleText(c telebot.Context) error {
	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil {
		return h.sendOrEditMenu(c)
	}

	text := strings.TrimSpace(c.Text())
	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	switch state.Step {
	case "EXPECT_TICKER":
		ticker := strings.ToUpper(text)
		_, err := h.marketSvc.GetQuote(context.Background(), ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado na bolsa. Verifique se há erros de digitação e envie o código novamente:", menu)
		}

		return h.handleSelectedTicker(c, ticker)

	case "EXPECT_QTY":
		text = strings.ReplaceAll(text, ",", ".")
		var qty float64
		if _, err := fmt.Sscanf(text, "%f", &qty); err != nil || qty <= 0 {
			return c.Send("⚠️ Quantidade inválida. Por favor, envie apenas o número (ex: 10):", menu)
		}

		state.Quantity = qty
		state.Step = "EXPECT_PRICE"
		_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

		return c.Send("Qual o preço unitário da transação? (ex: 15.50)", menu)

	case "EXPECT_PRICE":
		text = strings.ReplaceAll(text, ",", ".")
		var price float64
		if _, err := fmt.Sscanf(text, "%f", &price); err != nil || price <= 0 {
			return c.Send("⚠️ Preço inválido. Por favor, envie apenas o número (ex: 15.50):", menu)
		}

		userIDStr := c.Get("user_id").(string)

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

		_, err = h.portfolioSvc.AddTransaction(context.Background(), userIDStr, tx)
		if err != nil {
			slog.Error("Erro ao lançar transação via telegram", "error", err)
			return c.Send("❌ Ocorreu um erro ao salvar a transação. Tente novamente mais tarde.", menu)
		}

		_ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)

		tipoStr := "COMPRA"
		if state.Type == "SELL" {
			tipoStr = "VENDA"
		}

		p := message.NewPrinter(language.BrazilianPortuguese)
		successMsg := p.Sprintf("✅ *Operação Lançada com Sucesso!*\n\nAtivo: %s\nTipo: %s\nQuantidade: %.4f\nPreço Unitário: %.2f\nTotal: %.2f",
			state.Ticker, tipoStr, state.Quantity, price, tx.TotalCost)

		successMenu := &telebot.ReplyMarkup{}
		btnMenu := successMenu.Data("🏠 Voltar ao Menu", "btn_menu")
		successMenu.Inline(successMenu.Row(btnMenu))

		return c.Send(successMsg, telebot.ModeMarkdown, successMenu)
	}

	return nil
}
