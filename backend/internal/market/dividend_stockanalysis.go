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
	return []string{"STOCK_BR", "ETF_BR", "BDR", "STOCK_US", "ETF_US"}
}

func (s *StockAnalysisDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	rawDividends, err := s.client.FetchDividends(ctx, ticker, assetType)
	if err != nil {
		return nil, err
	}

	var events []DividendEvent
	layout := "Jan 2, 2006"

	for _, raw := range rawDividends {
		exDateStr := raw.RecordDate
		if exDateStr == "-" || exDateStr == "n/a" || exDateStr == "" {
			exDateStr = raw.ExDivDate // Fallback to Ex-Div Date
		}

		exDate, err := time.Parse(layout, exDateStr)
		if err != nil {
			continue
		}

		amtStr := strings.ReplaceAll(raw.Amount, "$", "")
		amtStr = strings.ReplaceAll(amtStr, " BRL", "")
		amtStr = strings.TrimSpace(amtStr)
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			continue
		}

		paymentDate := exDate
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
			Date:        exDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        cleanType,
		})
	}

	return events, nil
}
