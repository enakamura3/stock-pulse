package portfolio

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/calculator"
)

// GetPortfolioDividends calcula todos os dividendos (históricos e futuros) com base na posição da carteira na data com-dividendo (cum-dividend date).
func (s *Service) GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]CalculatedDividend, error) {
	// Verifica a existência do portfólio para garantir autorização
	_, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return nil, err
	}

	// Busca todas as transações para calcular as posições no tempo
	transactions, err := s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
	if err != nil {
		return nil, err
	}

	// Agrupa transações por ticker para facilitar processamento cronológico
	txByTicker := make(map[string][]Transaction)
	for _, tx := range transactions {
		txByTicker[tx.Ticker] = append(txByTicker[tx.Ticker], tx)
	}

	var results []CalculatedDividend

	// Para cada ativo, buscamos os proventos e iteramos para calcular
	for ticker, txs := range txByTicker {
		// Ordena transações cronologicamente
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].ExecutedAt.Before(txs[j].ExecutedAt)
		})

		// A moeda base do ativo (BRL ou USD)
		currency := "BRL"
		if len(txs) > 0 && txs[0].Currency != "" {
			currency = txs[0].Currency
		}

		divs, err := s.repo.GetAssetEvents(ctx, txs[0].AssetID)
		if err != nil {
			log.Printf("Aviso: falha ao buscar dividendos locais para %s: %v", ticker, err)
			continue
		}

		for _, div := range divs {
			cumDate := div.CumDate

			// Calcula a quantidade na carteira na Data Com / Data Base (cum_date),
			// ou seja, transações efetuadas até a cum_date inclusive.
			var quantity float64 = 0
			cumDateNorm := time.Date(cumDate.Year(), cumDate.Month(), cumDate.Day(), 0, 0, 0, 0, time.UTC)
			for _, tx := range txs {
				txDate := time.Date(tx.ExecutedAt.Year(), tx.ExecutedAt.Month(), tx.ExecutedAt.Day(), 0, 0, 0, 0, time.UTC)
				// Se a transação ocorreu em um dia após a Data Com (cum_date), interrompe a soma
				if txDate.After(cumDateNorm) {
					break
				}

				if tx.Type == "BUY" {
					quantity += tx.Quantity
				} else if tx.Type == "SELL" {
					quantity -= tx.Quantity
				} else if tx.Type == "SPLIT" {
					quantity = quantity * tx.Quantity
				} else if tx.Type == "REVERSE_SPLIT" {
					if tx.Quantity > 0 {
						quantity = math.Floor(quantity / tx.Quantity)
					}
				} else if tx.Type == "BONUS" {
					quantity += tx.Quantity
				}
			}

			// Se a quantidade resultante é > 0, o usuário tem direito ao dividendo!
			if quantity > 0 {
				grossAmount := quantity * div.GrossAmount
				netAmount := grossAmount
				divCurrency := currency // cópia local para não afetar as próximas iterações

				// Regras de Impostos
				if divCurrency == "USD" {
					// EUA: 30% retido na fonte
					netAmount = grossAmount * calculator.USWithholdingNetFactor
				} else if divCurrency == "BRL" {
					if div.Type == "JCP" {
						// JCP: 15% de imposto retido na fonte
						netAmount = grossAmount * calculator.B3WithholdingNetFactor
					} else if strings.HasPrefix(txs[0].AssetType, "ETF") {
						// Exceção: ETFs na B3 que sofrem tributação de 15% nos dividendos retidos na fonte
						netAmount = grossAmount * calculator.B3WithholdingNetFactor
					} else {
						// Dividendos, Rendimentos (FII), Amortização: 0% de imposto
						netAmount = grossAmount
					}
				}

				exchangeRate := 1.0
				var originalGross float64 = 0
				var originalNet float64 = 0

				// Conversão Cambial (Apenas para USD -> BRL)
				if divCurrency == "USD" {
					originalGross = grossAmount
					originalNet = netAmount

					now := time.Now()
					// Regra de Mercado:
					// 1. Se a Data de Pagamento está definida e já ocorreu no passado: usa o câmbio histórico da PaymentDate
					if !div.PaymentDate.IsZero() && !div.PaymentDate.After(now) {
						fx, err := s.marketService.GetHistoricalExchangeRate(ctx, div.PaymentDate)
						if err == nil && fx > calculator.FinancialEpsilon {
							exchangeRate = fx
						} else {
							exchangeRate = s.getCurrencyRate(ctx, "USD", "BRL")
						}
					} else {
						// 2. Se o dividendo é futuro ou com PaymentDate não informada: usa o câmbio atual (Hoje / Spot) como projeção
						exchangeRate = s.getCurrencyRate(ctx, "USD", "BRL")
					}

					// Converte Gross e Net para a moeda base do Portfolio (assumindo BRL)
					grossAmount = grossAmount * exchangeRate
					netAmount = netAmount * exchangeRate
					divCurrency = "BRL"
				}

				results = append(results, CalculatedDividend{
					AssetID:        txs[0].AssetID,
					Ticker:         ticker,
					CumDate:        cumDate,
					PaymentDate:    div.PaymentDate,
					GrossAmount:    grossAmount,
					NetAmount:      netAmount,
					Currency:       divCurrency,
					OriginalGross:  originalGross,
					OriginalNet:    originalNet,
					Type:           div.Type,
					Quantity:       quantity,
					PerShareAmount: div.GrossAmount * exchangeRate,
					AssetType:      txs[0].AssetType,
					AssetName:      txs[0].AssetName,
				})
			}
		}
	}

	// Deduplica proventos mensais de FII/FIAGRO/ETF no mesmo mês/ano
	dedupedResults := make([]CalculatedDividend, 0, len(results))
	seenMonth := make(map[string]bool)

	for _, res := range results {
		upperType := strings.ToUpper(res.AssetType)
		if upperType == "FII" || upperType == "FIAGRO" || upperType == "ETF" || upperType == "ETF_BR" {
			dateForMonth := res.PaymentDate
			if dateForMonth.IsZero() {
				dateForMonth = res.CumDate
			}
			key := fmt.Sprintf("%s-%d-%02d", res.Ticker, dateForMonth.Year(), dateForMonth.Month())
			if seenMonth[key] {
				continue
			}
			seenMonth[key] = true
		}
		dedupedResults = append(dedupedResults, res)
	}

	// Ordena do mais recente para o mais antigo
	sort.Slice(dedupedResults, func(i, j int) bool {
		return dedupedResults[i].PaymentDate.After(dedupedResults[j].PaymentDate)
	})

	return dedupedResults, nil
}
