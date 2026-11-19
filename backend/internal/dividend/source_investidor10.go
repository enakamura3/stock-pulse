package dividend

import (
	"context"
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
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR", "BDR", "STOCK_US", "ETF_US"}
}

func (s *Investidor10DividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]Event, error) {
	rawDividends, err := s.client.FetchDividends(ctx, ticker, assetType)
	if err != nil {
		return nil, err
	}

	var events []Event
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

		var paymentDate time.Time
		if raw.PaymentDate != "" && raw.PaymentDate != "-" && raw.PaymentDate != "n/a" {
			pd, err := time.Parse(layoutDate, raw.PaymentDate)
			if err == nil {
				paymentDate = pd
			}
		}

		events = append(events, Event{
			Date:        cumDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        raw.Type,
		})
	}

	return events, nil
}
