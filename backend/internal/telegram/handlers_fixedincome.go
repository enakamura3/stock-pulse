package telegram

import (
	"context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleFixedIncome(c telebot.Context) error {
	defer c.Respond()
	if h.fiSvc == nil {
		return c.Edit("⚠️ Módulo de Renda Fixa não está ativo.")
	}

	userIDStr := c.Get("user_id").(string)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Nenhuma carteira encontrada.")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	positions, err := h.fiSvc.GetPortfolioPositions(context.Background(), portfolioID)
	if err != nil {
		return c.Edit("❌ Erro ao buscar posições de Renda Fixa.")
	}

	if len(positions) == 0 {
		return c.Edit("🏛️ Você ainda não possui ativos de Renda Fixa cadastrados.")
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
			taxa = p.Sprintf("%.2f%% %s", pos.Asset.Rate, pos.Asset.Indexer)
		} else {
			taxa = p.Sprintf("%.2f%% a.a.", pos.Asset.Rate)
		}

		msg += p.Sprintf("• `%s %s` - %s\n", pos.Asset.Institution, pos.Asset.Type, taxa)
		msg += p.Sprintf("  Líquido: R$ %.2f (+%.2f%%)%s\n", pos.NetValue, pos.NetReturnPercent, status)
	}

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	menu.Inline(menu.Row(btnBack))

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}
