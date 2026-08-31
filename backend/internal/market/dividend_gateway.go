package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type sourceEntry struct {
	source DividendSource
	role   string // "primary", "secondary", "fallback"
}

type DividendGateway struct {
	routes map[string][]sourceEntry
	cache  *redis.Client
	ttl    time.Duration
}

func NewDividendGateway(
	b3 DividendSource,
	fundamentus DividendSource,
	stockAnalysis DividendSource,
	yahoo DividendSource,
	rdb *redis.Client,
	ttl time.Duration,
) *DividendGateway {
	routes := map[string][]sourceEntry{
		"STOCK_BR": {
			{source: b3, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: stockAnalysis, role: "fallback"},
		},
		"FII": {
			{source: fundamentus, role: "primary"},
			{source: stockAnalysis, role: "secondary"},
		},
		"FIAGRO": {
			{source: fundamentus, role: "primary"},
			{source: stockAnalysis, role: "secondary"},
		},
		"ETF_BR": {
			{source: b3, role: "primary"},
			{source: stockAnalysis, role: "secondary"},
		},
		"BDR": {
			{source: stockAnalysis, role: "primary"},
		},
		"STOCK_US": {
			{source: stockAnalysis, role: "primary"},
			{source: yahoo, role: "fallback"},
		},
		"ETF_US": {
			{source: stockAnalysis, role: "primary"},
			{source: yahoo, role: "fallback"},
		},
		"CRYPTO": {
			{source: yahoo, role: "primary"},
		},
	}

	return &DividendGateway{
		routes: routes,
		cache:  rdb,
		ttl:    ttl,
	}
}

func (g *DividendGateway) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	cacheKey := fmt.Sprintf("dividends:%s", ticker)

	// 1. Verificar cache Redis
	val, err := g.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached []DividendEvent
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			slog.DebugContext(ctx, "cache hit proventos", slog.String("ticker", ticker))
			return cached, nil
		}
	}

	slog.InfoContext(ctx, "cache miss proventos, consultando rotas", slog.String("ticker", ticker))

	// 2. Buscar rota
	assetTypeUpper := strings.ToUpper(assetType)
	route, ok := g.routes[assetTypeUpper]
	if !ok {
		return nil, fmt.Errorf("tipo de ativo desconhecido para buscar proventos: %s", assetTypeUpper)
	}

	// 3. Identificar sources
	var primary, secondary, fallback DividendSource
	for _, r := range route {
		if r.role == "primary" && primary == nil {
			primary = r.source
		} else if r.role == "secondary" && secondary == nil {
			secondary = r.source
		} else if r.role == "fallback" && fallback == nil {
			fallback = r.source
		}
	}

	var events []DividendEvent
	var fetchErr error

	// 4. Chamar sources
	if primary != nil {
		primaryEvents, errP := primary.GetDividends(ctx, ticker, assetType)
		if errP == nil && len(primaryEvents) > 0 {
			if secondary != nil {
				secondaryEvents, _ := secondary.GetDividends(ctx, ticker, assetType)
				if assetTypeUpper == "FII" || assetTypeUpper == "FIAGRO" {
					events = mergeAndDedupDividends(primaryEvents, secondaryEvents, assetType)
				} else { // STOCK_BR, ETF_BR, BDR
					events = mergeAndDedupDividends(secondaryEvents, primaryEvents, assetType)
				}
			} else {
				events = primaryEvents
			}
		} else {
			fetchErr = errP
			// Tenta secondary sozinho se primary falhou ou retornou vazio
			if secondary != nil {
				events, fetchErr = secondary.GetDividends(ctx, ticker, assetType)
			}
			// Se continuou falhando, tenta fallback
			if (fetchErr != nil || len(events) == 0) && fallback != nil {
				slog.WarnContext(ctx, "falha nas rotas primárias/secundárias, usando fallback", slog.String("ticker", ticker), slog.String("fallback", fallback.Name()))
				events, fetchErr = fallback.GetDividends(ctx, ticker, assetType)
			}
		}
	} else if fallback != nil {
		events, fetchErr = fallback.GetDividends(ctx, ticker, assetType)
	}

	if fetchErr != nil && len(events) == 0 {
		return nil, fmt.Errorf("nenhum provento encontrado para %s: %v", ticker, fetchErr)
	}

	// 5. Salvar cache
	if data, err := json.Marshal(events); err == nil {
		g.cache.Set(ctx, cacheKey, data, g.ttl)
	}

	return events, nil
}

func isFiiType(assetType string) bool {
	upper := strings.ToUpper(assetType)
	return upper == "FII" || upper == "FIAGRO"
}

func isMonthlyYieldAsset(assetType string) bool {
	upper := strings.ToUpper(assetType)
	return upper == "FII" || upper == "FIAGRO" || upper == "ETF" || upper == "ETF_BR"
}

func mergeAndDedupDividends(saEvents, fundEvents []DividendEvent, assetType string) []DividendEvent {
	isMonthly := isMonthlyYieldAsset(assetType)
	isFii := isFiiType(assetType)

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
			if isMonthly {
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
