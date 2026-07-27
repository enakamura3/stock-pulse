package market

import (
	"context"
	"sort"
	"time"
)

type YahooDividendSource struct {
	client *YahooClient
}

func NewYahooDividendSource(client *YahooClient) *YahooDividendSource {
	return &YahooDividendSource{client: client}
}

func (s *YahooDividendSource) Name() string {
	return "yahoo"
}

func (s *YahooDividendSource) SupportedAssetTypes() []string {
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR", "BDR", "STOCK_US", "ETF_US", "CRYPTO"}
}

func (s *YahooDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	queryTicker := ticker
	isBR := assetType == "STOCK_BR" || assetType == "FII" || assetType == "FIAGRO" || assetType == "ETF_BR" || assetType == "BDR"
	if isBR && len(ticker) > 0 && ticker[len(ticker)-3:] != ".SA" && ticker[len(ticker)-3:] != ".sa" {
		queryTicker += ".SA"
	}

	rawDividends, err := s.client.FetchDividends(ctx, queryTicker)
	if err != nil {
		return nil, err
	}

	var events []DividendEvent

	for _, raw := range rawDividends {
		t := time.Unix(raw.Timestamp, 0)
		events = append(events, DividendEvent{
			Date:        t,
			PaymentDate: t, // Fallback to Ex-Date
			Amount:      raw.Amount,
			Type:        "Dividendo",
		})
	}

	// Sort by date ascending
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	return events, nil
}
