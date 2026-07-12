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
	now := time.Now()
	currentMonth := now.Month()
	currentYear := now.Year()

	for _, d := range divs {
		if d.PaymentDate.Year() == currentYear && d.PaymentDate.Month() == currentMonth {
			currency := d.Currency
			if currency == "" {
				currency = "BRL"
			}
			if d.PaymentDate.After(now) {
				totalFutureMonth[currency] += d.NetAmount
			} else {
				totalPaidMonth[currency] += d.NetAmount
			}
		}
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("💸 *Proventos: %s*\n\n", portfolioName)

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
		msg += p.Sprintf("✅ *Recebidos (Mês Atual):* %s\n", strings.Join(paidStrings, " | "))
	} else {
		msg += p.Sprintf("✅ *Recebidos (Mês Atual):* R$ 0,00\n")
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
		msg += p.Sprintf("⏳ *A Receber (Mês Atual):* %s\n", strings.Join(futureStrings, " | "))
	} else {
		msg += p.Sprintf("⏳ *A Receber (Mês Atual):* R$ 0,00\n")
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
			limit := 3
			if len(pastDivs) < 3 {
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
				msg += p.Sprintf("✅ `%s` (%s) • %s %.2f • %s\n", d.Ticker, abbreviateDividendType(tipoStr), getCurrencySymbol(curr), d.NetAmount, d.PaymentDate.Format("02/01/2006"))
			}
		}

		if len(futureDivs) > 0 {
			msg += "\n📅 *Próximos Pagamentos*\n"
			sort.Slice(futureDivs, func(i, j int) bool {
				return futureDivs[i].PaymentDate.Before(futureDivs[j].PaymentDate)
			})
			limit := 3
			if len(futureDivs) < 3 {
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
				msg += p.Sprintf("⏳ `%s` (%s) • %s %.2f • %s\n", d.Ticker, abbreviateDividendType(tipoStr), getCurrencySymbol(curr), d.NetAmount, d.PaymentDate.Format("02/01/2006"))
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
			parts := strings.Split(k, "-")
			display = fmt.Sprintf("%s/%s", parts[1], parts[0]) // MM/YYYY format
		}

		totalMonthCurr := make(map[string]float64)

		type tSummary struct {
			amount   float64
			dates    []string
			dType    string
			currency string
		}
		summaryMap := make(map[string]*tSummary)

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

			mapKey := d.Ticker + "|" + dType + "|" + curr
			if _, exists := summaryMap[mapKey]; !exists {
				summaryMap[mapKey] = &tSummary{dType: dType, currency: curr}
			}
			summaryMap[mapKey].amount += d.NetAmount

			dateStr := d.PaymentDate.Format("02/01/2006")
			if d.PaymentDate.IsZero() || d.PaymentDate.Year() <= 1 {
				dateStr = "-"
			}
			foundDate := false
			for _, existing := range summaryMap[mapKey].dates {
				if existing == dateStr {
					foundDate = true
					break
				}
			}
			if !foundDate {
				summaryMap[mapKey].dates = append(summaryMap[mapKey].dates, dateStr)
			}
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

		mapKeys := make([]string, 0, len(summaryMap))
		for mk := range summaryMap {
			mapKeys = append(mapKeys, mk)
		}
		sort.Strings(mapKeys)

		for _, mk := range mapKeys {
			sum := summaryMap[mk]
			ticker := strings.Split(mk, "|")[0]
			datesStr := strings.Join(sum.dates, ", ")
			msg += p.Sprintf("   ↳ `%s` (%s) • %s %.2f • %s\n", ticker, abbreviateDividendType(sum.dType), getCurrencySymbol(sum.currency), sum.amount, datesStr)
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
