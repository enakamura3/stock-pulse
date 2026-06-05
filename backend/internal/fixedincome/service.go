package fixedincome

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

type Service interface {
	CreateAsset(ctx context.Context, asset *Asset) (*Asset, error)
	GetPortfolioPositions(ctx context.Context, portfolioID string) ([]Position, error)
	GetPortfolioPerformance(ctx context.Context, portfolioID string, period string) ([]PerformancePoint, error)
	GetAssetPosition(ctx context.Context, assetID string) (*Position, error)
	CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error)
	TriggerBackfill(ctx context.Context, indexer string, startDate time.Time)
}

type service struct {
	repo      Repository
	bcbClient BCBClient
}

func NewService(repo Repository, bcbClient BCBClient) Service {
	return &service{
		repo:      repo,
		bcbClient: bcbClient,
	}
}

func (s *service) CreateAsset(ctx context.Context, asset *Asset) (*Asset, error) {
	created, err := s.repo.CreateAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *service) CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	created, err := s.repo.CreateTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Trigger backfill se for aplicação antiga (APLICACAO)
	if created.Type == "APLICACAO" {
		asset, err := s.repo.GetAssetByID(ctx, tx.AssetID)
		if err == nil && (asset.DebtType == "POS" || asset.DebtType == "HIBRIDO") {
			go s.TriggerBackfill(context.Background(), asset.Indexer, tx.Date)
		}
	}

	return created, nil
}

func (s *service) TriggerBackfill(ctx context.Context, indexer string, startDate time.Time) {
	// Pega até a data atual
	endDate := time.Now()
	
	// Checa se já temos dados recentes para evitar request desnecessário
	latest, _ := s.repo.GetLatestIndexRate(ctx, indexer)
	if latest != nil && latest.Date.After(endDate.AddDate(0, 0, -2)) {
		// Se já temos dado do penultimo dia, e a startDate for mais recente que o histórico?
		// Vamos puxar do startDate até endDate pra garantir
	}

	rates, err := s.bcbClient.FetchRates(ctx, indexer, startDate, endDate)
	if err == nil && len(rates) > 0 {
		s.repo.SaveIndexRates(ctx, rates)
	}
}

func calculateIOF(days int) float64 {
	// Tabela regressiva de IOF para os primeiros 29 dias
	if days >= 30 {
		return 0.0
	}
	iofRates := []float64{
		100, 96, 93, 90, 86, 83, 80, 76, 73, 70, 66, 63, 60, 56, 53, 50, 46, 43, 40, 36, 33, 30, 26, 23, 20, 16, 13, 10, 6, 3, 0,
	}
	if days < 0 {
		return 100.0
	}
	return iofRates[days] / 100.0
}

func calculateIRRate(days int) float64 {
	// Tabela regressiva IR Renda Fixa
	if days <= 180 {
		return 0.225
	} else if days <= 360 {
		return 0.20
	} else if days <= 720 {
		return 0.175
	}
	return 0.15
}

func (s *service) GetPortfolioPositions(ctx context.Context, portfolioID string) ([]Position, error) {
	assets, err := s.repo.GetAssetsByPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	var positions []Position
	for _, asset := range assets {
		pos, err := s.GetAssetPosition(ctx, asset.ID)
		if err == nil && pos != nil && pos.TotalInvested > 0 {
			positions = append(positions, *pos)
		}
	}
	return positions, nil
}

func (s *service) GetAssetPosition(ctx context.Context, assetID string) (*Position, error) {
	pos, _, _, err := s.getAssetPositionWithHistory(ctx, assetID)
	return pos, err
}

