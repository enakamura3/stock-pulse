package dividend

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type StockAnalysisDividendSource struct {
	client *StockAnalysisClient
}

func NewStockAnalysisDividendSource(client *StockAnalysisClient) *StockAnalysisDividendSource {
	return &StockAnalysisDividendSource{client: client}
}

func (s *StockAnalysisDividendSource) Name() string {
	return "stockanalysis"
}

func (s *StockAnalysisDividendSource) SupportedAssetTypes() []string {
	return []string{"STOCK_US", "ETF_US"}
}

func (s *StockAnalysisDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]Event, error) {
	rawDividends, err := s.client.FetchDividends(ctx, ticker, assetType)
	if err != nil {
		return nil, err
	}

	var events []Event
	layout := "Jan 2, 2006"

	for _, raw := range rawDividends {
		exDate, err := time.Parse(layout, raw.ExDivDate)
		if err != nil {
			continue
		}

		// Calcula a Data Com (CumDate) recuando para o dia útil anterior à Ex-Dividend Date,
		// garantindo consistência exata com o calendário da NYSE/Nasdaq e Investidor10.
		cumDate := PreviousBusinessDay(exDate)

		amtStr := strings.ReplaceAll(raw.Amount, "$", "")
		amtStr = strings.ReplaceAll(amtStr, " BRL", "")
		amtStr = strings.TrimSpace(amtStr)
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			continue
		}

		var paymentDate time.Time
		if raw.PayDate != "-" && raw.PayDate != "n/a" && raw.PayDate != "" {
			pd, err := time.Parse(layout, raw.PayDate)
			if err == nil {
				paymentDate = pd
			}
		}

		events = append(events, Event{
			Date:        cumDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        "Dividendo",
		})
	}

	return events, nil
}
