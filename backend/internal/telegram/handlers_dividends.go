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

func getCurrencySymbol(code string) string {
	switch code {
	case "USD":
		return "US$"
	case "EUR":
		return "€"
	case "BRL":
		return "R$"
	default:
		return "R$"
	}
}

func abbreviateDividendType(t string) string {
	switch strings.ToUpper(t) {
	case "DIVIDENDO", "DIVIDENDOS", "DIV":
		return "DIV"
	case "JUROS SOBRE CAPITAL PRÓPRIO", "JUROS SOBRE CAPITAL PROPRIO", "JCP":
		return "JCP"
	case "RENDIMENTO", "RENDIMENTOS", "REND":
		return "REND"
	case "AMORTIZAÇÃO", "AMORTIZACAO":
		return "AMORT"
	default:
		if len(t) > 4 {
			return strings.ToUpper(t[:4])
		}
		return strings.ToUpper(t)
	}
}

func getMonthNamePT(m time.Month) string {
	months := map[time.Month]string{
		time.January:   "Janeiro",
		time.February:  "Fevereiro",
		time.March:     "Março",
		time.April:     "Abril",
		time.May:       "Maio",
		time.June:      "Junho",
		time.July:      "Julho",
		time.August:    "Agosto",
		time.September: "Setembro",
		time.October:   "Outubro",
		time.November:  "Novembro",
		time.December:  "Dezembro",
	}
	return months[m]
}

func getAssetTypeEmoji(assetType, ticker string) string {
	tickerUpper := strings.ToUpper(ticker)
	if !strings.HasSuffix(tickerUpper, ".SA") {
		if strings.Contains(tickerUpper, "-") {
			return "🪙"
		}
		return "🇺🇸"
	}

	switch strings.ToUpper(assetType) {
	case "FII", "FIAGRO":
		return "🏢"
	case "ETF_BR", "ETF":
		return "📊"
	case "BDR":
		return "🌐"
	case "CRYPTO":
		return "🪙"
	default:
		if strings.HasSuffix(tickerUpper, "11.SA") {
			return "🏢"
		}
		return "📈"
	}
}

func cleanTickerForDisplay(ticker string) string {
	return strings.TrimSuffix(strings.ToUpper(ticker), ".SA")
}

func formatQuantity(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%.0f", q)
	}
	return fmt.Sprintf("%.2f", q)
}

func formatPerShareAmount(p *message.Printer, amount float64) string {
	s := p.Sprintf("%.4f", amount)
	if strings.HasSuffix(s, "00") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "0") {
		return s[:len(s)-1]
	}
	return s
}

func (h *Handlers) fetchDividends(c telebot.Context) ([]portfolio.CalculatedDividend, string, error) {
	userIDStr := c.Get("user_id").(string)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return nil, "", fmt.Errorf("nenhuma carteira")
	}

	portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	divs, err := h.portfolioSvc.GetPortfolioDividends(context.Background(), portfolioID, userIDStr)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao buscar")
	}
	return divs, portfolioName, nil
}

func sortCurrencies(currencies []string) {
	sort.Slice(currencies, func(i, j int) bool {
		if currencies[i] == "BRL" {
			return true
		}
		if currencies[j] == "BRL" {
			return false
		}
		if currencies[i] == "USD" {
			return true
		}
		if currencies[j] == "USD" {
			return false
		}
		return currencies[i] < currencies[j]
	})
}

