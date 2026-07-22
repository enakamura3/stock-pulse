package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Service gerencia cotações de ativos agregando cacheamento Redis de alta performance.
type Service struct {
	provider     QuoteProvider
	scraper      *Scraper
	fundamentus  *FundamentusScraper
	stockAnalysis *StockAnalysisScraper
	rdb          *redis.Client
	ttlQuotes          time.Duration
	ttlDividends       time.Duration
	ttlFundamentals    time.Duration
	ttlExchangeRates   time.Duration
}

// NewService cria uma nova instância de Service com TTLs configuráveis.
func NewService(provider QuoteProvider, rdb *redis.Client) *Service {
	getDuration := func(key string, defaultVal time.Duration) time.Duration {
		if val := os.Getenv(key); val != "" {
			if d, err := time.ParseDuration(val); err == nil {
				return d
			}
		}
		return defaultVal
	}

	return &Service{
		provider:         provider,
		scraper:          NewScraper(),
		fundamentus:      NewFundamentusScraper(),
		stockAnalysis:    NewStockAnalysisScraper(),
		rdb:              rdb,
		ttlQuotes:        getDuration("REDIS_TTL_QUOTES", 60*time.Second),
		ttlDividends:     getDuration("REDIS_TTL_DIVIDENDS", 12*time.Hour),
		ttlFundamentals:  getDuration("REDIS_TTL_FUNDAMENTALS", 12*time.Hour),
		ttlExchangeRates: getDuration("REDIS_TTL_EXCHANGE_RATES", 12*time.Hour),
	}
}

// GetQuoteWithCacheStatus resgata a cotação e indica se foi hit ou miss no cache.
func (s *Service) GetQuoteWithCacheStatus(ctx context.Context, symbol string) (*Quote, bool, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, false, fmt.Errorf("símbolo do ativo inválido")
	}

	key := fmt.Sprintf("quote:%s", symbol)

	// Tenta buscar no Redis (Cache Hit)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil && val != "" {
		var cachedQuote Quote
		if err := json.Unmarshal([]byte(val), &cachedQuote); err == nil {
			log.Printf("[Redis] CACHE HIT para o ativo %s", symbol)
			return &cachedQuote, true, nil
		}
	}

	// Se deu erro ou cache miss, busca do provedor externo
	log.Printf("[Redis] CACHE MISS para o ativo %s. Consultando provedor...", symbol)
	quote, err := s.provider.GetQuote(ctx, symbol)
	if err != nil {
		return nil, false, err
	}

	// Serializa e salva no Redis
	quoteJSON, err := json.Marshal(quote)
	if err == nil {
		err = s.rdb.Set(ctx, key, quoteJSON, s.ttlQuotes).Err()
		if err != nil {
			log.Printf("[Redis] Erro ao salvar cache para %s: %v", symbol, err)
		} else {
			log.Printf("[Redis] Novo cache salvo para %s com sucesso", symbol)
		}
	}

	return quote, false, nil
}

// GetQuote faz o wrapper para manter retrocompatibilidade com interfaces existentes.
func (s *Service) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	q, _, err := s.GetQuoteWithCacheStatus(ctx, symbol)
	return q, err
}

