package market

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// BenchmarkItem representa um índice ou ativo de referência de mercado.
type BenchmarkItem struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Value         float64 `json:"value"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	PreviousClose float64 `json:"previous_close,omitempty"`
}

// MarketBenchmarks consolida os principais índices de mercado para comparação intradiária.
type MarketBenchmarks struct {
	IBOV      *BenchmarkItem `json:"ibov,omitempty"`
	SP500     *BenchmarkItem `json:"sp500,omitempty"`
	USDBRL    *BenchmarkItem `json:"usd_brl,omitempty"`
	IFIX      *BenchmarkItem `json:"ifix,omitempty"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// GetBenchmarks busca os principais benchmarks de mercado consolidando cotações em tempo real com cache Redis.
func (s *Service) GetBenchmarks(ctx context.Context) (*MarketBenchmarks, error) {
	cacheKey := "benchmarks:summary:v1"

	// Tenta resgatar do Redis (Cache Hit)
	if s.rdb != nil {
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil && val != "" {
			var cached MarketBenchmarks
			if err := json.Unmarshal([]byte(val), &cached); err == nil {
				return &cached, nil
			}
		}
	}

	// Busca os índices em paralelo
	benchmarks := &MarketBenchmarks{
		UpdatedAt: time.Now().UTC(),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	targets := []struct {
		key    string
		symbol string
		name   string
	}{
		{"IBOV", "^BVSP", "Ibovespa"},
		{"SP500", "^GSPC", "S&P 500"},
		{"USDBRL", "BRL=X", "Dólar Comercial"},
		{"IFIX", "IFIX.SA", "IFIX"},
	}

	for _, t := range targets {
		wg.Add(1)
		go func(targetKey, symbol, fallbackName string) {
			defer wg.Done()
			q, err := s.provider.GetQuote(ctx, symbol)
			if err != nil || q == nil {
				return
			}
			name := q.Name
			if name == "" || name == symbol {
				name = fallbackName
			}
			item := &BenchmarkItem{
				Symbol:        q.Symbol,
				Name:          name,
				Value:         q.Price,
				Change:        q.Change,
				ChangePercent: q.ChangePercent,
				PreviousClose: q.PreviousClose,
			}
			mu.Lock()
			switch targetKey {
			case "IBOV":
				benchmarks.IBOV = item
			case "SP500":
				benchmarks.SP500 = item
			case "USDBRL":
				benchmarks.USDBRL = item
			case "IFIX":
				benchmarks.IFIX = item
			}
			mu.Unlock()
		}(t.key, t.symbol, t.name)
	}

	wg.Wait()

	// Serializa e salva no Redis
	if s.rdb != nil {
		if data, err := json.Marshal(benchmarks); err == nil {
			s.rdb.Set(ctx, cacheKey, data, s.ttlQuotes)
		}
	}

	return benchmarks, nil
}
