package telegram

import (
	"fmt"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleFixedIncome(c telebot.Context) error {
	defer c.Respond()
	if h.fiSvc == nil {
		return h.sendOrEdit(c, "⚠️ Módulo de Renda Fixa não está ativo.")
	}

	page := 0
	if rawData := c.Data(); rawData != "" {
		fmt.Sscanf(rawData, "%d", &page)
	}

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
	positions, err1 := h.fiSvc.GetPortfolioPositions(ctx, portfolioID)
	treasuryPositions, err2 := h.fiSvc.GetTreasuryPositions(ctx, portfolioID)

	if (err1 != nil && err2 != nil) || (err1 != nil && len(treasuryPositions) == 0) {
		return h.sendOrEdit(c, "❌ Erro ao buscar posições de Renda Fixa.")
	}

	if len(positions) == 0 && len(treasuryPositions) == 0 {
		menu := &telebot.ReplyMarkup{}
		btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
		menu.Inline(menu.Row(btnBack))
		return h.sendOrEdit(c, "🏛️ Você ainda não possui ativos de Renda Fixa ou Tesouro Direto cadastrados.", menu)
	}

	var totalBruto, totalLiquido, totalCusto float64
	for _, pos := range positions {
		totalBruto += pos.GrossValue
		totalLiquido += pos.NetValue
		totalCusto += pos.TotalInvested
	}
	for _, pos := range treasuryPositions {
		totalBruto += pos.GrossValue
		totalLiquido += pos.NetValue
		totalCusto += pos.TotalInvested
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	var items []string

	for _, pos := range positions {
		status := ""
		if pos.IsMatured {
			status = " *(VENCIDO)*"
		} else if pos.DaysToMaturity <= 30 {
			status = " *(Vence logo!)*"
		}

		taxa := ""
		if pos.Asset.DebtType == "POS" {
			taxa = p.Sprintf("%.2f%% do %s", pos.Asset.Rate, pos.Asset.Indexer)
		} else {
			taxa = p.Sprintf("%.2f%% a.a.", pos.Asset.Rate)
		}

		vencStr := ""
		if !pos.Asset.MaturityDate.IsZero() {
			vencStr = p.Sprintf(" • Venc.: %s", pos.Asset.MaturityDate.Format("02/01/2006"))
		}

		line := p.Sprintf("• `%s %s` - %s%s\n  Aplicado: R$ %s → Líquido: R$ %s (%+.2f%%)%s",
			pos.Asset.Institution, pos.Asset.Type, taxa, vencStr,
			formatFinancialPrice(p, pos.TotalInvested), formatFinancialPrice(p, pos.NetValue), pos.NetReturnPercent, status)
		items = append(items, line)
	}

	for _, pos := range treasuryPositions {
		status := ""
		if pos.IsMatured {
			status = " *(VENCIDO)*"
		} else if pos.DaysToMaturity <= 30 {
			status = " *(Vence logo!)*"
		}

		netReturnPct := 0.0
		if pos.TotalInvested > 1e-6 {
			netReturnPct = ((pos.NetValue - pos.TotalInvested) / pos.TotalInvested) * 100
		}

		vencStr := ""
		if !pos.MaturityDate.IsZero() {
			vencStr = p.Sprintf(" • Venc.: %s", pos.MaturityDate.Format("02/01/2006"))
		}

		line := p.Sprintf("• `Tesouro %s` (%s un.)%s\n  Aplicado: R$ %s → Líquido: R$ %s (%+.2f%%)%s",
			pos.TreasuryType, formatQuantity(pos.Quantity), vencStr,
			formatFinancialPrice(p, pos.TotalInvested), formatFinancialPrice(p, pos.NetValue), netReturnPct, status)
		items = append(items, line)
	}

	pageSize := 5
	totalPages := (len(items) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	start := page * pageSize
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	msg := p.Sprintf("🏛️ *Renda Fixa & Tesouro: %s*\n\n", escapeMarkdown(portfolioName))
	msg += p.Sprintf("💰 Valor Líquido: *R$ %s*\n", formatFinancialPrice(p, totalLiquido))
	msg += p.Sprintf("💵 Total Aplicado: R$ %s\n", formatFinancialPrice(p, totalCusto))
	msg += p.Sprintf("📈 Valor Bruto: R$ %s\n", formatFinancialPrice(p, totalBruto))

	lucro := totalLiquido - totalCusto
	lucroPct := 0.0
	if totalCusto > 1e-6 {
		lucroPct = (lucro / totalCusto) * 100
	}
	msg += p.Sprintf("⚖️ Lucro Líquido: R$ %s (%.2f%%)\n\n", formatFinancialPrice(p, lucro), lucroPct)

	if totalPages > 1 {
		msg += p.Sprintf("*Minhas Posições* (Página %d de %d):\n", page+1, totalPages)
	} else {
		msg += "*Minhas Posições:*\n"
	}

	for _, item := range items[start:end] {
		msg += item + "\n"
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	var navBtns []telebot.Btn
	if page > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_renda_fixa", fmt.Sprintf("%d", page-1)))
	}
	if end < len(items) {
		navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_renda_fixa", fmt.Sprintf("%d", page+1)))
	}
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}

	btnRefresh := menu.Data("🔄 Atualizar", "btn_renda_fixa", fmt.Sprintf("%d", page))
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnRefresh), menu.Row(btnBack))

	menu.Inline(rows...)

	return h.sendOrEdit(c, msg, telebot.ModeMarkdown, menu)
}
