package fixedincome

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	repo      Repository
	bcbClient BCBClient
}

func NewWorker(repo Repository, bcbClient BCBClient) *Worker {
	return &Worker{
		repo:      repo,
		bcbClient: bcbClient,
	}
}

func (w *Worker) SyncRates(ctx context.Context) {
	indexers := []string{"CDI", "SELIC"}
	endDate := time.Now()

	for _, indexer := range indexers {
		var startDate time.Time

		latest, err := w.repo.GetLatestIndexRate(ctx, indexer)
		if err == nil && latest != nil {
			startDate = latest.Date.AddDate(0, 0, 1)
		} else {
			// Se não tem nada (primeira sincronização ou wipe), busca histórico desde 2010 para garantir o cálculo de ativos antigos.
			startDate = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		if startDate.After(endDate) {
			continue // já está atualizado
		}

		// The BCB API limits daily series queries to 10 years.
		// We will chunk the requests in 5-year intervals.
		currentStart := startDate
		for currentStart.Before(endDate) {
			currentEnd := currentStart.AddDate(5, 0, 0)
			if currentEnd.After(endDate) {
				currentEnd = endDate
			}

			rates, err := w.bcbClient.FetchRates(ctx, indexer, currentStart, currentEnd)
			if err != nil {
				log.Printf("fixedincome worker: error fetching rates for %s (%v to %v): %v", indexer, currentStart, currentEnd, err)
			} else if len(rates) > 0 {
				err = w.repo.SaveIndexRates(ctx, rates)
				if err != nil {
					log.Printf("fixedincome worker: error saving rates for %s: %v", indexer, err)
				} else {
					log.Printf("fixedincome worker: successfully saved %d rates for %s (%v to %v)", len(rates), indexer, currentStart.Format("2006-01-02"), currentEnd.Format("2006-01-02"))
				}
			}

			// Move to the next chunk (next day after currentEnd)
			currentStart = currentEnd.AddDate(0, 0, 1)
			
			// Sleep briefly to avoid hammering the BCB API
			time.Sleep(500 * time.Millisecond)
		}
	}
}
