package portfolio

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/calculator"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
)

// GetPortfolioPerformance gera a rentabilidade diária agregada e comparações contra benchmarks para a carteira.
func (s *Service) GetPortfolioPerformance(ctx context.Context, portfolioID string, userID string, period string, filterTickers []string) ([]PerformancePoint, error) {
	// Valida se a carteira pertence ao usuário
	p, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return nil, errors.New("carteira não encontrada ou acesso não autorizado")
	}

	// Carrega todas as transações
	txs, err := s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar transações: %w", err)
	}
	if len(txs) == 0 {
		return []PerformancePoint{}, nil
	}

	// Filtra as transações caso o usuário tenha selecionado uma categoria específica
	if len(filterTickers) > 0 {
		tickerMap := make(map[string]bool)
		for _, t := range filterTickers {
			tickerMap[t] = true
		}
		filteredTxs := make([]Transaction, 0)
		for _, tx := range txs {
			if tickerMap[tx.Ticker] {
				filteredTxs = append(filteredTxs, tx)
			}
		}
		txs = filteredTxs
	}

	if len(txs) == 0 {
		return []PerformancePoint{}, nil
	}

	// Ordena cronologicamente do mais antigo para o mais novo
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].ExecutedAt.Before(txs[j].ExecutedAt)
	})

	// Determina janela temporal da consulta
	endDate := time.Now()
	var startDate time.Time
	switch strings.ToUpper(period) {
	case "1M":
		startDate = endDate.AddDate(0, -1, 0)
	case "3M":
		startDate = endDate.AddDate(0, -3, 0)
	case "6M":
		startDate = endDate.AddDate(0, -6, 0)
	case "1Y":
		startDate = endDate.AddDate(-1, 0, 0)
	default: // "ALL" ou padrão
		startDate = txs[0].ExecutedAt
	}

	// Cobre o caso de a data inicial ser posterior à primeira transação
	if startDate.After(txs[0].ExecutedAt) {
		startDate = txs[0].ExecutedAt
	}

	// Busca histórico de cotações diárias para cada ativo envolvido na carteira
	assetIDsMap := make(map[string]bool)
	hasUSDAsset := false
	for _, tx := range txs {
		assetIDsMap[tx.AssetID] = true
		if tx.Currency == "USD" {
			hasUSDAsset = true
		}
	}

	// Injeta USDBRL=X se houver ativos em USD e carteira BRL
	var usdBrlID string
	if hasUSDAsset && p.BaseCurrency == "BRL" {
		id, err := s.repo.GetAssetByTicker(ctx, "USDBRL=X")
		if err == nil {
			usdBrlID = id
			assetIDsMap[id] = true
		}
	}

	// Mapeia preços históricos no formato: pricesMap[asset_id][date_string] = close_price
	pricesMap := make(map[string]map[string]float64)

	// Prepara a lista de IDs para buscar em batch
	var assetIDsToFetch []string
	for assetID := range assetIDsMap {
		pricesMap[assetID] = make(map[string]float64)
		assetIDsToFetch = append(assetIDsToFetch, assetID)
	}

	// Faz uma única query (batch) buscando todos os preços históricos da carteira
	histBatch, err := s.repo.GetDailyPricesBatch(ctx, assetIDsToFetch, txs[0].ExecutedAt, endDate)
	if err == nil {
		for _, dp := range histBatch {
			if pricesMap[dp.AssetID] == nil {
				pricesMap[dp.AssetID] = make(map[string]float64)
			}
			pricesMap[dp.AssetID][dp.PriceDate.Format("2006-01-02")] = dp.ClosePrice
		}
	}

	// Reconstrói a linha do tempo dia a dia aplicando LOCF
	var points []PerformancePoint
	currDate := startDate

	// LOCF helper
	getPriceLOCF := func(assetID string, d time.Time) float64 {
		chk := d
		for i := 0; i < 100; i++ {
			dateStr := chk.Format("2006-01-02")
			if val, ok := pricesMap[assetID][dateStr]; ok && val > 0 {
				return val
			}
			chk = chk.AddDate(0, 0, -1)
		}
		return 0.0
	}

	// Precomputa fatores acumulados de split/reverse-split futuros: O(T)
	futureSplitFactors := make(map[string]float64)
	for _, tx := range txs {
		if (tx.Type == "SPLIT" || tx.Type == "REVERSE_SPLIT") && tx.Quantity > 0 {
			if futureSplitFactors[tx.AssetID] == 0 {
				futureSplitFactors[tx.AssetID] = 1.0
			}
			if tx.Type == "SPLIT" {
				futureSplitFactors[tx.AssetID] *= tx.Quantity
			} else {
				futureSplitFactors[tx.AssetID] /= tx.Quantity
			}
		}
	}

	// Estruturas de controle de posição acumulada
	runningQuantities := make(map[string]float64)
	runningCosts := make(map[string]float64)
	dailyCurrencies := make(map[string]string)
	dailyTickers := make(map[string]string)

	txIdx := 0

	for !currDate.After(endDate) {
		dayStr := currDate.Format("2006-01-02")

		// Avança o ponteiro de transações consumindo todas as transações até currDate: O(T) amortizado
		for txIdx < len(txs) && !txs[txIdx].ExecutedAt.After(currDate) {
			tx := txs[txIdx]

			dailyCurrencies[tx.AssetID] = tx.Currency
			dailyTickers[tx.AssetID] = tx.Ticker

			rate := tx.ExchangeRate
			if rate <= calculator.FinancialEpsilon {
				rate = 1.0
			}

			if tx.Type == "BUY" || tx.Type == "BONUS" {
				runningQuantities[tx.AssetID] += tx.Quantity
				runningCosts[tx.AssetID] += ((tx.Quantity * tx.UnitPrice) + tx.Fee) * rate
			} else if tx.Type == "SELL" {
				if runningQuantities[tx.AssetID] >= tx.Quantity {
					runningQuantities[tx.AssetID] -= tx.Quantity
					// Reduz o custo proporcionalmente
					runningCosts[tx.AssetID] = runningQuantities[tx.AssetID] * (runningCosts[tx.AssetID] / (runningQuantities[tx.AssetID] + tx.Quantity))
				} else {
					runningQuantities[tx.AssetID] = 0
					runningCosts[tx.AssetID] = 0
				}
			} else if tx.Type == "SPLIT" {
				if runningQuantities[tx.AssetID] > 0 && tx.Quantity > 0 {
					runningQuantities[tx.AssetID] = runningQuantities[tx.AssetID] * tx.Quantity
				}
				// O split agora ocorreu no passado, remove do multiplicador futuro
				if tx.Quantity > 0 && futureSplitFactors[tx.AssetID] > 0 {
					futureSplitFactors[tx.AssetID] /= tx.Quantity
				}
			} else if tx.Type == "REVERSE_SPLIT" {
				if runningQuantities[tx.AssetID] > 0 && tx.Quantity > 0 {
					runningQuantities[tx.AssetID] = math.Floor(runningQuantities[tx.AssetID] / tx.Quantity)
				}
				// O grupamento agora ocorreu no passado, remove do multiplicador futuro
				if tx.Quantity > 0 && futureSplitFactors[tx.AssetID] > 0 {
					futureSplitFactors[tx.AssetID] *= tx.Quantity
				}
			}

			txIdx++
		}

		// Calcula valor total de mercado e custo investido para a data analisada
		var totalMarketValue float64
		var totalInvested float64

		for assetID, qty := range runningQuantities {
			if qty > 0 {
				price := getPriceLOCF(assetID, currDate)
				cost := runningCosts[assetID]

				// Se o preço não for encontrado, usa o custo médio de aquisição como fallback temporário
				if math.Abs(price) < calculator.FinancialEpsilon && qty > 0 {
					price = cost / qty
				}

				// Taxa cambial do dia
				rate := 1.0
				if dailyCurrencies[assetID] != p.BaseCurrency && p.BaseCurrency == "BRL" && usdBrlID != "" {
					rate = getPriceLOCF(usdBrlID, currDate)
					if math.Abs(rate) < calculator.FinancialEpsilon {
						rate = 5.0 // Fallback seguro
					}
				}

				adjFactor := futureSplitFactors[assetID]
				if math.Abs(adjFactor) < calculator.FinancialEpsilon {
					adjFactor = 1.0
				}
				adjustedQty := qty * adjFactor

				totalMarketValue += adjustedQty * price * rate
				totalInvested += cost
			}
		}

		points = append(points, PerformancePoint{
			Date:          dayStr,
			Value:         totalMarketValue,
			TotalInvested: totalInvested,
		})

		currDate = currDate.AddDate(0, 0, 1)
	}

	// Filtra a janela final de exibição com base no período solicitado pelo usuário
	var finalPoints []PerformancePoint
	limitDate := time.Now()
	switch strings.ToUpper(period) {
	case "1M":
		limitDate = time.Now().AddDate(0, -1, 0)
	case "3M":
		limitDate = time.Now().AddDate(0, -3, 0)
	case "6M":
		limitDate = time.Now().AddDate(0, -6, 0)
	case "1Y":
		limitDate = time.Now().AddDate(-1, 0, 0)
	default:
		limitDate = startDate
	}

	for _, pt := range points {
		ptDate, err := time.Parse("2006-01-02", pt.Date)
		if err == nil && !ptDate.Before(limitDate) {
			finalPoints = append(finalPoints, pt)
		}
	}

	if len(finalPoints) == 0 {
		return finalPoints, nil
	}

	// Precomputa o fluxo de caixa diário das transações: O(T)
	dailyCashFlowMap := make(map[string]float64)
	for _, tx := range txs {
		rate := tx.ExchangeRate
		if rate <= calculator.FinancialEpsilon {
			rate = 1.0
		}
		dateKey := tx.ExecutedAt.Format("2006-01-02")
		if tx.Type == "BUY" {
			dailyCashFlowMap[dateKey] += tx.TotalCost * rate
		} else if tx.Type == "SELL" {
			dailyCashFlowMap[dateKey] -= tx.TotalCost * rate
		}
	}

	// 1. TWRR da Carteira dentro da janela finalPoints: O(D)
	var cumulativeTWRR float64 = 0.0
	var prevValue float64 = 0.0

	for idx := range finalPoints {
		pt := &finalPoints[idx]
		dailyCashFlow := dailyCashFlowMap[pt.Date]

		var dailyReturn float64 = 0.0
		if idx > 0 && prevValue > calculator.FinancialEpsilon {
			// Retorno diário isolando fluxos de caixa externos (TWRR)
			dailyReturn = (pt.Value - dailyCashFlow - prevValue) / prevValue
		}

		cumulativeTWRR = (1+cumulativeTWRR)*(1+dailyReturn) - 1
		pt.ReturnPct = cumulativeTWRR * 100.0
		prevValue = pt.Value
	}

	// 2. Cálculo dos Benchmarks Reais
	startT, err := time.Parse("2006-01-02", finalPoints[0].Date)
	if err == nil {
		endT, err := time.Parse("2006-01-02", finalPoints[len(finalPoints)-1].Date)
		if err == nil {
			// Busca as taxas no banco de dados via fiService
			var cdiRatesRaw, ipcaRatesRaw, ifixRatesRaw, ibovRatesRaw, sp500RatesRaw []fixedincome.IndexRate
			if s.fiService != nil {
				var errIdx error
				cdiRatesRaw, errIdx = s.fiService.GetIndexRates(ctx, "CDI", startT, endT)
				if errIdx != nil {
					log.Printf("portfolio performance: benchmark CDI indisponível (%v) — execute o sync de índices", errIdx)
				}
				ipcaRatesRaw, errIdx = s.fiService.GetIndexRates(ctx, "IPCA", startT, endT)
				if errIdx != nil {
					log.Printf("portfolio performance: benchmark IPCA indisponível (%v)", errIdx)
				}
				ifixRatesRaw, errIdx = s.fiService.GetIndexRates(ctx, "IFIX", startT, endT)
				if errIdx != nil {
					log.Printf("portfolio performance: benchmark IFIX indisponível (%v)", errIdx)
				}
				ibovRatesRaw, errIdx = s.fiService.GetIndexRates(ctx, "IBOV", startT, endT)
				if errIdx != nil {
					log.Printf("portfolio performance: benchmark IBOV indisponível (%v)", errIdx)
				}
				sp500RatesRaw, errIdx = s.fiService.GetIndexRates(ctx, "SP500", startT, endT)
				if errIdx != nil {
					log.Printf("portfolio performance: benchmark SP500 indisponível (%v)", errIdx)
				}
			}

			cdiMap := make(map[string]float64)

			for _, r := range cdiRatesRaw {
				cdiMap[r.Date.Format("2006-01-02")] = r.Rate
			}

			ipcaMap := make(map[string]float64)
			var latestIpcaRate float64
			var latestIpcaDate string
			for _, r := range ipcaRatesRaw {
				key := r.Date.Format("2006-01-02")
				ipcaMap[key] = r.Rate
				if key > latestIpcaDate {
					latestIpcaDate = key
					latestIpcaRate = r.Rate
				}
			}

			ifixMap := make(map[string]float64)
			for _, r := range ifixRatesRaw {
				ifixMap[r.Date.Format("2006-01-02")] = r.Rate
			}

			ibovMap := make(map[string]float64)
			for _, r := range ibovRatesRaw {
				ibovMap[r.Date.Format("2006-01-02")] = r.Rate
			}

			sp500Map := make(map[string]float64)
			for _, r := range sp500RatesRaw {
				sp500Map[r.Date.Format("2006-01-02")] = r.Rate
			}

			// Carrega câmbio USD/BRL se a carteira for BRL
			usdBrlMap := make(map[string]float64)
			if p.BaseCurrency == "BRL" {
				usdBrlID, err := s.repo.GetAssetByTicker(ctx, "USDBRL=X")
				if err == nil {
					histPrices, err := s.repo.GetDailyPrices(ctx, usdBrlID, startT, endT)
					if err == nil {
						for _, dp := range histPrices {
							usdBrlMap[dp.PriceDate.Format("2006-01-02")] = dp.ClosePrice
						}
					}
				}
			}

			// Helper LOCF para cotações (evitar falha em finais de semana/feriados)
			getIndexLOCF := func(d time.Time, rates map[string]float64) float64 {
				chk := d
				for i := 0; i < 100; i++ {
					dateStr := chk.Format("2006-01-02")
					if val, ok := rates[dateStr]; ok && val > 0 {
						return val
					}
					chk = chk.AddDate(0, 0, -1)
				}
				return 0.0
			}

			getUsdBrlLOCF := func(d time.Time) float64 {
				chk := d
				for i := 0; i < 100; i++ {
					dateStr := chk.Format("2006-01-02")
					if val, ok := usdBrlMap[dateStr]; ok && val > 0 {
						return val
					}
					chk = chk.AddDate(0, 0, -1)
				}
				return 1.0
			}

			countBusinessDaysInMonth := func(year int, month time.Month) int {
				days := 0
				t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
				for t.Month() == month {
					wd := t.Weekday()
					if wd != time.Saturday && wd != time.Sunday {
						days++
					}
					t = t.AddDate(0, 0, 1)
				}
				return days
			}

			// Valores iniciais no dia t_0 (startT)
			ifixInit := getIndexLOCF(startT, ifixMap)
			ibovInit := getIndexLOCF(startT, ibovMap)
			sp500Init := getIndexLOCF(startT, sp500Map)
			sp500InitBrl := sp500Init * getUsdBrlLOCF(startT)

			cdiCumulativeFactor := 1.0
			ipcaCumulativeFactor := 1.0

			for idx := range finalPoints {
				pt := &finalPoints[idx]
				d, _ := time.Parse("2006-01-02", pt.Date)

				if idx > 0 {
					// CDI: acumula fator diário apenas em dias úteis
					wd := d.Weekday()
					if wd != time.Saturday && wd != time.Sunday {
						cdiRate := cdiMap[pt.Date]
						cdiCumulativeFactor *= 1.0 + (cdiRate / 100.0)
					}

					// IPCA: pro-rata diário por dias úteis
					if wd != time.Saturday && wd != time.Sunday {
						monthKey := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
						rMonthly, ok := ipcaMap[monthKey]
						if !ok {
							rMonthly = latestIpcaRate // fallback para último IPCA conhecido
						}
						busDays := countBusinessDaysInMonth(d.Year(), d.Month())
						if busDays > 0 {
							dailyFactor := math.Pow(1.0+(rMonthly/100.0), 1.0/float64(busDays))
							ipcaCumulativeFactor *= dailyFactor
						}
					}
				}

				pt.CdiReturnPct = (cdiCumulativeFactor - 1.0) * 100.0
				pt.IpcaReturnPct = (ipcaCumulativeFactor - 1.0) * 100.0

				// IFIX
				if ifixInit > calculator.FinancialEpsilon {
					ifixVal := getIndexLOCF(d, ifixMap)
					if ifixVal > 0 {
						pt.IfixReturnPct = ((ifixVal - ifixInit) / ifixInit) * 100.0
					}
				}

				// Ibovespa
				if ibovInit > calculator.FinancialEpsilon {
					ibovVal := getIndexLOCF(d, ibovMap)
					if ibovVal > 0 {
						pt.IbovReturnPct = ((ibovVal - ibovInit) / ibovInit) * 100.0
					}
				}

				// S&P 500
				if sp500Init > calculator.FinancialEpsilon {
					sp500Val := getIndexLOCF(d, sp500Map)
					if sp500Val > 0 {
						if p.BaseCurrency == "BRL" {
							sp500ValBrl := sp500Val * getUsdBrlLOCF(d)
							if sp500InitBrl > calculator.FinancialEpsilon {
								pt.Sp500ReturnPct = ((sp500ValBrl - sp500InitBrl) / sp500InitBrl) * 100.0
							}
						} else {
							pt.Sp500ReturnPct = ((sp500Val - sp500Init) / sp500Init) * 100.0
						}
					}
				}
			}
		}
	}

	return finalPoints, nil
}
