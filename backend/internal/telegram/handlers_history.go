package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleHistory(c telebot.Context) error {
	defer c.Respond()
	userIDStr := c.Get("user_id").(string)

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
		return c.Edit("📜 Nenhuma operação encontrada na sua carteira.")
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

	var rows []telebot.Row
	if len(btns) > 0 {
		rows = append(rows, menu.Row(btns...))
	}
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}
