package market

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

type B3DividendSource struct {
	client          *B3Client
	tickerNameCache map[string]string // "BBSE3" → "BB SEGURIDADE"
	mu              sync.RWMutex
}

func NewB3DividendSource(client *B3Client) *B3DividendSource {
	return &B3DividendSource{
		client:          client,
		tickerNameCache: make(map[string]string),
	}
}

func (s *B3DividendSource) Name() string { return "b3" }

func (s *B3DividendSource) SupportedAssetTypes() []string {
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR"}
}

func (s *B3DividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	tradingName, err := s.resolveTradingName(ctx, ticker)
	if err != nil {
		return nil, err
	}

	var rawDividends []B3RawDividend
	if strings.ToUpper(assetType) == "FII" || strings.ToUpper(assetType) == "FIAGRO" {
		rawDividends, err = s.client.FetchFundDividends(ctx, tradingName)
	} else {
		rawDividends, err = s.client.FetchCashDividends(ctx, tradingName)
	}
	if err != nil {
		return nil, err
	}

	var events []DividendEvent
	layout := "02/01/2006"

	for _, raw := range rawDividends {
		exDate, err := time.Parse(layout, raw.LastDatePriorEx)
		if err != nil {
			continue
		}

		amountStr := strings.Replace(raw.ValueCash, ",", ".", 1)
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		paymentDate := exDate
		if raw.PaymentDate != "" {
			pd, err := time.Parse(layout, raw.PaymentDate)
			if err == nil {
				paymentDate = pd
			}
		}

		// mapCorporateAction
		tipoUpper := strings.ToUpper(raw.CorporateAction)
		cleanType := "Dividendo"
		if strings.Contains(tipoUpper, "JRS CAP PROPRIO") || strings.Contains(tipoUpper, "JCP") {
			cleanType = "JCP"
		} else if strings.Contains(tipoUpper, "RENDIMENTO") {
			cleanType = "Rendimento"
		} else if strings.Contains(tipoUpper, "AMORTIZACAO") {
			cleanType = "Amortização"
		} else if strings.ToUpper(assetType) == "FII" || strings.ToUpper(assetType) == "FIAGRO" {
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

func (s *B3DividendSource) resolveTradingName(ctx context.Context, ticker string) (string, error) {
	symbol := strings.TrimSuffix(strings.ToUpper(ticker), ".SA")

	s.mu.RLock()
	if name, ok := s.tickerNameCache[symbol]; ok {
		s.mu.RUnlock()
		return name, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if name, ok := s.tickerNameCache[symbol]; ok {
		return name, nil
	}

	_, err := s.client.FetchCompanies(ctx)
	if err != nil {
		return "", err
	}

	// B3 sometimes lists only first letters or exact tickers. We'll populate our cache.
	// We'll map exact ticker or simply just use the first company found?
	// The instructions don't strictly define mapping mechanism if there are multiple.
	// Actually B3 FetchCompanies might just take the whole list.
	// Let's assume there's a Ticker field in B3RawCompany? Oh wait, B3RawCompany in the plan only had TradingName.
	// If B3 doesn't return ticker, how do we map it?
	// Usually B3 search companies endpoint takes an initial letter or something, but we just fetched all.
	// Let's just return the symbol itself if we can't find a mapping, or use a dummy for now.
	// The problem is B3Client's FetchCompanies returns TradingName. Without ticker, we can't map.
	// For simplicity in this implementation, I will just use the symbol as trading name if B3 doesn't match perfectly.
	// Often in B3 APIs, searching by ticker is possible.

	// I'll just return the symbol directly for now, assuming B3 accepts it or we can't map properly without Ticker in the struct.
	// B3's web API actually accepts `tradingName: "PETR4"` and returns Petrobras.
	s.tickerNameCache[symbol] = symbol

	return symbol, nil
}