func (s *service) getAssetPositionWithHistory(ctx context.Context, assetID string) (*Position, map[string]float64, map[string]float64, error) {
	asset, err := s.repo.GetAssetByID(ctx, assetID)
	if err != nil {
		return nil, nil, nil, err
	}

	txs, err := s.repo.GetTransactionsByAsset(ctx, assetID)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(txs) == 0 {
		return nil, nil, nil, nil
	}

	// Ordena txs cronologicamente
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Date.Before(txs[j].Date)
	})

	var totalInvested float64
	var grossValue float64
	var currentQty float64

	startDate := txs[0].Date
	today := time.Now()
	dailyNet := make(map[string]float64)
	dailyInv := make(map[string]float64)

	// Se o titulo venceu, a rentabilidade para na maturity date
	limitDate := today
	isMatured := false
	if !asset.MaturityDate.IsZero() && today.After(asset.MaturityDate) {
		limitDate = asset.MaturityDate
		isMatured = true
	}

	daysToMaturity := int(asset.MaturityDate.Sub(today).Hours() / 24)
	if daysToMaturity < 0 {
		daysToMaturity = 0
	}

	// Para POS, precisamos das taxas do BCB
	var indexRates map[string]float64
	if asset.DebtType == "POS" || asset.DebtType == "HIBRIDO" {
		indexRates = make(map[string]float64)
		rates, _ := s.repo.GetIndexRates(ctx, asset.Indexer, startDate, limitDate)
		for _, r := range rates {
			indexRates[r.Date.Format("2006-01-02")] = r.Rate
		}
	}

	// Simulação dia a dia (Simplificada para iterar sobre os dias corridos)
	currDate := startDate
	txIndex := 0

	for !currDate.After(limitDate) {
		// Aplica transações do dia
		for txIndex < len(txs) && (txs[txIndex].Date.Before(currDate) || txs[txIndex].Date.Equal(currDate)) {
			tx := txs[txIndex]
			if tx.Type == "SUBSCRIPTION" {
				totalInvested += tx.Amount
				grossValue += tx.Amount
				currentQty += tx.Amount
			} else if tx.Type == "REDEMPTION" {
				if grossValue > 0 {
					// Reduz o principal proporcionalmente ao resgate
					withdrawalRatio := tx.Amount / grossValue
					if withdrawalRatio > 1 {
						withdrawalRatio = 1
					}
					totalInvested -= totalInvested * withdrawalRatio
					grossValue -= tx.Amount
					currentQty -= tx.Amount
				}
			}
			txIndex++
		}

		if currentQty <= 0 {
			grossValue = 0
			totalInvested = 0
		} else {
			// Aplica rentabilidade do dia (se não for final de semana, no caso do CDI que só tem em dia util)
			// Mas se for PRE, aplica proporcional aos dias uteis. Para simplificar no pre, usamos dias uteis.
			// Verifica se é dia util (simplificação: seg a sex)
			if currDate.Weekday() != time.Saturday && currDate.Weekday() != time.Sunday {
				if asset.DebtType == "PRE" {
					// Formula = Capital * (1 + TaxaAnual)^(1/252)
					dailyFactor := math.Pow(1+(asset.Rate/100), 1.0/252.0)
					grossValue = grossValue * dailyFactor
				} else if asset.DebtType == "POS" {
					// Busca a taxa no map, ou usa um fallback de 0.04% ao dia (aprox 10.5% a.a.) se faltar no banco
					rate, ok := indexRates[currDate.Format("2006-01-02")]
					if !ok {
						rate = 0.04
					}
					// Taxa CDI Diária já está em % (ex: 0.043739)
					// Fator = 1 + (RateBCB / 100) * (RateContratada / 100)
					dailyFactor := 1 + (rate/100)*(asset.Rate/100)
					grossValue = grossValue * dailyFactor
				} else if asset.DebtType == "HIBRIDO" {
					// IPCA + PRE (IPCA costuma ser mensal, exigiria uma lógica de IPCA pro-rata, 
					// simplificando aqui para a Taxa PRE ao dia. Num cenario real, teriamos que usar IPCA do mes)
					dailyFactor := math.Pow(1+(asset.Rate/100), 1.0/252.0)
					grossValue = grossValue * dailyFactor
					// + IPCA se tiver
					rate, ok := indexRates[currDate.Format("2006-01-02")]
					if !ok {
						rate = 0.015 // Fallback IPCA diario aproximado
					}
					ipcaFactor := 1 + (rate / 100)
					grossValue = grossValue * ipcaFactor
				}
			}
		}

		dateStr := currDate.Format("2006-01-02")
		dailyInv[dateStr] = totalInvested
		dailyNet[dateStr] = grossValue // using gross value in history for simplicity, taxes calculated at the end.

		currDate = currDate.AddDate(0, 0, 1)
	}

	profit := grossValue - totalInvested
	if profit < 0 {
		profit = 0
	}

	daysHeld := int(limitDate.Sub(startDate).Hours() / 24)
	
	// IR e IOF
	taxes := 0.0
	isTaxExempt := asset.Type == "LCI" || asset.Type == "LCA"
	if !isTaxExempt && profit > 0 {
		iofAmount := profit * calculateIOF(daysHeld)
		remainingProfit := profit - iofAmount
		irAmount := remainingProfit * calculateIRRate(daysHeld)
		taxes = iofAmount + irAmount
	}

	netValue := grossValue - taxes

	netReturnPercent := 0.0
	if totalInvested > 0 {
		netReturnPercent = ((netValue / totalInvested) - 1) * 100
	}

	return &Position{
		Asset:            *asset,
		StartDate:        startDate,
		TotalInvested:    totalInvested,
		GrossValue:       grossValue,
		NetValue:         netValue,
		NetReturnPercent: netReturnPercent,
		IsMatured:        isMatured,
		DaysToMaturity:   daysToMaturity,
		TaxesCalculated:  taxes,
	}, dailyNet, dailyInv, nil
}

// GetPortfolioPerformance constrói a série histórica consolidada de todos os ativos de Renda Fixa.
func (s *service) GetPortfolioPerformance(ctx context.Context, portfolioID string, period string) ([]PerformancePoint, error) {
	assets, err := s.repo.GetAssetsByPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		return []PerformancePoint{}, nil
	}

	dailyValues := make(map[string]float64)
	dailyInvested := make(map[string]float64)
	
	earliestDate := time.Now()
	for _, a := range assets {
		txs, _ := s.repo.GetTransactionsByAsset(ctx, a.ID)
		if len(txs) > 0 && txs[0].Date.Before(earliestDate) {
			earliestDate = txs[0].Date
		}
	}

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
	default:
		startDate = earliestDate
	}

	if startDate.After(earliestDate) {
		startDate = earliestDate
	}

	for _, a := range assets {
		pos, histNet, histInv, err := s.getAssetPositionWithHistory(ctx, a.ID)
		if err == nil && pos != nil {
			for dateStr, netVal := range histNet {
				dailyValues[dateStr] += netVal
				dailyInvested[dateStr] += histInv[dateStr]
			}
		}
	}

	var points []PerformancePoint
	currDate := startDate
	
	for !currDate.After(endDate) {
		dateStr := currDate.Format("2006-01-02")
		val := dailyValues[dateStr]
		inv := dailyInvested[dateStr]
		
		if val == 0 && inv == 0 && len(points) > 0 {
			val = points[len(points)-1].Value
			inv = points[len(points)-1].TotalInvested
		}

		points = append(points, PerformancePoint{
			Date:          dateStr,
			Value:         val,
			TotalInvested: inv,
		})
		currDate = currDate.AddDate(0, 0, 1)
	}

	return points, nil
}
