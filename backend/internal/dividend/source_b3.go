package dividend

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

type B3DividendSource struct {
	client          *B3Client
	tickerNameCache map[string]string
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
	return []string{"STOCK_BR", "FII", "FIAGRO", "ETF_BR", "BDR"}
}

func (s *B3DividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]Event, error) {
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

	var events []Event
	layout := "02/01/2006"

	for _, raw := range rawDividends {
		cumDate, err := time.Parse(layout, raw.LastDatePriorEx)
		if err != nil {
			continue
		}

		amountStr := strings.Replace(raw.ValueCash, ",", ".", 1)
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		var paymentDate time.Time
		if raw.PaymentDate != "" && raw.PaymentDate != "-" {
			pd, err := time.Parse(layout, raw.PaymentDate)
			if err == nil {
				paymentDate = pd
			}
		}

		events = append(events, Event{
			Date:        cumDate,
			PaymentDate: paymentDate,
			Amount:      amount,
			Type:        raw.CorporateAction,
		})
	}

	return events, nil
}

func (s *B3DividendSource) resolveTradingName(ctx context.Context, ticker string) (string, error) {
	cleanTicker := strings.TrimSuffix(strings.ToUpper(ticker), ".SA")

	s.mu.Lock()
	defer s.mu.Unlock()

	if name, ok := s.tickerNameCache[cleanTicker]; ok {
		return name, nil
	}

	_, err := s.client.FetchCompanies(ctx)
	if err != nil {
		return "", err
	}

	s.tickerNameCache[cleanTicker] = cleanTicker
	return cleanTicker, nil
}