// GetDividends busca os proventos de um ativo e faz cache.
func (s *Service) GetDividends(ctx context.Context, symbol string, assetType string) ([]DividendEvent, error) {
	cacheKey := fmt.Sprintf("dividends:%s", symbol)
	
	// Tenta no Redis primeiro
	val, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached []DividendEvent
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			log.Printf("[Redis] CACHE HIT proventos para %s", symbol)
			return cached, nil
		}
	}

	log.Printf("[Redis] CACHE MISS proventos para %s. Consultando provedor...", symbol)

	// Roteamento: tenta buscar com o scraper correto
	var events []DividendEvent
	var fetchErr error

	if strings.HasSuffix(strings.ToUpper(symbol), ".SA") {
		// Busca de ambas as fontes para obter dados mais completos e precisos, evitando atrasos e truncamentos.
		saEvents, saErr := s.stockAnalysis.GetDividends(ctx, symbol, assetType)
		fundEvents, fundErr := s.fundamentus.GetDividends(ctx, symbol, assetType)

		if saErr != nil && fundErr != nil {
			fetchErr = fmt.Errorf("ambos os scrapers falharam: sa_err=%v, fund_err=%v", saErr, fundErr)
		} else {
			// Prioriza StockAnalysis (passando saEvents primeiro) pela precisão centesimal do valor do provento
			events = mergeAndDedupDividends(saEvents, fundEvents, assetType)
		}
	} else {
		events, fetchErr = s.stockAnalysis.GetDividends(ctx, symbol, assetType)
	}

	// Fallback para Yahoo Finance caso dê erro
	if fetchErr != nil || len(events) == 0 {
		log.Printf("[Market] Falha no scraper de proventos para %s (%v). Usando fallback do Yahoo Finance.", symbol, fetchErr)
		events, err = s.provider.GetDividends(ctx, symbol, assetType)
		if err != nil {
			return nil, err
		}
	}

	// Cacheia proventos
	if data, err := json.Marshal(events); err == nil {
		s.rdb.Set(ctx, cacheKey, data, s.ttlDividends)
	}

	return events, nil
}

func mergeAndDedupDividends(saEvents, fundEvents []DividendEvent, assetType string) []DividendEvent {
	isFii := strings.ToUpper(assetType) == "FII" || strings.ToUpper(assetType) == "FIAGRO"

	var baseEvents []DividendEvent
	var secondaryEvents []DividendEvent

	if isFii {
		baseEvents = saEvents
		secondaryEvents = fundEvents
	} else {
		// Prioriza Fundamentus para Ações, para manter JCP e valores brutos corretos
		baseEvents = fundEvents
		secondaryEvents = saEvents
	}

	deduped := append([]DividendEvent{}, baseEvents...)

	for _, sEv := range secondaryEvents {
		exists := false
		for _, dEv := range deduped {
			if isFii {
				if sEv.Date.Month() == dEv.Date.Month() && sEv.Date.Year() == dEv.Date.Year() {
					exists = true
					break
				}
			} else {
				// Para Ações: se o Fundamentus (base) já reportou QUALQUER provento nesta Data Com,
				// ignoramos o evento do StockAnalysis (secundário). O StockAnalysis costuma agrupar
				// JCP + Dividendo do mesmo dia num único valor, o que quebra a conciliação.
				if sEv.Date.Equal(dEv.Date) {
					exists = true
					break
				}
			}
		}
		if !exists {
			deduped = append(deduped, sEv)
		}
	}

	for i := range deduped {
		if isFii && deduped[i].Type == "Dividendo" {
			deduped[i].Type = "Rendimento"
		}
	}

	sort.SliceStable(deduped, func(i, j int) bool {
		return deduped[i].Date.After(deduped[j].Date)
	})

	return deduped
}

