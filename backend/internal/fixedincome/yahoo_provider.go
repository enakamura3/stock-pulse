package fixedincome

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// YahooFinanceIndexProvider implementa IndexProvider consumindo dados da API pública do Yahoo Finance
type YahooFinanceIndexProvider struct {
	client *http.Client
}

// NewYahooFinanceIndexProvider inicializa o provedor Yahoo
func NewYahooFinanceIndexProvider() *YahooFinanceIndexProvider {
	return &YahooFinanceIndexProvider{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchRates busca cotações históricas de índices (IFIX, IBOV, S&P 500)
func (p *YahooFinanceIndexProvider) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	ticker := p.mapIndexerToTicker(indexer)
	if ticker == "" {
		return nil, fmt.Errorf("yahoo: indexador nao suportado: %s", indexer)
	}

	period1 := startDate.Unix()
	period2 := endDate.Unix()

	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d", url.PathEscape(ticker), period1, period2)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo api retornou status %d", resp.StatusCode)
	}

	var data struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error interface{} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Chart.Error != nil {
		return nil, fmt.Errorf("erro no provedor yahoo: %v", data.Chart.Error)
	}

	if len(data.Chart.Result) == 0 {
		return nil, errors.New("yahoo: resultado historico vazio")
	}

	res := data.Chart.Result[0]
	if len(res.Timestamp) == 0 || len(res.Indicators.Quote) == 0 {
		return nil, errors.New("yahoo: serie historica sem timestamps ou quotes")
	}

	closes := res.Indicators.Quote[0].Close
	if len(res.Timestamp) != len(closes) {
		return nil, errors.New("yahoo: inconsistencia de tamanho nos dados historicos")
	}

	var rates []IndexRate
	for i := range res.Timestamp {
		if closes[i] == nil {
			continue // Ignora dias sem cotação
		}
		dateVal := time.Unix(res.Timestamp[i], 0).UTC()
		// Arredonda para 00:00:00 da data
		dateVal = time.Date(dateVal.Year(), dateVal.Month(), dateVal.Day(), 0, 0, 0, 0, time.UTC)

		rates = append(rates, IndexRate{
			Indexer: indexer,
			Date:    dateVal,
			Rate:    *closes[i],
		})
	}

	return rates, nil
}

func (p *YahooFinanceIndexProvider) mapIndexerToTicker(indexer string) string {
	switch strings.ToUpper(indexer) {
	case "IFIX":
		return "IFIX.SA"
	case "IBOV":
		return "^BVSP"
	case "SP500":
		return "^GSPC"
	default:
		return ""
	}
}
