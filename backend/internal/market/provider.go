package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

type DividendEvent struct {
	Date        time.Time `json:"date"`
	PaymentDate time.Time `json:"payment_date"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
}

// Quote representa os dados consolidados da cotação em tempo real de um ativo.
type Quote struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Volume        int64   `json:"volume"`
	Currency      string  `json:"currency"`
}

// SearchResult representa um resultado de busca autocomplete de ativo.
type SearchResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Type     string `json:"type"`
}

// QuoteProvider define o contrato para fornecer cotações, buscas de ativos e eventos corporativos.
type QuoteProvider interface {
	GetQuote(ctx context.Context, symbol string) (*Quote, error)
	SearchAssets(ctx context.Context, query string) ([]SearchResult, error)
	GetDividends(ctx context.Context, symbol string, assetType string) ([]DividendEvent, error)
}

// YahooFinanceProvider implementa QuoteProvider consumindo endpoints públicos do Yahoo Finance.
type YahooFinanceProvider struct {
	client *http.Client
}

// NewYahooFinanceProvider inicializa uma instância do provedor Yahoo.
func NewYahooFinanceProvider() *YahooFinanceProvider {
	return &YahooFinanceProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string  `json:"currency"`
				Symbol               string  `json:"symbol"`
				LongName             string  `json:"longName"`
				ShortName            string  `json:"shortName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
			} `json:"meta"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

type searchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		LongName  string `json:"longname"`
		ShortName string `json:"shortname"`
		Exchange  string `json:"exchange"`
		QuoteType string `json:"quoteType"`
	} `json:"quotes"`
}

// GetQuote obtém a cotação intradiária em tempo real usando o endpoint de Chart.
func (y *YahooFinanceProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", url.PathEscape(symbol))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// User-Agent simulado para evitar rate-limiting e 401 Unauthorized do Yahoo Finance
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api do yahoo finance retornou status %d", resp.StatusCode)
	}

	var data chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Chart.Error != nil {
		return nil, fmt.Errorf("erro retornado pelo provedor de mercado: %v", data.Chart.Error)
	}

	if len(data.Chart.Result) == 0 {
		return nil, errors.New("ativo não encontrado ou sem cotação recente")
	}

	meta := data.Chart.Result[0].Meta

	name := meta.LongName
	if name == "" {
		name = meta.ShortName
	}
	if name == "" {
		name = meta.Symbol
	}

	change := meta.RegularMarketPrice - meta.ChartPreviousClose
	changePercent := 0.0
	if meta.ChartPreviousClose > 0 {
		changePercent = (change / meta.ChartPreviousClose) * 100
	}

	quote := &Quote{
		Symbol:        meta.Symbol,
		Name:          name,
		Price:         meta.RegularMarketPrice,
		Change:        change,
		ChangePercent: changePercent,
		High:          meta.RegularMarketDayHigh,
		Low:           meta.RegularMarketDayLow,
		Volume:        meta.RegularMarketVolume,
		Currency:      meta.Currency,
	}

	return quote, nil
}

// SearchAssets realiza a busca textual autocomplete de ativos globais e da B3.
func (y *YahooFinanceProvider) SearchAssets(ctx context.Context, query string) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v1/finance/search?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api do yahoo finance retornou status %d", resp.StatusCode)
	}

	var data searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, q := range data.Quotes {
		// Ignora registros vazios de símbolos ou sem tipo de ativo
		if q.Symbol == "" || q.QuoteType == "" {
			continue
		}

		name := q.LongName
		if name == "" {
			name = q.ShortName
		}
		if name == "" {
			name = q.Symbol
		}

		results = append(results, SearchResult{
			Symbol:   q.Symbol,
			Name:     name,
			Exchange: q.Exchange,
			Type:     q.QuoteType,
		})
	}

	return results, nil
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

func (y *YahooFinanceProvider) GetDividends(ctx context.Context, symbol string, assetType string) ([]DividendEvent, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?events=div&range=10y&interval=1d", symbol)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := y.client.Do(req)
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

	var events []DividendEvent
	if len(data.Chart.Result) > 0 && data.Chart.Result[0].Events.Dividends != nil {
		for _, div := range data.Chart.Result[0].Events.Dividends {
			t := time.Unix(div.Date, 0)
			events = append(events, DividendEvent{
				Date:        t,
				PaymentDate: t, // Fallback to Ex-Date
				Amount:      div.Amount,
				Type:        "Dividendo",
			})
		}
	}
	
	// Sort by date ascending
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	return events, nil
}
