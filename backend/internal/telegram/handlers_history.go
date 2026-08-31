package telegram

import (
	"context"
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

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Nenhuma carteira encontrada.")
	}
	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)

	txs, err := h.portfolioSvc.GetPortfolioTransactions(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch transactions for telegram bot", "error", err, "user_id", userIDStr)
		return c.Edit("❌ Ocorreu um erro ao buscar o histórico.")
	}

	if len(txs) == 0 {
		menu := &telebot.ReplyMarkup{}
		btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
		menu.Inline(menu.Row(btnBack))
		return c.Edit("📜 Nenhuma operação encontrada na sua carteira.", menu)
	}

	// Parse callback data: "PAGE:FILTER" (ex: "0:ALL", "1:BUY", "0:SELL")
	rawData := c.Data()
	page := 0
	filter := "ALL"
	if rawData != "" {
		parts := strings.SplitN(rawData, ":", 2)
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &page)
			filter = strings.ToUpper(parts[1])
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

	menu := &telebot.ReplyMarkup{}
	btnAll := menu.Data(fmt.Sprintf("%s Todas", map[bool]string{true: "▪️", false: "📋"}[filter == "ALL"]), "btn_history", "0:ALL")
	btnBuy := menu.Data(fmt.Sprintf("%s Compras", map[bool]string{true: "▪️", false: "🟢"}[filter == "BUY"]), "btn_history", "0:BUY")
	btnSell := menu.Data(fmt.Sprintf("%s Vendas", map[bool]string{true: "▪️", false: "🔴"}[filter == "SELL"]), "btn_history", "0:SELL")

	if len(filtered) == 0 {
		msg := fmt.Sprintf("📜 *Histórico: %s*\n_%s %s — Nenhuma operação encontrada._", portfolioName, filterEmoji, filterLabel)
		btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
		menu.Inline(menu.Row(btnAll, btnBuy, btnSell), menu.Row(btnBack))
		err = c.Edit(msg, telebot.ModeMarkdown, menu)
		if err != nil && strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📜 *Histórico: %s*\n_%s %s — Página %d de %d_\n\n", portfolioName, filterEmoji, filterLabel, page+1, totalPages)

	for _, tx := range filtered[start:end] {
		tipoStr := "🟢 Compra"
		if tx.Type == "SELL" {
			tipoStr = "🔴 Venda"
		}

		msg += p.Sprintf("%s | `%s`\n", tipoStr, tx.Ticker)
		msg += p.Sprintf("Data: %s\n", tx.ExecutedAt.Format("2006-01-02"))
		if tx.Fee > 1e-6 {
			msg += p.Sprintf("Qtd: %.4f | Preço: R$ %.2f | Total: R$ %.2f (Taxas: R$ %.2f)\n\n", tx.Quantity, tx.UnitPrice, tx.TotalCost, tx.Fee)
		} else {
			msg += p.Sprintf("Qtd: %.4f | Preço: R$ %.2f | Total: R$ %.2f\n\n", tx.Quantity, tx.UnitPrice, tx.TotalCost)
		}
	}

	var rows []telebot.Row
	rows = append(rows, menu.Row(btnAll, btnBuy, btnSell))

	var navBtns []telebot.Btn
	if start > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_history", fmt.Sprintf("%d:%s", page-1, filter)))
	}
	if end < len(filtered) {
		navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_history", fmt.Sprintf("%d:%s", page+1, filter)))
	}
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}

	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)

	err = c.Edit(msg, telebot.ModeMarkdown, menu)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}
