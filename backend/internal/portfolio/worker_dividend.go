package portfolio

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/calculator"
)

type DividendWorker struct {
	repo          PortfolioRepository
	marketService MarketService
}

func NewDividendWorker(repo PortfolioRepository, ms MarketService) *DividendWorker {
	return &DividendWorker{
		repo:          repo,
		marketService: ms,
	}
}

func (w *DividendWorker) SyncAllDividends(ctx context.Context) {
	assets, err := w.repo.GetAllAssets(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "erro ao buscar ativos para sincronização de dividendos", slog.Any("error", err))
		return
	}

	slog.InfoContext(ctx, "iniciando sincronização de dividendos", slog.Int("assets_count", len(assets)))

	for _, asset := range assets {
		if asset.Ticker == "" {
			continue
		}

		assetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		events, err := w.marketService.GetDividends(assetCtx, asset.Ticker, asset.AssetType)
		if err != nil {
			slog.WarnContext(assetCtx, "erro ao buscar dividendos para ativo", slog.String("ticker", asset.Ticker), slog.Any("error", err))
			cancel()
			continue
		}

		var successCount int
		for i, ev := range events {
			existingEvents, err := w.repo.GetAssetEventsByDate(assetCtx, asset.ID, ev.Date)
			if err != nil {
				slog.ErrorContext(assetCtx, "erro ao buscar dividendos existentes", slog.String("ticker", asset.Ticker), slog.Time("date", ev.Date), slog.Any("error", err))
				continue
			}

			var bestMatch *AssetEvent
			var minDiff float64 = -1

			for j := range existingEvents {
				existing := &existingEvents[j]
				if existing.Type != ev.Type {
					continue
				}

				diff := math.Abs(existing.GrossAmount - ev.Amount)
				if diff <= calculator.FuzzyMatchGrossAmountThreshold {
					if minDiff == -1 || diff < minDiff {
						minDiff = diff
						bestMatch = existing
					}
				}
			}

			if bestMatch != nil {
				if minDiff < calculator.FinancialEpsilon && bestMatch.PaymentDate.Equal(ev.PaymentDate) {
					successCount++
					continue
				}

				// Update existing
				err = w.repo.UpdateAssetEventValueByID(assetCtx, bestMatch.ID, ev.Amount, ev.Amount, ev.PaymentDate)
				if err != nil {
					slog.ErrorContext(assetCtx, "erro ao atualizar dividendo (Fuzzy Match)",
						slog.Int("current", i+1),
						slog.Int("total", len(events)),
						slog.String("id", bestMatch.ID),
						slog.String("ticker", asset.Ticker),
						slog.Any("error", err),
					)
				} else {
					successCount++
				}
			} else {
				// Insert new
				err = w.repo.UpsertAssetEvent(assetCtx, AssetEvent{
					AssetID:     asset.ID,
					Type:        ev.Type,
					GrossAmount: ev.Amount,
					NetAmount:   ev.Amount, // We store gross in both places, taxes are applied per-portfolio later
					CumDate:     ev.Date,
					PaymentDate: ev.PaymentDate,
				})
				if err != nil {
					slog.ErrorContext(assetCtx, "erro ao salvar novo dividendo",
						slog.Int("current", i+1),
						slog.Int("total", len(events)),
						slog.String("cum_date", ev.Date.Format("2006-01-02")),
						slog.String("type", ev.Type),
						slog.Float64("amount", ev.Amount),
						slog.String("ticker", asset.Ticker),
						slog.Any("error", err),
					)
				} else {
					successCount++
				}
			}
		}

		if successCount > 0 {
			slog.InfoContext(assetCtx, "proventos sincronizados para ativo", slog.Int("count", successCount), slog.String("ticker", asset.Ticker))
		}

		cancel()

		// Small sleep to avoid hammering the scrapers
		time.Sleep(2 * time.Second)
	}

	slog.InfoContext(ctx, "sincronização de dividendos finalizada")
}
