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
		txCost := ((txQty * txUnitPrice) + txFee) * fxRate
		if FloatIsZero(currentQty) {
			newQty = txQty
			newTotalCost = txCost
			newAvgPrice = newTotalCost / (newQty * fxRate)
		} else {
			newQty = currentQty + txQty
			newTotalCost = currentTotalCost + txCost
			newAvgPrice = newTotalCost / (newQty * fxRate)
		}

	case "SELL":
		if currentQty >= txQty {
			newQty = currentQty - txQty
			newTotalCost = newQty * currentAvgPrice * fxRate
			newAvgPrice = currentAvgPrice
		} else {
			// Venda acima do saldo zera a posição
			newQty = 0
			newTotalCost = 0
			newAvgPrice = 0
		}

	case "SPLIT":
		if currentQty > 0 && txQty > 0 {
			newQty = currentQty * txQty
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice / txQty
		} else {
			newQty = currentQty
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice
		}

	case "REVERSE_SPLIT":
		if currentQty > 0 && txQty > 0 {
			newQty = math.Floor(currentQty / txQty)
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice * txQty
		} else {
			newQty = currentQty
			newTotalCost = currentTotalCost
			newAvgPrice = currentAvgPrice
		}

	case "BONUS":
		txCost := (txQty * txUnitPrice * fxRate)
		newQty = currentQty + txQty
		newTotalCost = currentTotalCost + txCost
		if newQty > 0 {
			newAvgPrice = newTotalCost / (newQty * fxRate)
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
