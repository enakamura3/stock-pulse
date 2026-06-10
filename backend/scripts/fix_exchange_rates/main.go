package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/database"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
)

func main() {
	ctx := context.Background()

	// 1. Inicializar Banco de Dados
	dbPool, err := database.NewPool()
	if err != nil {
		log.Fatalf("Falha ao conectar no banco de dados: %v", err)
	}
	defer dbPool.Close()

	// 2. Inicializar PortfolioService
	repo := portfolio.NewRepository(dbPool)
	provider := market.NewYahooFinanceProvider()
	marketService := market.NewService(provider, nil) // Não precisa mais do Redis
	portfolioService := portfolio.NewService(repo, marketService, provider, nil)

	log.Println("=== Iniciando varredura de Correção Cambial Retroativa ===")

	// 3. Buscar transações suspeitas
	query := `
		SELECT t.id, a.ticker, t.executed_at, p.base_currency, a.currency
		FROM transaction t
		JOIN portfolio p ON t.portfolio_id = p.id
		JOIN asset a ON t.asset_id = a.id
		WHERE t.exchange_rate = 1.0 AND p.base_currency != a.currency;
	`

	rows, err := dbPool.Query(ctx, query)
	if err != nil {
		log.Fatalf("Erro ao consultar transações: %v", err)
	}
	defer rows.Close()

	type txRow struct {
		ID           string
		Ticker       string
		ExecutedAt   time.Time
		BaseCurrency string
		AssetCurr    string
	}

	var toFix []txRow
	for rows.Next() {
		var r txRow
		if err := rows.Scan(&r.ID, &r.Ticker, &r.ExecutedAt, &r.BaseCurrency, &r.AssetCurr); err != nil {
			log.Printf("Erro ao fazer scan da linha: %v", err)
			continue
		}
		toFix = append(toFix, r)
	}

	if len(toFix) == 0 {
		log.Println("Nenhuma transação com câmbio incorreto foi encontrada. Banco já está saneado!")
		return
	}

	log.Printf("Encontradas %d transações para corrigir.", len(toFix))

	// 4. Corrigir as transações
	for _, r := range toFix {
		log.Printf("Processando TX %s | Ticker: %s | Data: %s", r.ID, r.Ticker, r.ExecutedAt.Format("2006-01-02"))

		currencyPair := fmt.Sprintf("%s%s=X", r.AssetCurr, r.BaseCurrency)

		// Tenta puxar do banco
		rate, err := repo.GetExchangeRateByDate(ctx, currencyPair, r.ExecutedAt)
		if err != nil || rate <= 0 {
			log.Printf("  [Info] Taxa não encontrada na base. Acionando BackfillGap...")
			portfolioService.BackfillGap(ctx, currencyPair, r.ExecutedAt)

			// Tenta novamente após tapar o buraco
			rate, err = repo.GetExchangeRateByDate(ctx, currencyPair, r.ExecutedAt)
			if err != nil || rate <= 0 {
				log.Printf("  [ERRO] Falha ao buscar cotação histórica mesmo após Backfill: %v", err)
				continue
			}
		}

		updateQuery := `UPDATE transaction SET exchange_rate = $1, total_cost = quantity * unit_price WHERE id = $2`
		cmdTag, err := dbPool.Exec(ctx, updateQuery, rate, r.ID)
		if err != nil {
			log.Printf("  [ERRO] Falha ao atualizar banco de dados: %v", err)
		} else if cmdTag.RowsAffected() > 0 {
			log.Printf("  [SUCESSO] Atualizado para %.4f", rate)
		}
	}

	log.Println("=== Varredura concluída ===")
}
