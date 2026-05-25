package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Service gerencia cotações de ativos agregando cacheamento Redis de alta performance.
type Service struct {
	provider QuoteProvider
	scraper  *Scraper
	rdb      *redis.Client
	ttl      time.Duration
}

// NewService cria uma nova instância de Service com o TTL configurado para 60 segundos.
func NewService(provider QuoteProvider, rdb *redis.Client) *Service {
	return &Service{
		provider: provider,
		scraper:  NewScraper(),
		rdb:      rdb,
		ttl:      60 * time.Second, // Decisão aprovada pelo usuário
	}
}

// GetQuote resgata a cotação do cache do Redis ou faz bypass consultando o provedor Yahoo.
func (s *Service) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("símbolo do ativo inválido")
	}

	key := fmt.Sprintf("quote:%s", symbol)

	// Tenta buscar no Redis (Cache Hit)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil && val != "" {
		var cachedQuote Quote
		if err := json.Unmarshal([]byte(val), &cachedQuote); err == nil {
			log.Printf("[Redis] CACHE HIT para o ativo %s", symbol)
			return &cachedQuote, nil
		}
	}

	// Se deu erro ou cache miss, busca do provedor externo
	log.Printf("[Redis] CACHE MISS para o ativo %s. Consultando provedor...", symbol)
	quote, err := s.provider.GetQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// Serializa e salva no Redis de forma assíncrona (ou imediata) com TTL de 60s
	quoteJSON, err := json.Marshal(quote)
	if err == nil {
		err = s.rdb.Set(ctx, key, quoteJSON, s.ttl).Err()
		if err != nil {
			log.Printf("[Redis] Erro ao salvar cache para %s: %v", symbol, err)
		} else {
			log.Printf("[Redis] Novo cache salvo para %s com sucesso (TTL 60s)", symbol)
		}
	}

	return quote, nil
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

	key := fmt.Sprintf("fundamentals:%s", symbol)

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
		// Salva no Redis com TTL longo de 24 horas, já que muda a cada trimestre
		err = s.rdb.Set(ctx, key, fundJSON, 24*time.Hour).Err()
		if err != nil {
			log.Printf("[Redis] Erro ao salvar cache de fundamentos para %s: %v", symbol, err)
		}
	}

	return fund, nil
}
