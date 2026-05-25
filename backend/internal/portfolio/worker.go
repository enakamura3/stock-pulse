package portfolio

import (
	"context"
	"log/slog"
	"time"

	"github.com/onigiri/stockpulse/backend/internal/market"
)

// DailyWorker gerencia Goroutines em segundo plano para capturar snapshots de fechamento de mercado.
type DailyWorker struct {
	repo           *Repository
	marketProvider market.QuoteProvider
}

// NewDailyWorker cria uma nova instância do DailyWorker.
func NewDailyWorker(repo *Repository, marketProvider market.QuoteProvider) *DailyWorker {
	return &DailyWorker{
		repo:           repo,
		marketProvider: marketProvider,
	}
}

// Start inicializa o loop periódico do worker (com trigger imediato no startup).
func (w *DailyWorker) Start(ctx context.Context) {
	slog.Info("Daily Worker inicializado com sucesso em background.")
	
	// Executa uma varredura imediata no startup de dev para garantir dados frescos locais
	go w.run(ctx)
	
	// Executa a cada 24 horas
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Parando rotina diária em background...")
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *DailyWorker) run(ctx context.Context) {
	slog.Info("Executando varredura agendada de cotações de fechamento...")
	
	// Recupera todos os ativos cadastrados
	assets, err := w.repo.GetAllAssets(ctx)
	if err != nil {
		slog.Error("Erro ao recuperar ativos no banco para atualização diária", "error", err)
		return
	}
	
	if len(assets) == 0 {
		slog.Warn("Nenhum ativo cadastrado na base para atualização diária.")
		return
	}
	
	slog.Info("Sincronizando preços diários de ativos em lote", "count", len(assets))
	
	now := time.Now().UTC()
	// Normaliza a data (DATE) zerando as frações de hora para conformidade PostgreSQL
	priceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	
	for _, asset := range assets {
		// Pausa leve para evitar ser rate-limited pelas APIs do Yahoo Finance
		select {
		case <-ctx.Done():
			return
		case <-time.After(350 * time.Millisecond):
		}
		
		quote, err := w.marketProvider.GetQuote(ctx, asset.Ticker)
		if err != nil {
			slog.Error("Erro ao obter cotação diária do provedor de mercado", "ticker", asset.Ticker, "error", err)
			continue
		}
		
		prices := []DailyPrice{
			{
				AssetID:    asset.ID,
				PriceDate:  priceDate,
				ClosePrice: quote.Price,
			},
		}
		
		err = w.repo.SaveDailyPrices(ctx, asset.ID, prices)
		if err != nil {
			slog.Error("Erro ao salvar preço diário consolidado", "ticker", asset.Ticker, "error", err)
		} else {
			slog.Info("SUCESSO: Ativo diário atualizado", "ticker", asset.Ticker, "price", quote.Price, "currency", asset.Currency)
		}
	}
	
	slog.Info("Varredura diária concluída.")
}
