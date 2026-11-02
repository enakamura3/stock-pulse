package market

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Investidor10RawDividend contém os dados brutos extraídos da tabela HTML do Investidor10.
type Investidor10RawDividend struct {
	Type        string // "Dividendos", "JCP", "Rendimento", "Amortização"
	CumDate     string // "20/10/2026" (DD/MM/YYYY)
	PaymentDate string // "10/11/2026" (DD/MM/YYYY) ou "-"
	Amount      string // "1,00440000" (vírgula como separador decimal)
}

type Investidor10Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewInvestidor10Client() *Investidor10Client {
	return &Investidor10Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://investidor10.com.br",
	}
}

func (c *Investidor10Client) WithBaseURL(baseURL string) *Investidor10Client {
	c.baseURL = baseURL
	return c
}

func (c *Investidor10Client) WithHTTPClient(client *http.Client) *Investidor10Client {
	c.httpClient = client
	return c
}

// FetchDividends busca os proventos no Investidor10.
// Determina as rotas a tentar com base no assetType (ex: "etfs", "fiis", "acoes").
func (c *Investidor10Client) FetchDividends(ctx context.Context, ticker string, assetType string) ([]Investidor10RawDividend, error) {
	cleanTicker := strings.ToLower(strings.TrimSuffix(ticker, ".SA"))
	cleanTicker = strings.TrimSpace(cleanTicker)
	if cleanTicker == "" {
		return nil, fmt.Errorf("ticker inválido para investidor10")
	}

	var categories []string
	switch strings.ToUpper(assetType) {
	case "ETF_BR", "ETF":
		categories = []string{"etfs", "fiis", "acoes"}
	case "FII", "FIAGRO":
		categories = []string{"fiis", "acoes", "etfs"}
	case "BDR":
		categories = []string{"bdrs", "acoes", "etfs"}
	default: // STOCK_BR ou outros
		categories = []string{"acoes", "fiis", "etfs"}
	}

	var lastErr error
	for _, cat := range categories {
		reqURL := fmt.Sprintf("%s/%s/%s/", strings.TrimSuffix(c.baseURL, "/"), cat, cleanTicker)
		raw, err := c.fetchAndScrape(ctx, reqURL)
		if err == nil && len(raw) > 0 {
			return raw, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("nenhum provento encontrado no investidor10 para %s", ticker)
}

func (c *Investidor10Client) fetchAndScrape(ctx context.Context, reqURL string) ([]Investidor10RawDividend, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("investidor10 retornou status %d para %s", resp.StatusCode, reqURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawDividends []Investidor10RawDividend
	doc.Find("table#table-dividends-history tbody tr").Each(func(i int, sel *goquery.Selection) {
		tds := sel.Find("td")
		if tds.Length() >= 4 {
			tipoStr := strings.TrimSpace(tds.Eq(0).Text())
			cumDateStr := strings.TrimSpace(tds.Eq(1).Text())
			paymentDateStr := strings.TrimSpace(tds.Eq(2).Text())
			amountStr := strings.TrimSpace(tds.Eq(3).Text())

			if tipoStr != "" && cumDateStr != "" && amountStr != "" {
				rawDividends = append(rawDividends, Investidor10RawDividend{
					Type:        tipoStr,
					CumDate:     cumDateStr,
					PaymentDate: paymentDateStr,
					Amount:      amountStr,
				})
			}
		}
	})

	if len(rawDividends) == 0 {
		return nil, nil
	}

	return rawDividends, nil
}