func (h *Handlers) HandleDividends(c telebot.Context) error {
	defer c.Respond()

	divs, portfolioName, err := h.fetchDividends(c)
	if err != nil {
		slog.Error("Failed to fetch dividends for telegram bot", "error", err)
		return c.Edit("❌ Ocorreu um erro ao buscar os proventos da sua carteira.")
	}

	totalPaidMonth := make(map[string]float64)
	totalFutureMonth := make(map[string]float64)
	totalAccumulated := make(map[string]float64)
	now := time.Now()
	currentMonth := now.Month()
	currentYear := now.Year()

	for _, d := range divs {
		currency := d.Currency
		if currency == "" {
			currency = "BRL"
		}

		if !d.PaymentDate.After(now) {
			totalAccumulated[currency] += d.NetAmount
		}

		if d.PaymentDate.Year() == currentYear && d.PaymentDate.Month() == currentMonth {
			if d.PaymentDate.After(now) {
				totalFutureMonth[currency] += d.NetAmount
			} else {
				totalPaidMonth[currency] += d.NetAmount
			}
		}
	}

	p := message.NewPrinter(language.BrazilianPortuguese)

	var accumulatedStrings []string
	var accumulatedCurrencies []string
	for curr := range totalAccumulated {
		accumulatedCurrencies = append(accumulatedCurrencies, curr)
	}
	sortCurrencies(accumulatedCurrencies)
	for _, curr := range accumulatedCurrencies {
		accumulatedStrings = append(accumulatedStrings, p.Sprintf("%s %.2f", getCurrencySymbol(curr), totalAccumulated[curr]))
	}

	totalAccumulatedStr := "R$ 0,00"
	if len(accumulatedStrings) > 0 {
		totalAccumulatedStr = strings.Join(accumulatedStrings, " | ")
	}

	refMonthName := getMonthNamePT(currentMonth)
	msg := p.Sprintf("💸 *Proventos: %s*\n", portfolioName)
	msg += p.Sprintf("💰 *Total Acumulado:* %s\n", totalAccumulatedStr)
	msg += p.Sprintf("📅 *Mês de Referência:* %s/%s\n\n", refMonthName, fmt.Sprintf("%d", currentYear))

	var paidStrings []string
	var paidCurrencies []string
	for curr := range totalPaidMonth {
		paidCurrencies = append(paidCurrencies, curr)
	}
	sortCurrencies(paidCurrencies)
	for _, curr := range paidCurrencies {
		paidStrings = append(paidStrings, p.Sprintf("%s %.2f", getCurrencySymbol(curr), totalPaidMonth[curr]))
	}

	if len(paidStrings) > 0 {
		msg += p.Sprintf("✅ *Recebidos no Mês:* %s\n", strings.Join(paidStrings, " | "))
	} else {
		msg += p.Sprintf("✅ *Recebidos no Mês:* R$ 0,00\n")
	}

	var futureStrings []string
	var futureCurrencies []string
	for curr := range totalFutureMonth {
		futureCurrencies = append(futureCurrencies, curr)
	}
	sortCurrencies(futureCurrencies)
	for _, curr := range futureCurrencies {
		futureStrings = append(futureStrings, p.Sprintf("%s %.2f", getCurrencySymbol(curr), totalFutureMonth[curr]))
	}

	if len(futureStrings) > 0 {
		msg += p.Sprintf("⏳ *A Receber no Mês:* %s\n", strings.Join(futureStrings, " | "))
	} else {
		msg += p.Sprintf("⏳ *A Receber no Mês:* R$ 0,00\n")
	}

	if len(divs) > 0 {
		var pastDivs []portfolio.CalculatedDividend
		var futureDivs []portfolio.CalculatedDividend
		for _, d := range divs {
			if !d.PaymentDate.After(now) {
				pastDivs = append(pastDivs, d)
			} else {
				futureDivs = append(futureDivs, d)
			}
		}

		if len(pastDivs) > 0 {
			msg += "\n📋 *Últimos Pagamentos*\n"
			sort.Slice(pastDivs, func(i, j int) bool {
				return pastDivs[i].PaymentDate.After(pastDivs[j].PaymentDate)
			})
			limit := 5
			if len(pastDivs) < 5 {
				limit = len(pastDivs)
			}
			for i := 0; i < limit; i++ {
				d := pastDivs[i]
				tipoStr := "Div"
				if d.Type != "" {
					tipoStr = d.Type
				}
				curr := d.Currency
				if curr == "" {
					curr = "BRL"
				}
				emoji := getAssetTypeEmoji(d.AssetType, d.Ticker)
				tickerClean := cleanTickerForDisplay(d.Ticker)
				tipoAbbr := abbreviateDividendType(tipoStr)

				msg += p.Sprintf("✅ %s `%s` • %s %.2f • %s\n", emoji, tickerClean, getCurrencySymbol(curr), d.NetAmount, d.PaymentDate.Format("2006-01-02"))
				if d.Quantity > 0 && d.PerShareAmount > 0 {
					msg += p.Sprintf("   ↳ _%s • %s un x %s %s_\n", tipoAbbr, formatQuantity(d.Quantity), getCurrencySymbol(curr), formatPerShareAmount(p, d.PerShareAmount))
				} else {
					msg += p.Sprintf("   ↳ _%s_\n", tipoAbbr)
				}
			}
		}

		if len(futureDivs) > 0 {
			msg += "\n📅 *Próximos Pagamentos*\n"
			sort.Slice(futureDivs, func(i, j int) bool {
				return futureDivs[i].PaymentDate.Before(futureDivs[j].PaymentDate)
			})
			limit := 5
			if len(futureDivs) < 5 {
				limit = len(futureDivs)
			}
			for i := 0; i < limit; i++ {
				d := futureDivs[i]
				tipoStr := "Div"
				if d.Type != "" {
					tipoStr = d.Type
				}
				curr := d.Currency
				if curr == "" {
					curr = "BRL"
				}
				emoji := getAssetTypeEmoji(d.AssetType, d.Ticker)
				tickerClean := cleanTickerForDisplay(d.Ticker)
				tipoAbbr := abbreviateDividendType(tipoStr)

				msg += p.Sprintf("⏳ %s `%s` • %s %.2f • %s\n", emoji, tickerClean, getCurrencySymbol(curr), d.NetAmount, d.PaymentDate.Format("2006-01-02"))
				if d.Quantity > 0 && d.PerShareAmount > 0 {
					msg += p.Sprintf("   ↳ _%s • %s un x %s %s_\n", tipoAbbr, formatQuantity(d.Quantity), getCurrencySymbol(curr), formatPerShareAmount(p, d.PerShareAmount))
				} else {
					msg += p.Sprintf("   ↳ _%s_\n", tipoAbbr)
				}
			}
		}
	} else {
		msg += "\nNenhum provento registrado na sua carteira ainda."
	}

	menu := &telebot.ReplyMarkup{}
	btnAno := menu.Data("📅 Agrupar por Ano", "btn_divs_year")
	btnMes := menu.Data("📆 Agrupar por Mês", "btn_divs_month")
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")

	if len(divs) > 0 {
		menu.Inline(menu.Row(btnAno, btnMes), menu.Row(btnBack))
	} else {
		menu.Inline(menu.Row(btnBack))
	}

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleDividendsByYear(c telebot.Context) error {
	defer c.Respond()
	divs, portfolioName, err := h.fetchDividends(c)
	if err != nil {
		return c.Edit("❌ Erro ao buscar proventos.")
	}

	grouped := make(map[int]map[string]float64)
	for _, d := range divs {
		y := d.PaymentDate.Year()
		curr := d.Currency
		if curr == "" {
			curr = "BRL"
		}
		if _, exists := grouped[y]; !exists {
			grouped[y] = make(map[string]float64)
		}
		grouped[y][curr] += d.NetAmount
	}

	years := make([]int, 0, len(grouped))
	for y := range grouped {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📅 *Proventos por Ano: %s*\n\n", portfolioName)
	for _, y := range years {
		var yearStrings []string
		currencies := make([]string, 0, len(grouped[y]))
		for c := range grouped[y] {
			currencies = append(currencies, c)
		}
		sortCurrencies(currencies)
		for _, curr := range currencies {
			yearStrings = append(yearStrings, p.Sprintf("%s %.2f", getCurrencySymbol(curr), grouped[y][curr]))
		}

		if y <= 1 {
			msg += p.Sprintf("• *A Definir*: %s\n", strings.Join(yearStrings, " | "))
		} else {
			msg += p.Sprintf("• *%s*: %s\n", fmt.Sprint(y), strings.Join(yearStrings, " | "))
		}
	}

	menu := &telebot.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Voltar", "btn_proventos")
	btnMenu := menu.Data("🏠 Menu", "btn_menu")
	menu.Inline(menu.Row(btnBack, btnMenu))

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleDividendsByMonth(c telebot.Context) error {
	defer c.Respond()
	divs, portfolioName, err := h.fetchDividends(c)
	if err != nil {
		return c.Edit("❌ Erro ao buscar proventos.")
	}

	grouped := make(map[string][]portfolio.CalculatedDividend)
	for _, d := range divs {
		key := d.PaymentDate.Format("2006-01") // Sortable key YYYY-MM
		if d.PaymentDate.IsZero() || d.PaymentDate.Year() <= 1 {
			key = "0000-00"
		}
		grouped[key] = append(grouped[key], d)
	}

	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	pageStr := c.Data()
	page := 0
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}

	pageSize := 3
	start := page * pageSize
	if start >= len(keys) {
		start = len(keys)
	}
	end := start + pageSize
	if end > len(keys) {
		end = len(keys)
	}

	if len(keys) == 0 {
		return c.Edit("📆 Nenhum provento encontrado.")
	}

	pageKeys := keys[start:end]

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📆 *Proventos por Mês: %s*\n_Página %d_\n\n", portfolioName, page+1)

	for _, k := range pageKeys {
		display := ""
		if k == "0000-00" {
			display = "A Definir"
		} else {
			display = k
		}

		totalMonthCurr := make(map[string]float64)

		type monthItem struct {
			day      int
			dateStr  string
			ticker   string
			dType    string
			currency string
			amount   float64
		}

		type itemKey struct {
			day      int
			ticker   string
			dType    string
			currency string
		}

		summaryMap := make(map[itemKey]float64)

		for _, d := range grouped[k] {
			curr := d.Currency
			if curr == "" {
				curr = "BRL"
			}
			totalMonthCurr[curr] += d.NetAmount

			dType := "Div"
			if d.Type != "" {
				dType = d.Type
			}

			day := 0
			if !d.PaymentDate.IsZero() && d.PaymentDate.Year() > 1 {
				day = d.PaymentDate.Day()
			}

			key := itemKey{
				day:      day,
				ticker:   d.Ticker,
				dType:    dType,
				currency: curr,
			}
			summaryMap[key] += d.NetAmount
		}

		var monthTotalStrings []string
		monthCurrencies := make([]string, 0, len(totalMonthCurr))
		for c := range totalMonthCurr {
			monthCurrencies = append(monthCurrencies, c)
		}
		sortCurrencies(monthCurrencies)
		for _, curr := range monthCurrencies {
			monthTotalStrings = append(monthTotalStrings, p.Sprintf("%s %.2f", getCurrencySymbol(curr), totalMonthCurr[curr]))
		}

		msg += p.Sprintf("• *%s*: %s\n", display, strings.Join(monthTotalStrings, " | "))

		var items []monthItem
		for key, amt := range summaryMap {
			dateStr := "-"
			if key.day > 0 {
				dateStr = fmt.Sprintf("%02d", key.day)
			}
			items = append(items, monthItem{
				day:      key.day,
				dateStr:  dateStr,
				ticker:   key.ticker,
				dType:    key.dType,
				currency: key.currency,
				amount:   amt,
			})
		}

		sort.Slice(items, func(i, j int) bool {
			if items[i].day != items[j].day {
				return items[i].day < items[j].day
			}
			if items[i].ticker != items[j].ticker {
				return items[i].ticker < items[j].ticker
			}
			if items[i].dType != items[j].dType {
				return items[i].dType < items[j].dType
			}
			return items[i].currency < items[j].currency
		})

		for _, item := range items {
			formattedDates := item.dateStr
			if item.dateStr != "-" {
				formattedDates = "Dia " + item.dateStr
			}
			msg += p.Sprintf("   ↳ `%s` (%s) • %s %.2f • %s\n", item.ticker, abbreviateDividendType(item.dType), getCurrencySymbol(item.currency), item.amount, formattedDates)
		}
		msg += "\n"
	}

	menu := &telebot.ReplyMarkup{}
	var btns []telebot.Btn

	if start > 0 {
		btns = append(btns, menu.Data("⬅️ Anterior", "btn_divs_month", fmt.Sprintf("%d", page-1)))
	}
	if end < len(keys) {
		btns = append(btns, menu.Data("Próxima ➡️", "btn_divs_month", fmt.Sprintf("%d", page+1)))
	}

	var rows []telebot.Row
	if len(btns) > 0 {
		rows = append(rows, menu.Row(btns...))
	}
	btnBack := menu.Data("⬅️ Voltar", "btn_proventos")
	btnMenu := menu.Data("🏠 Menu", "btn_menu")
	rows = append(rows, menu.Row(btnBack, btnMenu))
	menu.Inline(rows...)

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}
