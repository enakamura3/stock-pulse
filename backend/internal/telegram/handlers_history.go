package telegram

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleHistory(c telebot.Context) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	var chatID int64
	if c.Chat() != nil {
		chatID = c.Chat().ID
		_ = h.svc.ClearConversationState(ctx, chatID)
	}

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		return h.sendOrEdit(c, "⚠️ Nenhuma carteira encontrada.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(ctx, chatID, portfolios)

	txs, err := h.portfolioSvc.GetPortfolioTransactions(ctx, portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch transactions for telegram bot", "error", err, "user_id", userIDStr)
		return h.sendOrEdit(c, "❌ Ocorreu um erro ao buscar o histórico.")
	}

	if len(txs) == 0 {
		menu := &telebot.ReplyMarkup{}
		btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
		menu.Inline(menu.Row(btnBack))
		return h.sendOrEdit(c, "📜 Nenhuma operação encontrada na sua carteira.", menu)
	}

	// Parse callback data: "PAGE:FILTER[:TICKER]" (ex: "0:ALL", "1:BUY", "0:ALL:PETR4")
	rawData := c.Data()
	page := 0
	filter := "ALL"
	tickerFilter := ""
	if args := c.Args(); len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		tickerFilter = cleanTickerForDisplay(strings.TrimSpace(args[0]))
	}
	if rawData != "" {
		parts := strings.SplitN(rawData, ":", 3)
		if len(parts) >= 2 {
			fmt.Sscanf(parts[0], "%d", &page)
			filter = strings.ToUpper(parts[1])
			if len(parts) == 3 {
				tickerFilter = cleanTickerForDisplay(strings.TrimSpace(parts[2]))
			}
		} else {
			// backward compat: apenas número sem filtro
			fmt.Sscanf(rawData, "%d", &page)
		}
	}
	if filter != "ALL" && filter != "BUY" && filter != "SELL" {
		filter = "ALL"
	}

	// Aplicar filtro
	var filtered []portfolio.Transaction
	for _, tx := range txs {
		if tickerFilter != "" && cleanTickerForDisplay(tx.Ticker) != tickerFilter {
			continue
		}
		if filter == "ALL" || tx.Type == filter {
			filtered = append(filtered, tx)
		}
	}

	pageSize := 10
	totalPages := (len(filtered) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	start := page * pageSize
	if start >= len(filtered) {
		start = 0
		page = 0
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	filterEmoji := "📋"
	filterLabel := "Todas"
	if filter == "BUY" {
		filterEmoji = "🟢"
		filterLabel = "Compras"
	} else if filter == "SELL" {
		filterEmoji = "🔴"
		filterLabel = "Vendas"
	}

	buildFilterData := func(p int, f string) string {
		if tickerFilter != "" {
			return fmt.Sprintf("%d:%s:%s", p, f, tickerFilter)
		}
		return fmt.Sprintf("%d:%s", p, f)
	}

	menu := &telebot.ReplyMarkup{}
	btnAll := menu.Data(fmt.Sprintf("%s Todas", map[bool]string{true: "▪️", false: "📋"}[filter == "ALL"]), "btn_history", buildFilterData(0, "ALL"))
	btnBuy := menu.Data(fmt.Sprintf("%s Compras", map[bool]string{true: "▪️", false: "🟢"}[filter == "BUY"]), "btn_history", buildFilterData(0, "BUY"))
	btnSell := menu.Data(fmt.Sprintf("%s Vendas", map[bool]string{true: "▪️", false: "🔴"}[filter == "SELL"]), "btn_history", buildFilterData(0, "SELL"))

	if len(filtered) == 0 {
		msg := fmt.Sprintf("📜 *Histórico: %s*\n_%s %s — Nenhuma operação encontrada._", escapeMarkdown(portfolioName), filterEmoji, filterLabel)
		btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
		menu.Inline(menu.Row(btnAll, btnBuy, btnSell), menu.Row(btnBack))
		err = h.sendOrEdit(c, msg, telebot.ModeMarkdown, menu)
		return err
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📜 *Histórico: %s*\n", escapeMarkdown(portfolioName))
	if tickerFilter != "" {
		msg += p.Sprintf("🔎 Ativo filtrado: *`%s`*\n", escapeMarkdown(tickerFilter))
	}
	msg += p.Sprintf("_%s %s — Página %d de %d_\n\n", filterEmoji, filterLabel, page+1, totalPages)

	for _, tx := range filtered[start:end] {
		tipoStr := "🟢 Compra"
		if tx.Type == "SELL" {
			tipoStr = "🔴 Venda"
		}

		curr := getTickerDefaultCurrency(tx.Ticker)
		if tx.Currency != "" {
			curr = getCurrencySymbol(tx.Currency)
		}

		msg += p.Sprintf("%s | `%s` • %s\n", tipoStr, cleanTickerForDisplay(tx.Ticker), tx.ExecutedAt.Format("02/01/2006"))
		if tx.Fee > 1e-6 {
			msg += p.Sprintf("Qtd: %s | Preço: %s %s | Total: %s %s (Taxas: %s %s)\n\n",
				formatQuantity(tx.Quantity), curr, formatFinancialPrice(p, tx.UnitPrice), curr, formatFinancialPrice(p, tx.TotalCost), curr, formatFinancialPrice(p, tx.Fee))
		} else {
			msg += p.Sprintf("Qtd: %s | Preço: %s %s | Total: %s %s\n\n",
				formatQuantity(tx.Quantity), curr, formatFinancialPrice(p, tx.UnitPrice), curr, formatFinancialPrice(p, tx.TotalCost))
		}
	}

	var rows []telebot.Row
	rows = append(rows, menu.Row(btnAll, btnBuy, btnSell))

	// Atalhos para exclusão pontual de operações exibidas na página
	var delBtns []telebot.Btn
	for _, tx := range filtered[start:end] {
		btnDel := menu.Data(fmt.Sprintf("🗑️ %s (%s)", cleanTickerForDisplay(tx.Ticker), tx.ExecutedAt.Format("02/01")), "btn_tx_del_"+tx.ID)
		delBtns = append(delBtns, btnDel)
	}
	for i := 0; i < len(delBtns); i += 2 {
		if i+1 < len(delBtns) {
			rows = append(rows, menu.Row(delBtns[i], delBtns[i+1]))
		} else {
			rows = append(rows, menu.Row(delBtns[i]))
		}
	}

	var navBtns []telebot.Btn
	if start > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_history", buildFilterData(page-1, filter)))
	}
	if end < len(filtered) {
		navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_history", buildFilterData(page+1, filter)))
	}
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}

	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)

	return h.sendOrEdit(c, msg, telebot.ModeMarkdown, menu)
}
