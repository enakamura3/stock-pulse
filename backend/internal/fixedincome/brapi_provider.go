package fixedincome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// BrapiProvider implementa IndexProvider consumindo dados do brapi.dev
type BrapiProvider struct {
	client *http.Client
	apiKey string
}

// NewBrapiProvider inicializa o provedor BRAPI
func NewBrapiProvider() *BrapiProvider {
	return &BrapiProvider{
		client: &http.Client{Timeout: 15 * time.Second},
		apiKey: os.Getenv("BRAPI_TOKEN"),
	}
}

type brapiResponse struct {
	Results []struct {
		Symbol              string `json:"symbol"`
		HistoricalDataPrice []struct {
			Date  int64   `json:"date"` // Unix timestamp
			Close float64 `json:"close"`
		} `json:"historicalDataPrice"`
	} `json:"results"`
}

// FetchRates busca cotações históricas do IFIX ou Ibovespa na BRAPI
func (p *BrapiProvider) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	// Se o período solicitado for maior que 3 meses, forçamos o erro para acionar o fallback do Yahoo Finance (limite de 3 meses da BRAPI gratuita)
	threeMonthsAgo := time.Now().AddDate(0, -3, -5)
	if startDate.Before(threeMonthsAgo) {
		return nil, fmt.Errorf("brapi: periodo solicitado de %s e posterior ao limite de 3 meses do plano gratuito", startDate.Format("2006-01-02"))
	}

	ticker := p.mapIndexerToTicker(indexer)
	if ticker == "" {
		return nil, fmt.Errorf("brapi: indexador nao suportado: %s", indexer)
	}

	apiURL := fmt.Sprintf("https://brapi.dev/api/quote/%s?range=3mo&interval=1d", url.PathEscape(ticker))
	if p.apiKey != "" {
		apiURL = apiURL + "&token=" + p.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brapi api retornou status %d", resp.StatusCode)
	}

	var data brapiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data.Results) == 0 {
		return nil, fmt.Errorf("brapi: nenhum resultado para o ticker %s", ticker)
	}

	var rates []IndexRate
	for _, result := range data.Results {
		// Tolerância para comparar com ou sem sufixo
		if !strings.EqualFold(strings.TrimSuffix(result.Symbol, ".SA"), strings.TrimSuffix(ticker, ".SA")) && !strings.EqualFold(result.Symbol, ticker) {
			continue
		}
		for _, day := range result.HistoricalDataPrice {
			dateVal := time.Unix(day.Date, 0).UTC()
			// Respeita os limites de data da requisição
			if dateVal.Before(startDate) || dateVal.After(endDate) {
				continue
			}
			rates = append(rates, IndexRate{
				Indexer: indexer,
				Date:    dateVal,
				Rate:    day.Close,
			})
		}
	}

	return rates, nil
}

func (p *BrapiProvider) mapIndexerToTicker(indexer string) string {
	switch strings.ToUpper(indexer) {
	case "IFIX":
		return "IFIX.SA"
	case "IBOV":
		// A BRAPI usa o ticker nativo "IBOV", não "^BVSP" (que é formato Yahoo Finance)
		return "IBOV"
	default:
		return ""
	}
}
