package market

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Investidor10DividendSource struct {
	client *Investidor10Client
}

func NewInvestidor10DividendSource(client *Investidor10Client) *Investidor10DividendSource {
	return &Investidor10DividendSource{client: client}
}

func (s *Investidor10DividendSource) Name() string {
	return "investidor10"
}

func (s *Investidor10DividendSource) SupportedAssetTypes() []string {
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR", "BDR"}
}

func (s *Investidor10DividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	rawDividends, err := s.client.FetchDividends(ctx, ticker, assetType)
	if err != nil {
		return nil, err
	}

	var events []DividendEvent
	layoutDate := "02/01/2006"

	for _, raw := range rawDividends {
		cumDate, err := time.Parse(layoutDate, raw.CumDate)
		if err != nil {
			continue
		}

		amountStr := strings.ReplaceAll(raw.Amount, ".", "")
		amountStr = strings.ReplaceAll(amountStr, ",", ".")
		amountStr = strings.TrimSpace(amountStr)
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		paymentDate := cumDate
		if raw.PaymentDate != "" && raw.PaymentDate != "-" && raw.PaymentDate != "n/a" {
			pd, err := time.Parse(layoutDate, raw.PaymentDate)
			if err == nil {
				paymentDate = pd
			}
		}

		cleanType := normalizeDividendType(raw.Type, assetType)

		events = append(events, DividendEvent{
			Date:        cumDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        cleanType,
		})
	}

	if len(events) == 0 {
		return nil, nil
	}

	// Ordena por data decrescente (mais recente primeiro)
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Date.After(events[j].Date)
	})

	return events, nil
}
