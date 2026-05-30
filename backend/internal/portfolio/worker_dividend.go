package portfolio

import (
	"context"
	"log"
	"time"
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

func (w *DividendWorker) Start(ctx context.Context) {
	log.Println("[DividendWorker] Inicializado. Rodando a cada 24 horas.")

	// Executa uma vez imediatamente ao ligar o servidor
	w.syncAllDividends(ctx)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[DividendWorker] Encerrando graciosamente...")
			return
		case <-ticker.C:
			w.syncAllDividends(ctx)
		}
	}
}

func (w *DividendWorker) syncAllDividends(ctx context.Context) {
	log.Println("[DividendWorker] Iniciando sincronização de dividendos de mercado...")

	assets, err := w.repo.GetAllAssets(ctx)
	if err != nil {
		log.Printf("[DividendWorker] Erro ao buscar ativos: %v", err)
		return
	}

	for _, asset := range assets {
		// Use the market service to fetch the dividends (which uses scrapers or Yahoo as fallback)
		// We use a new background context with timeout for each asset to prevent hanging
		assetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		
		events, err := w.marketService.GetDividends(assetCtx, asset.Ticker, asset.AssetType)
		if err != nil {
			log.Printf("[DividendWorker] Aviso: falha ao buscar proventos para %s: %v", asset.Ticker, err)
			cancel()
			continue
		}

		successCount := 0
		for _, ev := range events {
			err = w.repo.UpsertAssetEvent(assetCtx, AssetEvent{
				AssetID:     asset.ID,
				Type:        ev.Type,
				GrossAmount: ev.Amount,
				NetAmount:   ev.Amount, // We store gross in both places, taxes are applied per-portfolio later
				ExDate:      ev.Date,
				PaymentDate: ev.PaymentDate,
			})
			if err != nil {
				log.Printf("[DividendWorker] Erro ao salvar dividendo %s para %s: %v", ev.Date.Format("2006-01-02"), asset.Ticker, err)
			} else {
				successCount++
			}
		}

		if successCount > 0 {
			log.Printf("[DividendWorker] Sincronizados %d proventos para %s", successCount, asset.Ticker)
		}
		
		cancel()
		
		// Small sleep to avoid hammering the scrapers
		time.Sleep(2 * time.Second)
	}

	log.Println("[DividendWorker] Sincronização finalizada.")
}
