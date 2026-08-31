package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandlePortfolioSummary(c telebot.Context) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

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

		trPos, err := h.fiSvc.GetTreasuryPositions(context.Background(), portfolioID)
		if err == nil {
			for _, pos := range trPos {
				totalFIValue += pos.NetValue
				totalValue += pos.NetValue
				totalCost += pos.TotalInvested

				if pos.DaysToMaturity <= 30 && !pos.IsMatured {
					nearMaturity = append(nearMaturity, fixedincome.Position{
						Asset: fixedincome.Asset{
							Institution: "Tesouro Direto",
							Type:        pos.TreasuryType,
						},
						DaysToMaturity: pos.DaysToMaturity,
					})
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

	// Consolidação de Alocação Patrimonial por Classe de Ativo
	catTotals := make(map[string]float64)
	catNames := map[string]string{
		"STOCK":  "Ações",
		"FII":    "FIIs",
		"ETF":    "ETFs",
		"RF":     "Renda Fixa & Tesouro",
		"CRYPTO": "Cripto",
		"BDR":    "BDRs",
		"OTHER":  "Outros",
	}
	catEmojis := map[string]string{
		"STOCK":  "📈",
		"FII":    "🏢",
		"ETF":    "🌐",
		"RF":     "💵",
		"CRYPTO": "₿",
		"BDR":    "📦",
		"OTHER":  "🎯",
	}

	for _, pos := range positions {
		catID := getMacroCategoryKey(pos.Type, pos.Ticker)
		catTotals[catID] += pos.CurrentValue
	}
	if totalFIValue > 0 {
		catTotals["RF"] += totalFIValue
	}

	if totalValue > 0 && len(catTotals) > 0 {
		msg += "\n🧱 *Alocação Patrimonial*\n"
		type catItem struct {
			key   string
			total float64
			pct   float64
		}
		var catList []catItem
		for k, val := range catTotals {
			if val > 0 {
				catList = append(catList, catItem{
					key:   k,
					total: val,
					pct:   (val / totalValue) * 100,
				})
			}
		}
		sort.Slice(catList, func(i, j int) bool {
			return catList[i].total > catList[j].total
		})

		for _, item := range catList {
			name := catNames[item.key]
			emoji := catEmojis[item.key]
			msg += p.Sprintf("• %s %s: *R$ %.2f* (%.2f%%)\n", emoji, name, item.total, item.pct)
		}
	}

	if len(nearMaturity) > 0 {
		msg += "\n⚠️ *Vencimentos Próximos (Renda Fixa)*\n"
		for _, pos := range nearMaturity {
			msg += p.Sprintf("• `%s` (%s): Vence em %d dias\n", pos.Asset.Institution, pos.Asset.Type, pos.DaysToMaturity)
		}
	}

	sortedPos := make([]portfolio.Position, len(positions))
	copy(sortedPos, positions)
	sort.Slice(sortedPos, func(i, j int) bool {
		return sortedPos[i].DailyChangePercent > sortedPos[j].DailyChangePercent
	})

	var risers []portfolio.Position
	var fallers []portfolio.Position

	for _, pos := range sortedPos {
		if pos.DailyChangePercent > 0 {
			risers = append(risers, pos)
		} else if pos.DailyChangePercent < 0 {
			fallers = append(fallers, pos)
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
		for i := len(fallers) - 1; i >= len(fallers)-limit; i-- {
			msg += p.Sprintf("• `%s`: %.2f%%\n", fallers[i].Ticker, fallers[i].DailyChangePercent)
		}
	}

	if h.marketSvc != nil {
		if benchmarks, err := h.marketSvc.GetBenchmarks(context.Background()); err == nil && benchmarks != nil {
			var bmItems []struct {
				name          string
				changePercent float64
			}
			if benchmarks.IBOV != nil {
				bmItems = append(bmItems, struct {
					name          string
					changePercent float64
				}{name: "IBOV", changePercent: benchmarks.IBOV.ChangePercent})
			}
			if benchmarks.IFIX != nil {
				bmItems = append(bmItems, struct {
					name          string
					changePercent float64
				}{name: "IFIX", changePercent: benchmarks.IFIX.ChangePercent})
			}
			if benchmarks.SP500 != nil {
				bmItems = append(bmItems, struct {
					name          string
					changePercent float64
				}{name: "S&P 500", changePercent: benchmarks.SP500.ChangePercent})
			}
			if benchmarks.USDBRL != nil {
				bmItems = append(bmItems, struct {
					name          string
					changePercent float64
				}{name: "Dólar", changePercent: benchmarks.USDBRL.ChangePercent})
			}

			if len(bmItems) > 0 {
				msg += "\n📊 *Benchmarks do Dia*\n"
				for _, item := range bmItems {
					var symbol string
					if item.changePercent > 1e-6 {
						symbol = "🟢"
					} else if item.changePercent < -1e-6 {
						symbol = "🔴"
					} else {
						symbol = "⚪"
					}
					msg += p.Sprintf("• %s *%s:* %+.2f%%\n", symbol, item.name, item.changePercent)
				}
			}
		}
	}

	menu := &telebot.ReplyMarkup{}
	btnRefresh := menu.Data("🔄 Atualizar", "btn_resumo")
	btnAtivos := menu.Data("📋 Todos os Ativos", "btn_ativos")
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	menu.Inline(menu.Row(btnRefresh, btnAtivos), menu.Row(btnBack))

	err = c.Edit(msg, telebot.ModeMarkdown, menu)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}

func (h *Handlers) HandleAssetList(c telebot.Context) error {
	defer c.Respond()
	pageStr := c.Data()
	page := 0
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

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

	if len(positions) == 0 {
		menu := &telebot.ReplyMarkup{}
		btnBack := menu.Data("⬅️ Voltar ao Resumo", "btn_resumo")
		btnMenu := menu.Data("🏠 Menu", "btn_menu")
		menu.Inline(menu.Row(btnBack, btnMenu))
		return c.Edit(fmt.Sprintf("📋 *Ativos: %s*\n\nNenhum ativo encontrado nesta carteira.", portfolioName), telebot.ModeMarkdown, menu)
	}

	posByValue := make([]portfolio.Position, len(positions))
	copy(posByValue, positions)
	sort.Slice(posByValue, func(i, j int) bool {
		return posByValue[i].CurrentValue > posByValue[j].CurrentValue
	})

	pageSize := 10
	totalPages := (len(posByValue) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	start := page * pageSize
	if start >= len(posByValue) {
		start = 0
		page = 0
	}
	end := start + pageSize
	if end > len(posByValue) {
		end = len(posByValue)
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📋 *Ativos: %s*\n_Página %d de %d_\n\n", portfolioName, page+1, totalPages)

	for _, pos := range posByValue[start:end] {
		var symbol string
		if pos.DailyChangePercent > 1e-6 {
			symbol = "🟢"
		} else if pos.DailyChangePercent < -1e-6 {
			symbol = "🔴"
		} else {
			symbol = "⚪"
		}

		totalReturn := 0.0
		if pos.TotalCost > 1e-6 {
			totalReturn = ((pos.CurrentValue - pos.TotalCost) / pos.TotalCost) * 100
		}

		msg += p.Sprintf("%s `%s`: *R$ %.2f* | Dia: %+.2f%% | L/P: %+.2f%%\n",
			symbol, pos.Ticker, pos.CurrentValue, pos.DailyChangePercent, totalReturn)
	}

	menu := &telebot.ReplyMarkup{}
	var navBtns []telebot.Btn
	if start > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_ativos", fmt.Sprintf("%d", page-1)))
	}
	if end < len(posByValue) {
		navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_ativos", fmt.Sprintf("%d", page+1)))
	}

	var rows []telebot.Row
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}
	btnBack := menu.Data("⬅️ Voltar ao Resumo", "btn_resumo")
	btnMenu := menu.Data("🏠 Menu", "btn_menu")
	rows = append(rows, menu.Row(btnBack, btnMenu))
	menu.Inline(rows...)

	err = c.Edit(msg, telebot.ModeMarkdown, menu)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}

func getMacroCategoryKey(assetType, ticker string) string {
	tUpper := strings.ToUpper(ticker)
	typeUpper := strings.ToUpper(assetType)

	if strings.Contains(typeUpper, "ETF") {
		return "ETF"
	}
	if strings.Contains(typeUpper, "CRYPTO") {
		return "CRYPTO"
	}
	if strings.Contains(typeUpper, "BDR") {
		return "BDR"
	}
	if strings.Contains(typeUpper, "RF") || strings.Contains(typeUpper, "FIXED") {
		return "RF"
	}
	if typeUpper == "FII" || strings.HasSuffix(tUpper, "11.SA") || strings.HasSuffix(tUpper, "11") {
		return "FII"
	}
	return "STOCK"
}

func (h *Handlers) HandleChangePortfolio(c telebot.Context) error {
	defer c.Respond()
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

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
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

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
