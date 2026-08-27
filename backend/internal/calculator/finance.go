package calculator

import (
	"math"
	"strings"
)

// FloatEquals compara dois floats com margem de tolerância financeira (1e-6).
func FloatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// FloatIsZero verifica se um float é matematicamente zero (dentro da tolerância).
func FloatIsZero(v float64) bool {
	return math.Abs(v) < 1e-6
}

// DetermineAssetType define a categoria oficial do ativo (STOCK_BR, FII, FIAGRO, ETF_BR, BDR, STOCK_US, ETF_US, CRYPTO).
func DetermineAssetType(ticker, name, currency string) string {
	if strings.Contains(ticker, "-") {
		return "CRYPTO"
	}

	if !strings.HasSuffix(ticker, ".SA") {
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "etf") || strings.Contains(lowerName, "trust") || strings.Contains(lowerName, "fund") {
			return "ETF_US"
		}
		return "STOCK_US"
	}

	// É do Brasil (.SA)
	if strings.HasSuffix(ticker, "34.SA") || strings.HasSuffix(ticker, "35.SA") || strings.HasSuffix(ticker, "39.SA") {
		return "BDR"
	}

	if strings.HasSuffix(ticker, "11.SA") {
		lowerName := strings.ToLower(name)
		isEtf := strings.Contains(lowerName, "etf") || strings.Contains(lowerName, "ishares") || strings.Contains(lowerName, "índice") || strings.Contains(lowerName, "indice")
		isFiagro := strings.Contains(lowerName, "fiagro") || strings.Contains(lowerName, "agro")
		isFii := strings.Contains(lowerName, "fii") || strings.Contains(lowerName, "fundo") || strings.Contains(lowerName, "fdo") || strings.Contains(lowerName, "imob") || strings.Contains(lowerName, "lajes") || strings.Contains(lowerName, "shopping")

		if isEtf {
			return "ETF_BR"
		}
		if isFiagro {
			return "FIAGRO"
		}

		tickerUpper := strings.ToUpper(ticker)
		if isFii || tickerUpper == "MXRF11.SA" || tickerUpper == "HGLG11.SA" || tickerUpper == "KNRI11.SA" || tickerUpper == "BTLG11.SA" || tickerUpper == "XPML11.SA" || tickerUpper == "VISC11.SA" {
			return "FII"
		}
		if tickerUpper == "SPYI11.SA" || tickerUpper == "QQQI11.SA" || tickerUpper == "IVVB11.SA" || tickerUpper == "NASD11.SA" || tickerUpper == "BOVA11.SA" {
			return "ETF_BR"
		}
	}

	return "STOCK_BR"
}

// UpdatePositionOnTransaction calcula a nova quantidade, custo total e preço médio da posição após uma operação.
func UpdatePositionOnTransaction(
	currentQty, currentTotalCost, currentAvgPrice float64,
	txType string,
	txQty, txUnitPrice, fxRate float64,
	txFee float64,
) (newQty, newTotalCost, newAvgPrice float64) {
	if FloatIsZero(fxRate) {
		fxRate = 1.0
	}

	switch txType {
	case "BUY":
		nativeCost := (txQty * txUnitPrice) + txFee
		txCostInBase := nativeCost * fxRate
		newQty = currentQty + txQty
		newTotalCost = currentTotalCost + txCostInBase
		if newQty > 1e-6 {
			newAvgPrice = ((currentQty * currentAvgPrice) + nativeCost) / newQty
		} else {
			newAvgPrice = txUnitPrice
		}

	case "SELL":
		remainingQty := currentQty - txQty
		if remainingQty > 1e-6 && currentQty > 1e-6 {
			newQty = remainingQty
			newTotalCost = currentTotalCost * (remainingQty / currentQty)
			newAvgPrice = currentAvgPrice
		} else {
			// Venda total ou acima do saldo zera a posição
			newQty = 0
			newTotalCost = 0
			newAvgPrice = 0
		}

	case "SPLIT":
		if currentQty > 1e-6 && txQty > 1e-6 {
			newQty = currentQty * txQty
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice / txQty
		} else {
			newQty = currentQty
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice
		}

	case "REVERSE_SPLIT":
		if currentQty > 1e-6 && txQty > 1e-6 {
			newQty = math.Floor(currentQty / txQty)
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice * txQty
		} else {
			newQty = currentQty
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice
		}

	case "BONUS":
		nativeCost := txQty * txUnitPrice
		txCostInBase := nativeCost * fxRate
		newQty = currentQty + txQty
		newTotalCost = currentTotalCost + txCostInBase
		if newQty > 1e-6 {
			newAvgPrice = ((currentQty * currentAvgPrice) + nativeCost) / newQty
		} else {
			newAvgPrice = currentAvgPrice
		}

	default:
		newQty = currentQty
		newTotalCost = currentTotalCost
		newAvgPrice = currentAvgPrice
	}

	return newQty, newTotalCost, newAvgPrice
}

