package market

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type StockAnalysisScraper struct {
	client *http.Client
}

func NewStockAnalysisScraper() *StockAnalysisScraper {
	return &StockAnalysisScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}


func (s *StockAnalysisScraper) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
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

	resp, err := s.client.Do(req)
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

	var events []DividendEvent
	layout := "Jan 2, 2006"

	doc.Find("table tbody tr").Each(func(i int, sel *goquery.Selection) {
		tds := sel.Find("td")
		if tds.Length() < 4 {
			return
		}

		exDateStr := strings.TrimSpace(tds.Eq(2).Text()) // Record Date
		if exDateStr == "-" || exDateStr == "n/a" || exDateStr == "" {
			exDateStr = strings.TrimSpace(tds.Eq(0).Text()) // Fallback to Ex-Div Date
		}
		amountStr := strings.TrimSpace(tds.Eq(1).Text())
		payDateStr := strings.TrimSpace(tds.Eq(3).Text())

		exDate, err := time.Parse(layout, exDateStr)
		if err != nil {
			return
		}

		amtStr := strings.ReplaceAll(amountStr, "$", "")
		amtStr = strings.ReplaceAll(amtStr, " BRL", "")
		amtStr = strings.TrimSpace(amtStr)
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return
		}

		paymentDate := exDate
		if payDateStr != "-" && payDateStr != "n/a" && payDateStr != "" {
			pd, err := time.Parse(layout, payDateStr)
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
	})

	if len(events) == 0 {
		return nil, fmt.Errorf("dados de histórico não encontrados ou vazios na tabela do stockanalysis")
	}

	return events, nil
}
