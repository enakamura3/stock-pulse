package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandlePortfolioSummary(c telebot.Context) error {
	defer c.Respond()
	userIDStr := c.Get("user_id").(string)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	_, positions, err := h.portfolioSvc.GetPortfolioDetails(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch portfolio for telegram bot", "error", err, "user_id", userIDStr)
		return c.Edit("❌ Ocorreu um erro ao buscar sua carteira.")
	}

	var totalValue, totalCost, totalDailyChange float64
	for _, pos := range positions {
		totalValue += pos.CurrentValue
		totalCost += pos.TotalCost
		rate := 1.0
		if pos.CurrentPrice > 0 && pos.Quantity > 0 {
			rate = pos.CurrentValue / (pos.CurrentPrice * pos.Quantity)
		}
		totalDailyChange += pos.DailyChange * pos.Quantity * rate
	}

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

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	menu.Inline(menu.Row(btnBack))

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleChangePortfolio(c telebot.Context) error {
	defer c.Respond()
	userIDStr := c.Get("user_id").(string)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Edit("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, p := range portfolios {
		btn := menu.Data(fmt.Sprintf("📂 %s", p.Name), "btn_sel_port_"+p.ID)
		rows = append(rows, menu.Row(btn))
	}
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)
	return c.Edit("Qual carteira você deseja definir como Ativa?", menu)
}

func (h *Handlers) handleSelectedPortfolio(c telebot.Context, portfolioID string) error {
	defer c.Respond()
	userIDStr := c.Get("user_id").(string)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil {
		return c.Edit("❌ Erro ao buscar carteiras.")
	}

	var pName string
	for _, p := range portfolios {
		if p.ID == portfolioID {
			pName = p.Name
			break
		}
	}

	if pName == "" {
		return c.Edit("❌ Carteira inválida.")
	}

	err = h.svc.SetActivePortfolio(context.Background(), c.Chat().ID, portfolioID)
	if err != nil {
		slog.Error("Failed to set active portfolio", "error", err)
		return c.Edit("❌ Erro interno ao salvar carteira ativa.")
	}

	// Após trocar com sucesso, voltar ao menu
	return h.sendOrEditMenu(c)
}
