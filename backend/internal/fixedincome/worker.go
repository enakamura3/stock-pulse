package fixedincome

import (
	"context"
	"log"
	"time"
)

// Worker gerencia a sincronização em segundo plano dos indicadores e índices
type Worker struct {
	repo     Repository
	registry *IndexRegistry
}

// NewWorker inicializa o worker com o repositório e o registro de provedores
func NewWorker(repo Repository, registry *IndexRegistry) *Worker {
	return &Worker{
		repo:     repo,
		registry: registry,
	}
}

// SyncRates sincroniza as séries históricas de todos os indexadores configurados
func (w *Worker) SyncRates(ctx context.Context) {
	indexers := []string{"CDI", "SELIC", "IPCA", "IFIX", "IBOV", "SP500"}
	endDate := time.Now()

	for _, indexer := range indexers {
		var startDate time.Time

		latest, err := w.repo.GetLatestIndexRate(ctx, indexer)
		if err == nil && latest != nil {
			startDate = latest.Date.AddDate(0, 0, 1)
		} else {
			// Se não tem dados históricos, inicia em 01/01/2010 (conforme acordado)
			startDate = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		if startDate.After(endDate) {
			continue // já está atualizado
		}

		// Divide a sincronização em blocos de no máximo 5 anos para evitar timeouts
		currentStart := startDate
		for currentStart.Before(endDate) {
			currentEnd := currentStart.AddDate(5, 0, 0)
			if currentEnd.After(endDate) {
				currentEnd = endDate
			}

			rates, err := w.registry.Fetch(ctx, indexer, currentStart, currentEnd)
			if err != nil {
				log.Printf("fixedincome worker: erro ao buscar dados de %s (%s a %s): %v", indexer, currentStart.Format("2006-01-02"), currentEnd.Format("2006-01-02"), err)
			} else if len(rates) > 0 {
				// Filtra finais de semana para economizar espaço e evitar inconsistências (exceto IPCA que é mensal)
				var filteredRates []IndexRate
				for _, r := range rates {
					wd := r.Date.Weekday()
					if indexer != "IPCA" && (wd == time.Saturday || wd == time.Sunday) {
						continue
					}
					filteredRates = append(filteredRates, r)
				}

				if len(filteredRates) > 0 {
					err = w.repo.SaveIndexRates(ctx, filteredRates)
					if err != nil {
						log.Printf("fixedincome worker: erro ao salvar taxas no banco para %s: %v", indexer, err)
					} else {
						log.Printf("fixedincome worker: sucesso ao sincronizar %d registros para %s (%s a %s)", len(filteredRates), indexer, currentStart.Format("2006-01-02"), currentEnd.Format("2006-01-02"))
					}
				}
			}

			// Avança para o próximo bloco (dia seguinte a currentEnd)
			currentStart = currentEnd.AddDate(0, 0, 1)

			// Pequeno delay entre requisições para evitar rate limit/bloqueio
			time.Sleep(500 * time.Millisecond)
		}
	}
}
