package fixedincome

import (
	"context"
	"log/slog"
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
		if ctx.Err() != nil {
			return
		}

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
			select {
			case <-ctx.Done():
				return
			default:
			}

			currentEnd := currentStart.AddDate(5, 0, 0)
			if currentEnd.After(endDate) {
				currentEnd = endDate
			}

			rates, err := w.registry.Fetch(ctx, indexer, currentStart, currentEnd)
			if err != nil {
				slog.ErrorContext(ctx, "fixedincome worker: erro ao buscar dados de indexador",
					slog.String("indexer", indexer),
					slog.String("start", currentStart.Format("2006-01-02")),
					slog.String("end", currentEnd.Format("2006-01-02")),
					slog.Any("error", err),
				)
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
						slog.ErrorContext(ctx, "fixedincome worker: erro ao salvar taxas no banco", slog.String("indexer", indexer), slog.Any("error", err))
					} else {
						slog.InfoContext(ctx, "fixedincome worker: sucesso ao sincronizar registros",
							slog.Int("count", len(filteredRates)),
							slog.String("indexer", indexer),
							slog.String("start", currentStart.Format("2006-01-02")),
							slog.String("end", currentEnd.Format("2006-01-02")),
						)
					}
				}
			}

			// Avança para o próximo bloco (dia seguinte a currentEnd)
			currentStart = currentEnd.AddDate(0, 0, 1)

			// Pequeno delay entre requisições para evitar rate limit/bloqueio (respeitando cancelamento de contexto)
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
}
