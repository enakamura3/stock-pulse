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
			{source: b3, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: stockAnalysis, role: "fallback"},
		},
		"FIAGRO": {
			{source: b3, role: "primary"},
			{source: fundamentus, role: "secondary"},
			{source: stockAnalysis, role: "fallback"},
		},
		"ETF_BR": {
			{source: b3, role: "primary"},
			{source: stockAnalysis, role: "secondary"},
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
				log.Printf("[Market] Falha nas rotas primárias/secundárias para %s. Usando fallback %s.", ticker, fallback.Name())
				events, fetchErr = fallback.GetDividends(ctx, ticker, assetType)
			}
		}
	} else if fallback != nil {
		events, fetchErr = fallback.GetDividends(ctx, ticker, assetType)
	}

	if fetchErr != nil && len(events) == 0 {
		return nil, fmt.Errorf("nenhum provento encontrado para %s: %v", ticker, fetchErr)
	}

	// 5. Enriquecimento com fallback para o mês corrente.
	//
	// Se primary/secondary retornaram dados mas o mês atual está ausente,
	// consultamos o fallback para preencher a lacuna. O merge usa a mesma
	// lógica de deduplicação: meses já cobertos pelo primary/secondary têm
	// prioridade — o fallback só contribui com meses que ainda não existem.
	if fallback != nil && len(events) > 0 && isMissingCurrentMonth(events) {
		log.Printf("[Market] %s: mês corrente ausente no resultado primary/secondary. Consultando fallback %s para enriquecer.", ticker, fallback.Name())
		fallbackEvents, errF := fallback.GetDividends(ctx, ticker, assetType)
		if errF == nil && len(fallbackEvents) > 0 {
			events = enrichWithFallback(events, fallbackEvents, assetType)
			log.Printf("[Market] %s: fallback %s contribuiu com %d evento(s) do mês corrente.", ticker, fallback.Name(), countCurrentMonthEvents(events))
		} else if errF != nil {
			log.Printf("[Market] %s: fallback %s falhou ao enriquecer: %v", ticker, fallback.Name(), errF)
		}
	}

	// 6. Salvar cache
	if data, err := json.Marshal(events); err == nil {
		g.cache.Set(ctx, cacheKey, data, g.ttl)
	}

	return events, nil
}

// isMissingCurrentMonth retorna true se nenhum evento da lista pertence ao mês e ano atuais.
func isMissingCurrentMonth(events []DividendEvent) bool {
	now := time.Now()
	for _, ev := range events {
		if ev.Date.Month() == now.Month() && ev.Date.Year() == now.Year() {
			return false
		}
	}
	return true
}

// countCurrentMonthEvents conta quantos eventos pertencem ao mês e ano atuais.
func countCurrentMonthEvents(events []DividendEvent) int {
	now := time.Now()
	count := 0
	for _, ev := range events {
		if ev.Date.Month() == now.Month() && ev.Date.Year() == now.Year() {
			count++
		}
	}
	return count
}

// enrichWithFallback adiciona ao resultado base os eventos do fallback para os quais
// o base ainda não possui cobertura. Eventos já cobertos pelo base (primary/secondary)
// são ignorados — preservando a prioridade das fontes principais.
//
// Para FIIs/FIAGROs, a cobertura é verificada por mês+ano (1 rendimento/mês).
// Para outros tipos, a cobertura é verificada por data exata.
func enrichWithFallback(baseEvents, fallbackEvents []DividendEvent, assetType string) []DividendEvent {
	isFii := strings.ToUpper(assetType) == "FII" || strings.ToUpper(assetType) == "FIAGRO"

	enriched := append([]DividendEvent{}, baseEvents...)

	for _, fEv := range fallbackEvents {
		covered := false
		for _, bEv := range baseEvents {
			if isFii {
				if fEv.Date.Month() == bEv.Date.Month() && fEv.Date.Year() == bEv.Date.Year() {
					covered = true
					break
				}
			} else {
				if fEv.Date.Equal(bEv.Date) {
					covered = true
					break
				}
			}
		}
		if !covered {
			// Normaliza o tipo para FIIs caso o fallback retorne "Dividendo"
			if isFii && fEv.Type == "Dividendo" {
				fEv.Type = "Rendimento"
			}
			enriched = append(enriched, fEv)
		}
	}

	sort.SliceStable(enriched, func(i, j int) bool {
		return enriched[i].Date.After(enriched[j].Date)
	})

	return enriched
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
