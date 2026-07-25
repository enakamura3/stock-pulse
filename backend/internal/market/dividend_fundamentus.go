package market

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FundamentusDividendSource struct {
	client *FundamentusClient
}

func NewFundamentusDividendSource(client *FundamentusClient) *FundamentusDividendSource {
	return &FundamentusDividendSource{client: client}
}

func (s *FundamentusDividendSource) Name() string {
	return "fundamentus"
}

func (s *FundamentusDividendSource) SupportedAssetTypes() []string {
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR", "BDR"}
}

func (s *FundamentusDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	rawDividends, layoutResult, err := s.client.FetchDividends(ctx, ticker)
	if err != nil {
		return nil, err
	}

	var events []DividendEvent
	layoutDate := "02/01/2006"

	for _, raw := range rawDividends {
		exDate, err := time.Parse(layoutDate, raw.Date)
		if err != nil {
			continue
		}

		amountStr := strings.ReplaceAll(raw.Amount, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		var paymentDate time.Time
		if raw.PaymentDate != "-" && raw.PaymentDate != "" {
			pd, err := time.Parse(layoutDate, raw.PaymentDate)
			if err == nil {
				paymentDate = pd
			}
		}

		// Formatar o Tipo para um padrão limpo
		tipoUpper := strings.ToUpper(raw.Type)
		cleanType := "Dividendo"
		if strings.Contains(tipoUpper, "JRS CAP PROPRIO") || strings.Contains(tipoUpper, "JCP") {
			cleanType = "JCP"
		} else if strings.Contains(tipoUpper, "RENDIMENTO") {
			cleanType = "Rendimento"
		} else if strings.Contains(tipoUpper, "AMORTIZACAO") {
			cleanType = "Amortização"
		}

		events = append(events, DividendEvent{
			Date:        exDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        cleanType,
		})
	}

	// Deduplicar eventos baseados na combinação exata pedida:
	deduped := make([]DividendEvent, 0, len(events))
	seen := make(map[string]bool)

	for _, ev := range events {
		var key string
		if layoutResult == "fii" {
			// Regra FII: Apenas 1 pagamento por mês garantido.
			key = fmt.Sprintf("fii|%s|%02d|%d", ev.Type, ev.Date.Month(), ev.Date.Year())
		} else {
			// Regra Ações: Podem ter múltiplos pagamentos no mesmo mês.
			key = fmt.Sprintf("acao|%s|%.6f|%02d|%d", ev.Type, ev.Amount, ev.Date.Month(), ev.Date.Year())
		}

		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, ev)
		}
	}

	return deduped, nil
}
