package market

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
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR", "BDR", "STOCK_US", "ETF_US"}
}

func (s *StockAnalysisDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	rawDividends, err := s.client.FetchDividends(ctx, ticker, assetType)
	if err != nil {
		return nil, err
	}

	var events []DividendEvent
	layout := "Jan 2, 2006"

	for _, raw := range rawDividends {
		dateStr := raw.RecordDate
		usedExDate := false
		if dateStr == "-" || dateStr == "n/a" || dateStr == "" {
			dateStr = raw.ExDivDate // Fallback to Ex-Div Date
			usedExDate = true
		}

		cumDate, err := time.Parse(layout, dateStr)
		if err != nil {
			continue
		}

		// A normalização da Data Ex: a Data Com é o dia anterior à Data Ex.
		// Se já pegamos a RecordDate, ela já é a Data Com. Se usamos a Ex-Div Date, subtraímos 1 dia.
		if usedExDate {
			cumDate = cumDate.Add(-24 * time.Hour)
		}

		amtStr := strings.ReplaceAll(raw.Amount, "$", "")
		amtStr = strings.ReplaceAll(amtStr, " BRL", "")
		amtStr = strings.TrimSpace(amtStr)
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			continue
		}

		paymentDate := cumDate
		if raw.PayDate != "-" && raw.PayDate != "n/a" && raw.PayDate != "" {
			pd, err := time.Parse(layout, raw.PayDate)
			if err == nil {
				paymentDate = pd
			}
		}

		cleanType := "Dividendo"
		upperType := strings.ToUpper(assetType)
		if upperType == "FII" || upperType == "FIAGRO" {
			cleanType = "Rendimento"
		}

		events = append(events, DividendEvent{
			Date:        cumDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        cleanType,
		})
	}

	return events, nil
}
