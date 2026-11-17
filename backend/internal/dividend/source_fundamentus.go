package dividend

import (
	"context"
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

func (s *FundamentusDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]Event, error) {
	rawDividends, _, err := s.client.FetchDividends(ctx, ticker)
	if err != nil {
		return nil, err
	}

	var events []Event
	layoutDate := "02/01/2006"

	for _, raw := range rawDividends {
		cumDate, err := time.Parse(layoutDate, raw.Date)
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

		events = append(events, Event{
			Date:        cumDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        raw.Type,
		})
	}

	return events, nil
}
