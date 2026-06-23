package fixedincome

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AnbimaHolidayWorker sincroniza feriados nacionais (ANBIMA) via BrasilAPI.
// Na primeira execução retroage até o ano de início definido em startYear para garantir
// que cálculos de DU/252 funcionem para compras retroativas.
// Nas execuções seguintes, garante que o ano corrente e o próximo estejam populados.
type AnbimaHolidayWorker struct {
	repo        Repository
	client      AnbimaClient
	startYear   int // Ano mínimo do histórico (ex: 2010)
}

func NewAnbimaHolidayWorker(repo Repository, client AnbimaClient) *AnbimaHolidayWorker {
	return &AnbimaHolidayWorker{
		repo:      repo,
		client:    client,
		startYear: 2010,
	}
}

// SyncHolidays é o ponto de entrada do worker.
// Garante que os feriados de todos os anos entre startYear e currentYear+1 estão no banco.
func (w *AnbimaHolidayWorker) SyncHolidays(ctx context.Context) {
	currentYear := time.Now().Year()

	// Verifica quais anos já têm dados
	seeded, err := w.repo.GetSeededHolidayYears(ctx)
	if err != nil {
		log.Printf("AnbimaHolidayWorker: failed to check seeded years: %v", err)
		return
	}

	seededSet := make(map[int]bool, len(seeded))
	for _, y := range seeded {
		seededSet[y] = true
	}

	// Sincroniza todos os anos faltando de startYear até currentYear+1
	for year := w.startYear; year <= currentYear+1; year++ {
		if seededSet[year] {
			continue
		}

		if err := w.syncYear(ctx, year); err != nil {
			log.Printf("AnbimaHolidayWorker: failed to sync year %d: %v", year, err)
			// Continua para o próximo ano mesmo em caso de erro
			continue
		}

		log.Printf("AnbimaHolidayWorker: seeded holidays for %d", year)

		// Pausa entre requisições para não sobrecarregar a BrasilAPI
		select {
		case <-ctx.Done():
			return
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func (w *AnbimaHolidayWorker) syncYear(ctx context.Context, year int) error {
	holidays, err := w.client.FetchHolidays(ctx, year)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	if len(holidays) == 0 {
		return nil
	}

	// Filtra apenas feriados nacionais (type == "national")
	// A BrasilAPI também retorna municipais/estaduais em alguns endpoints
	var dates []time.Time
	for _, h := range holidays {
		if h.Type != "" && h.Type != "national" {
			continue
		}
		t, err := time.Parse("2006-01-02", h.Date)
		if err != nil {
			continue
		}
		dates = append(dates, t)
	}

	if len(dates) == 0 {
		return nil
	}

	return w.repo.SaveAnbimaHolidays(ctx, dates)
}
