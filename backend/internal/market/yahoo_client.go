package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// YahooRawDividend contém os dados crus do Yahoo Finance Chart API.
type YahooRawDividend struct {
	Timestamp int64   // Unix timestamp (segundos)
	Amount    float64 // Já é float no JSON do Yahoo
}

type YahooClient struct {
	httpClient *http.Client
}

func NewYahooClient() *YahooClient {
	return &YahooClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type yahooDividendResponse struct {
	Chart struct {
		Result []struct {
			Events struct {
				Dividends map[string]struct {
					Amount float64 `json:"amount"`
					Date   int64   `json:"date"`
				} `json:"dividends"`
			} `json:"events"`
		} `json:"result"`
	} `json:"chart"`
}

// FetchDividends busca o histórico de dividendos via Yahoo Finance Chart API.
// Endpoint: query2.finance.yahoo.com/v8/finance/chart/{symbol}?events=div&range=10y&interval=1d
func (c *YahooClient) FetchDividends(ctx context.Context, symbol string) ([]YahooRawDividend, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?events=div&range=10y&interval=1d", symbol)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance api error: %d", resp.StatusCode)
	}

	var data yahooDividendResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var rawDividends []YahooRawDividend
	if len(data.Chart.Result) > 0 && data.Chart.Result[0].Events.Dividends != nil {
		for _, div := range data.Chart.Result[0].Events.Dividends {
			rawDividends = append(rawDividends, YahooRawDividend{
				Timestamp: div.Date,
				Amount:    div.Amount,
			})
		}
	}

	return rawDividends, nil
}
