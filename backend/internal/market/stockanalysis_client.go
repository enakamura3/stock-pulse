package market

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// StockAnalysisRawDividend contém os dados crus da tabela HTML do StockAnalysis.
type StockAnalysisRawDividend struct {
	ExDivDate  string // "Jan 2, 2006" (formato US)
	Amount     string // "$1.25" ou "1.25 BRL"
	RecordDate string // "Jan 5, 2006", "-", "n/a", ou ""
	PayDate    string // "Feb 1, 2006"
}

type StockAnalysisClient struct {
	httpClient *http.Client
}

func NewStockAnalysisClient() *StockAnalysisClient {
	return &StockAnalysisClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchDividends busca a tabela de dividendos do StockAnalysis.
// A URL é escolhida com base no ticker e assetType:
//   - .SA       → stockanalysis.com/quote/bvmf/{symbol}/dividend/
//   - ETF_US    → stockanalysis.com/etf/{symbol}/dividend/
//   - STOCK_US  → stockanalysis.com/stocks/{symbol}/dividend/
func (c *StockAnalysisClient) FetchDividends(ctx context.Context, ticker string, assetType string) ([]StockAnalysisRawDividend, error) {
	var url string
	if strings.HasSuffix(strings.ToUpper(ticker), ".SA") {
		symbol := strings.ToLower(strings.TrimSuffix(ticker, ".SA"))
		url = fmt.Sprintf("https://stockanalysis.com/quote/bvmf/%s/dividend/", symbol)
	} else {
		symbol := strings.ToLower(ticker)
		basePath := "stocks"
		if strings.HasPrefix(strings.ToUpper(assetType), "ETF") {
			basePath = "etf"
		}
		url = fmt.Sprintf("https://stockanalysis.com/%s/%s/dividend/", basePath, symbol)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stockanalysis retornou status %d para o ativo %s", resp.StatusCode, ticker)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var rawDividends []StockAnalysisRawDividend

	doc.Find("table tbody tr").Each(func(i int, sel *goquery.Selection) {
		tds := sel.Find("td")
		if tds.Length() < 4 {
			return
		}

		exDivDate := strings.TrimSpace(tds.Eq(0).Text())
		amount := strings.TrimSpace(tds.Eq(1).Text())
		recordDate := strings.TrimSpace(tds.Eq(2).Text())
		payDate := strings.TrimSpace(tds.Eq(3).Text())

		rawDividends = append(rawDividends, StockAnalysisRawDividend{
			ExDivDate:  exDivDate,
			Amount:     amount,
			RecordDate: recordDate,
			PayDate:    payDate,
		})
	})

	if len(rawDividends) == 0 {
		return nil, fmt.Errorf("dados de histórico não encontrados ou vazios na tabela do stockanalysis")
	}

	return rawDividends, nil
}