// CalculatePositionMetrics calcula o valor atual, lucro/prejuízo e percentual de retorno de uma posição.
func CalculatePositionMetrics(quantity, currentPrice, totalCost, fxRate float64) (currentValue, profitLoss, returnPercent float64) {
	if FloatIsZero(fxRate) {
		fxRate = 1.0
	}

	currentValue = quantity * currentPrice * fxRate
	profitLoss = currentValue - totalCost

	if totalCost > 1e-6 {
		returnPercent = (profitLoss / totalCost) * 100.0
	} else {
		returnPercent = 0.0
	}

	return currentValue, profitLoss, returnPercent
}

// CalculateValuationRatios calcula os múltiplos fundamentalistas P/VP e P/L (P/E).
func CalculateValuationRatios(currentPrice, bookValue, eps float64) (pvp, pe float64) {
	if bookValue > 1e-6 {
		pvp = currentPrice / bookValue
	}
	if eps > 1e-6 {
		pe = currentPrice / eps
	}
	return pvp, pe
}

// CalculateTWRR calcula o Time-Weighted Rate of Return para uma série de retornos periódicos.
func CalculateTWRR(periodReturns []float64) float64 {
	if len(periodReturns) == 0 {
		return 0.0
	}
	twrr := 1.0
	for _, r := range periodReturns {
		twrr *= (1.0 + r)
	}
	return (twrr - 1.0) * 100.0
}

// CalculateDailyFixedIncomeRate calcula a taxa diária equivalente (base 252 dias úteis) para um título de renda fixa.
func CalculateDailyFixedIncomeRate(indexer string, rate float64, benchmarkAnnualRate float64) float64 {
	indexerUpper := strings.ToUpper(indexer)
	var effectiveAnnualRate float64

	switch {
	case indexerUpper == "PREFIXADO" || indexerUpper == "PRE":
		effectiveAnnualRate = rate / 100.0
	case indexerUpper == "CDI" || indexerUpper == "POS":
		effectiveAnnualRate = (benchmarkAnnualRate / 100.0) * (rate / 100.0)
	case indexerUpper == "SELIC":
		if benchmarkAnnualRate > 1e-6 {
			effectiveAnnualRate = benchmarkAnnualRate / 100.0
		} else {
			effectiveAnnualRate = rate / 100.0
		}
	case indexerUpper == "IPCA" || indexerUpper == "HIBRIDO":
		effectiveAnnualRate = rate / 100.0
	default:
		effectiveAnnualRate = rate / 100.0
	}

	if effectiveAnnualRate <= -1.0 {
		return 0.0
	}

	// r_dia = (1 + r_anual)^(1/252) - 1
	dailyRate := math.Pow(1.0+effectiveAnnualRate, 1.0/252.0) - 1.0
	return dailyRate
}

// CalculateEstimatedDailyGain calcula o ganho financeiro estimado em 1 dia útil para a posição de renda fixa.
func CalculateEstimatedDailyGain(netValue float64, dailyRate float64) float64 {
	if netValue < 1e-6 || dailyRate < 1e-6 {
		return 0.0
	}
	return netValue * dailyRate
}
