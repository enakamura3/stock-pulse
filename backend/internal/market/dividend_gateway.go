package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	fundamentus DividendSource,
	stockAnalysis DividendSource,
	yahoo DividendSource,
	rdb *redis.Client,
	ttl time.Duration,
) *DividendGateway {
	routes := map[string][]sourceEntry{
		"STOCK_BR": {
			{source: fundamentus, role: "primary"},
			{source: stockAnalysis, role: "secondary"},
			{source: yahoo, role: "fallback"},
		},
		"FII": {
			{source: stockAnalysis, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: yahoo, role: "fallback"},
		},
		"FIAGRO": {
			{source: stockAnalysis, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: yahoo, role: "fallback"},
		},
		"ETF_BR": {
			{source: stockAnalysis, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: yahoo, role: "fallback"},
		},
		"BDR": {
			{source: stockAnalysis, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: yahoo, role: "fallback"},
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
	cacheKey := fmt.Sprintf("dividends:%s", ticker)
	
	// 1. Verificar cache Redis
	val, err := g.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached []DividendEvent
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			log.Printf("[Redis] CACHE HIT proventos para %s", ticker)
			return cached, nil
		}
	}

	log.Printf("[Redis] CACHE MISS proventos para %s. Consultando rotas...", ticker)

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
				// Merge primary and secondary according to roles in mergeAndDedupDividends
				// In our routes:
				// STOCK_BR: primary=fundamentus, secondary=stockAnalysis -> mergeAndDedup(saEvents=secondary, fundEvents=primary)
				// FII/FIAGRO: primary=stockAnalysis, secondary=fundamentus -> mergeAndDedup(saEvents=primary, fundEvents=secondary)
				if assetTypeUpper == "FII" || assetTypeUpper == "FIAGRO" {
					events = mergeAndDedupDividends(primaryEvents, secondaryEvents, assetType)
				} else { // STOCK_BR, ETF_BR, BDR
					events = mergeAndDedupDividends(secondaryEvents, primaryEvents, assetType) // saEvents, fundEvents
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
				log.Printf("[Market] Falha nas rotas primárias/secundárias para %s. Usando fallback %s.", ticker, fallback.Name())
				events, fetchErr = fallback.GetDividends(ctx, ticker, assetType)
			}
		}
	} else if fallback != nil { // No primary, only fallback? (shouldn't happen with our routes)
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
