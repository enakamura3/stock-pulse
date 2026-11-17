package dividend

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Gateway gerencia o roteamento entre as fontes de proventos (Primária -> Fallback 1 -> Fallback 2),
// aplicando o pipeline único (Normalize -> Deduplicate -> Validate) e armazenando em cache no Redis.
type Gateway struct {
	routes map[string][]Source
	cache  *redis.Client
	ttl    time.Duration
}

// NewGateway cria uma nova instância de Gateway com Investidor10 como fonte primária universal,
// Fundamentus e B3 como fallbacks para o Brasil, e StockAnalysis como fallback para os EUA.
func NewGateway(
	investidor10 Source,
	fundamentus Source,
	b3 Source,
	stockAnalysis Source,
	rdb *redis.Client,
	ttl time.Duration,
) *Gateway {
	routes := map[string][]Source{
		"STOCK_BR": {investidor10, fundamentus, b3},
		"FII":      {investidor10, fundamentus, b3},
		"FIAGRO":   {investidor10, fundamentus, b3},
		"ETF_BR":   {investidor10, fundamentus, b3},
		"BDR":      {investidor10, fundamentus, b3},
		"STOCK_US": {investidor10, stockAnalysis},
		"ETF_US":   {investidor10, stockAnalysis},
	}

	return &Gateway{
		routes: routes,
		cache:  rdb,
		ttl:    ttl,
	}
}

// NewDefaultGateway instancia todos os clientes e fontes padrão de proventos.
func NewDefaultGateway(rdb *redis.Client, ttl time.Duration) *Gateway {
	i10 := NewInvestidor10DividendSource(NewInvestidor10Client())
	fund := NewFundamentusDividendSource(NewFundamentusClient())
	b3 := NewB3DividendSource(NewB3Client())
	sa := NewStockAnalysisDividendSource(NewStockAnalysisClient())
	return NewGateway(i10, fund, b3, sa, rdb, ttl)
}

// GetDividends busca os proventos de um ativo percorrendo a cadeia de fontes da rota até obter eventos válidos
// pelo pipeline único (Normalize -> Deduplicate -> Validate).
func (g *Gateway) GetDividends(ctx context.Context, ticker string, assetType string) ([]Event, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	cacheKey := fmt.Sprintf("dividends:%s", ticker)

	if g.cache != nil {
		val, err := g.cache.Get(ctx, cacheKey).Result()
		if err == nil {
			var cached []Event
			if err := json.Unmarshal([]byte(val), &cached); err == nil {
				slog.DebugContext(ctx, "cache hit proventos", slog.String("ticker", ticker))
				return cached, nil
			}
		}
	}

	slog.InfoContext(ctx, "cache miss proventos, consultando rotas", slog.String("ticker", ticker))

	assetTypeUpper := strings.ToUpper(strings.TrimSpace(assetType))
	sources, ok := g.routes[assetTypeUpper]
	if !ok {
		return nil, fmt.Errorf("tipo de ativo desconhecido para buscar proventos: %s", assetTypeUpper)
	}

	var validEvents []Event
	var lastErr error

	for idx, src := range sources {
		if src == nil {
			continue
		}
		rawEvents, err := src.GetDividends(ctx, ticker, assetTypeUpper)
		if err != nil {
			lastErr = err
			slog.WarnContext(ctx, "fonte de proventos falhou, tentando próxima na rota",
				slog.String("ticker", ticker),
				slog.String("source", src.Name()),
				slog.Int("attempt", idx+1),
				slog.Any("error", err),
			)
			continue
		}

		// Executa o pipeline único: 1. Normalize -> 2. Deduplicate -> 3. Validate
		processed := ProcessPipeline(rawEvents, assetTypeUpper)
		if len(processed) > 0 {
			validEvents = processed
			break
		}
	}

	if len(validEvents) == 0 && lastErr != nil {
		return nil, fmt.Errorf("nenhum provento encontrado para %s: %v", ticker, lastErr)
	}

	if g.cache != nil && len(validEvents) > 0 {
		if data, err := json.Marshal(validEvents); err == nil {
			g.cache.Set(ctx, cacheKey, data, g.ttl)
		}
	}

	return validEvents, nil
}