// getExchangeRatesMap fetches the 10y history of BRL=X and returns it as a map[string]float64 (date string "YYYY-MM-DD" -> rate).
// It caches the entire map in Redis for 12 hours.
func (s *Service) getExchangeRatesMap(ctx context.Context) (map[string]float64, error) {
	cacheKey := "fx:BRL=X:10y"
	val, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var rates map[string]float64
		if err := json.Unmarshal([]byte(val), &rates); err == nil {
			return rates, nil
		}
	}

	// Fetch from Yahoo Finance
	url := "https://query2.finance.yahoo.com/v8/finance/chart/BRL=X?interval=1d&range=10y"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	var resp *http.Response
	if yp, ok := s.provider.(*YahooFinanceProvider); ok {
		resp, err = yp.client.Do(req)
	} else {
		err = fmt.Errorf("not a YahooFinanceProvider")
	}
	// We need to use http.DefaultClient since we can't easily access the unexported client.
	// Actually, just create a temporary client here for simplicity, since it's just an internal helper.
	if err != nil {
		// fallback to generic http
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("yahoo finance fx error: %d", resp.StatusCode)
	}

	// Parse Yahoo Finance Chart response manually or use a map
	var data struct {
		Chart struct {
			Result []struct {
				Timestamp []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	rates := make(map[string]float64)
	if len(data.Chart.Result) > 0 {
		res := data.Chart.Result[0]
		if len(res.Indicators.Quote) > 0 {
			closes := res.Indicators.Quote[0].Close
			for i, ts := range res.Timestamp {
				if i < len(closes) && closes[i] > 0 {
					dateStr := time.Unix(ts, 0).Format("2006-01-02")
					rates[dateStr] = closes[i]
				}
			}
		}
	}

	if len(rates) > 0 {
		if cacheData, err := json.Marshal(rates); err == nil {
			s.rdb.Set(ctx, cacheKey, cacheData, s.ttlExchangeRates)
		}
	}

	return rates, nil
}

// GetHistoricalExchangeRate returns the USD to BRL exchange rate for a specific past date.
func (s *Service) GetHistoricalExchangeRate(ctx context.Context, date time.Time) (float64, error) {
	rates, err := s.getExchangeRatesMap(ctx)
	if err != nil {
		return 1.0, err // fallback to generic 1.0
	}

	// Try exactly the date
	dateStr := date.Format("2006-01-02")
	if rate, exists := rates[dateStr]; exists {
		return rate, nil
	}

	// If exact date (e.g. weekend/holiday) is missing, search backwards up to 7 days
	for i := 1; i <= 7; i++ {
		prevDateStr := date.AddDate(0, 0, -i).Format("2006-01-02")
		if rate, exists := rates[prevDateStr]; exists {
			return rate, nil
		}
	}

	return 1.0, fmt.Errorf("rate not found for date %s", dateStr)
}

// SearchAssets repassa a busca diretamente para o autocomplete do provedor.
func (s *Service) SearchAssets(ctx context.Context, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}

	// Busca dinâmica no provedor externo
	return s.provider.SearchAssets(ctx, query)
}

// GetFundamentals busca os fundamentos de uma ação com cache vitalício (24h)
func (s *Service) GetFundamentals(ctx context.Context, symbol string) (*Fundamentals, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("símbolo do ativo inválido")
	}

	key := fmt.Sprintf("fundamentals:v2:%s", symbol)

	// Tenta buscar no Redis (Cache Hit)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil && val != "" {
		var cachedFund Fundamentals
		if err := json.Unmarshal([]byte(val), &cachedFund); err == nil {
			return &cachedFund, nil
		}
	}

	// Se deu erro ou cache miss, faz scraping
	log.Printf("[Redis] CACHE MISS fundamentos %s. Rodando Scraper...", symbol)
	fund, err := s.scraper.GetFundamentals(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Calcula Preço Teto de Bazin se soubermos a cotação. 
	// Para não criar deadlock ou chamadas lentas demais, faremos depois. 
	// Wait, we can get current price from s.GetQuote(ctx, symbol) to calc Bazin Yield Ceiling
	quote, errQ := s.GetQuote(ctx, symbol)
	if errQ == nil && quote != nil && quote.Price > 0 {
		// Bazin Value = (Current Price * (Dividend Yield / 100)) / 0.06
		// Example: Price = 100. Yield = 10%. Dividend paid = 10. Bazin Value = 10 / 0.06 = 166.66
		annualDividend := quote.Price * (fund.DividendYield / 100.0)
		if annualDividend > 0 {
			fund.BazinValue = annualDividend / 0.06
		}
	}

	fundJSON, err := json.Marshal(fund)
	if err == nil {
		err = s.rdb.Set(ctx, key, fundJSON, s.ttlFundamentals).Err()
		if err != nil {
			log.Printf("[Redis] Erro ao salvar cache de fundamentos para %s: %v", symbol, err)
		}
	}

	return fund, nil
}
