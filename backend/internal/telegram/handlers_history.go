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

func encodeHistoryFilterChar(filter string) string {
	switch filter {
	case "BUY":
		return "B"
	case "SELL":
		return "S"
	default:
		return "A"
	}
}

func decodeHistoryFilterChar(ch string) string {
	switch ch {
	case "B":
		return "BUY"
	case "S":
		return "SELL"
	default:
		return "ALL"
	}
}

// buildTxDeleteCallback constrói payload compacto "<prefix><txID>|<page>|<F>|<ticker>" garantindo <= 63 bytes (margem para '\f' no limite de 64B do Telegram).
func buildTxDeleteCallback(prefix, txID string, page int, filter, ticker string) string {
	base := fmt.Sprintf("%s%s|%d|%s", prefix, txID, page, encodeHistoryFilterChar(filter))
	if ticker != "" {
		full := base + "|" + ticker
		if len(full) <= 63 {
			return full
		}
	}
	if len(base) <= 63 {
		return base
	}
	return prefix + txID
}

// parseTxDeleteCallback extrai txID e returnData ("PAGE:FILTER[:TICKER]") de um payload compacto.
func parseTxDeleteCallback(payload string) (string, string) {
	parts := strings.SplitN(payload, "|", 4)
	txID := parts[0]
	if len(parts) < 3 {
		return txID, ""
	}
	page := 0
	fmt.Sscanf(parts[1], "%d", &page)
	filter := decodeHistoryFilterChar(parts[2])
	if len(parts) == 4 && parts[3] != "" {
		return txID, fmt.Sprintf("%d:%s:%s", page, filter, parts[3])
	}
	return txID, fmt.Sprintf("%d:%s", page, filter)
}

func (h *Handlers) HandleHistory(c telebot.Context) error {
	return h.renderHistory(c, false)
}

func (h *Handlers) HandleHistoryDeleteMode(c telebot.Context) error {
	return h.renderHistory(c, true)
}

func (h *Handlers) renderHistory(c telebot.Context, deleteMode bool) error {
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

	cbUnique := "btn_history"
	if deleteMode {
		cbUnique = "btn_history_del"
	}

	menu := &telebot.ReplyMarkup{}
	btnAll := menu.Data(fmt.Sprintf("%s Todas", map[bool]string{true: "▪️", false: "📋"}[filter == "ALL"]), cbUnique, buildFilterData(0, "ALL"))
	btnBuy := menu.Data(fmt.Sprintf("%s Compras", map[bool]string{true: "▪️", false: "🟢"}[filter == "BUY"]), cbUnique, buildFilterData(0, "BUY"))
	btnSell := menu.Data(fmt.Sprintf("%s Vendas", map[bool]string{true: "▪️", false: "🔴"}[filter == "SELL"]), cbUnique, buildFilterData(0, "SELL"))

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
	msg += p.Sprintf("_%s %s — Página %d de %d_\n", filterEmoji, filterLabel, page+1, totalPages)
	if deleteMode {
		msg += "_🗑️ Selecione a operação que deseja excluir:_\n\n"
	} else {
		msg += "\n"
	}

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

	if deleteMode {
		// Atalhos para exclusão pontual de operações exibidas na página (com confirmação inline)
		var delBtns []telebot.Btn
		for _, tx := range filtered[start:end] {
			cbData := buildTxDeleteCallback("txda_", tx.ID, page, filter, tickerFilter)
			btnDel := menu.Data(fmt.Sprintf("🗑️ %s (%s)", cleanTickerForDisplay(tx.Ticker), tx.ExecutedAt.Format("02/01")), cbData)
			delBtns = append(delBtns, btnDel)
		}
		for i := 0; i < len(delBtns); i += 2 {
			if i+1 < len(delBtns) {
				rows = append(rows, menu.Row(delBtns[i], delBtns[i+1]))
			} else {
				rows = append(rows, menu.Row(delBtns[i]))
			}
		}
	}

	var navBtns []telebot.Btn
	if start > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Anterior", cbUnique, buildFilterData(page-1, filter)))
	}
	if end < len(filtered) {
		navBtns = append(navBtns, menu.Data("Próxima ➡️", cbUnique, buildFilterData(page+1, filter)))
	}
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}

	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	if deleteMode {
		btnDone := menu.Data("✅ Concluir", "btn_history", buildFilterData(page, filter))
		rows = append(rows, menu.Row(btnDone, btnBack))
	} else {
		btnDelMode := menu.Data("🗑️ Excluir Operação", "btn_history_del", buildFilterData(page, filter))
		rows = append(rows, menu.Row(btnDelMode, btnBack))
	}

	menu.Inline(rows...)

	return h.sendOrEdit(c, msg, telebot.ModeMarkdown, menu)
}
