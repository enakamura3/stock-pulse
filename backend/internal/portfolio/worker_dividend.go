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

func (w *DividendWorker) SyncAllDividends(ctx context.Context) {
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
		
		log.Printf("[DividendWorker] Buscando proventos para o ativo %s (Tipo: %s)...", asset.Ticker, asset.AssetType)
		
		events, err := w.marketService.GetDividends(assetCtx, asset.Ticker, asset.AssetType)
		if err != nil {
			log.Printf("[DividendWorker] Aviso: falha ao buscar proventos para %s: %v", asset.Ticker, err)
			cancel()
			continue
		}
		
		log.Printf("[DividendWorker] Ativo %s: Encontrados %d proventos na origem.", asset.Ticker, len(events))

		successCount := 0
		for i, ev := range events {
			err = w.repo.UpsertAssetEvent(assetCtx, AssetEvent{
				AssetID:     asset.ID,
				Type:        ev.Type,
				GrossAmount: ev.Amount,
				NetAmount:   ev.Amount, // We store gross in both places, taxes are applied per-portfolio later
				ExDate:      ev.Date,
				PaymentDate: ev.PaymentDate,
			})
			if err != nil {
				log.Printf("[DividendWorker] Erro ao salvar dividendo %d/%d (DataCom: %s, Tipo: %s, Valor: %.4f) para %s: %v", 
					i+1, len(events), ev.Date.Format("2006-01-02"), ev.Type, ev.Amount, asset.Ticker, err)
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
