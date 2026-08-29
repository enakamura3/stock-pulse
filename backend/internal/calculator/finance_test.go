package calculator

import (
	"testing"
)

func TestFloatEqualsAndIsZero(t *testing.T) {
	if !FloatEquals(0.0000001, 0.0000002) {
		t.Errorf("expected 0.0000001 and 0.0000002 to be equal within 1e-6 tolerance")
	}
	if FloatEquals(1.0, 1.05) {
		t.Errorf("expected 1.0 and 1.05 not to be equal")
	}

	if !FloatIsZero(0.00000005) {
		t.Errorf("expected 0.00000005 to be identified as zero")
	}
	if FloatIsZero(0.1) {
		t.Errorf("expected 0.1 not to be identified as zero")
	}
}

func TestDetermineAssetType(t *testing.T) {
	tests := []struct {
		ticker   string
		name     string
		currency string
		expected string
	}{
		{"BTC-USD", "Bitcoin", "USD", "CRYPTO"},
		{"AAPL", "Apple Inc.", "USD", "STOCK_US"},
		{"VOO", "Vanguard S&P 500 ETF", "USD", "ETF_US"},
		{"SPY", "SPDR S&P 500 Trust", "USD", "ETF_US"},
		{"VTI", "Vanguard Index Fund", "USD", "ETF_US"},
		{"PETR4.SA", "Petroleo Brasileiro SA", "BRL", "STOCK_BR"},
		{"VALE3.SA", "Vale SA", "BRL", "STOCK_BR"},
		{"HGLG11.SA", "CSHG Logistica FII", "BRL", "FII"},
		{"MXRF11.SA", "Maxi Renda FII", "BRL", "FII"},
		{"KNRI11.SA", "Kinea Renda Imobiliaria", "BRL", "FII"},
		{"BTLG11.SA", "BTG Pactual Logistica", "BRL", "FII"},
		{"XPML11.SA", "XP Malls FII", "BRL", "FII"},
		{"VISC11.SA", "Vinci Shopping Centers", "BRL", "FII"},
		{"ABC11.SA", "Fundo Imobiliario Generico", "BRL", "FII"},
		{"RZAG11.SA", "Fundo Fiagro Agro", "BRL", "FIAGRO"},
		{"IVVB11.SA", "iShares S&P 500 Fundo de Indice", "BRL", "ETF_BR"},
		{"SPYI11.SA", "SPYI ETF", "BRL", "ETF_BR"},
		{"QQQI11.SA", "QQQI ETF", "BRL", "ETF_BR"},
		{"NASD11.SA", "Nasdaq 11 ETF", "BRL", "ETF_BR"},
		{"BOVA11.SA", "iShares Ibovespa ETF", "BRL", "ETF_BR"},
		{"ETF11.SA", "ETF Generico", "BRL", "ETF_BR"},
		{"UNKNOWN11.SA", "", "BRL", "FII"},
		{"IND11.SA", "Fundo de Índice", "BRL", "ETF_BR"},
		{"FDO11.SA", "Fdo Imob", "BRL", "FII"},
		{"LAJ11.SA", "Lajes Corporativas", "BRL", "FII"},
		{"SHP11.SA", "Shopping Center Fdo", "BRL", "FII"},
		{"AGRO11.SA", "Agro Investimento", "BRL", "FIAGRO"},
		{"AAPL34.SA", "Apple Inc BDR", "BRL", "BDR"},
		{"MELI35.SA", "Mercado Libre BDR", "BRL", "BDR"},
		{"BERK39.SA", "Berkshire BDR", "BRL", "BDR"},
		{"TAEE11.SA", "Taesa Unit", "BRL", "STOCK_BR"},
	}

	for _, tt := range tests {
		got := DetermineAssetType(tt.ticker, tt.name, tt.currency)
		if got != tt.expected {
			t.Errorf("DetermineAssetType(%s, %s) = %s, want %s", tt.ticker, tt.name, got, tt.expected)
		}
	}
}

func TestUpdatePositionOnTransaction(t *testing.T) {
	// 1. Initial Buy without fee and zero fxRate fallback
	qty, totalCost, avgPrice := UpdatePositionOnTransaction(0, 0, 0, "BUY", 100, 30.0, 0.0, 0.0)
	if qty != 100 || totalCost != 3000.0 || avgPrice != 30.0 {
		t.Errorf("Buy 1 failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 1b. Buy with 0 qty (edge case fallback to txUnitPrice)
	qBuyZero, cBuyZero, aBuyZero := UpdatePositionOnTransaction(0, 0, 0, "BUY", 0, 25.0, 1.0, 0.0)
	if qBuyZero != 0 || cBuyZero != 0 || aBuyZero != 25.0 {
		t.Errorf("Buy zero qty failed: got qty=%.2f, cost=%.2f, avg=%.2f", qBuyZero, cBuyZero, aBuyZero)
	}

	// 2. Second Buy at higher price with R$ 10.00 fee
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "BUY", 100, 40.0, 1.0, 10.0)
	if qty != 200 || totalCost != 7010.0 || avgPrice != 35.05 {
		t.Errorf("Buy 2 with fee failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 3. Partial Sell (Average price should remain 35.05)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SELL", 50, 45.0, 1.0, 5.0)
	if qty != 150 || !FloatEquals(totalCost, 5257.5) || !FloatEquals(avgPrice, 35.05) {
		t.Errorf("Sell failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 4. Split 2:1 (Quantity doubles, avg price halves)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SPLIT", 2.0, 0, 1.0, 0.0)
	if qty != 300 || !FloatEquals(totalCost, 5257.5) || !FloatEquals(avgPrice, 17.525) {
		t.Errorf("Split failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 4b. Split invalid (zero qty)
	qNo, cNo, aNo := UpdatePositionOnTransaction(0, 0, 0, "SPLIT", 0, 0, 1.0, 0.0)
	if qNo != 0 || cNo != 0 || aNo != 0 {
		t.Errorf("Split invalid failed")
	}

	// 5. Reverse Split 10:1 (300 -> 30)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "REVERSE_SPLIT", 10.0, 0, 1.0, 0.0)
	if qty != 30 || !FloatEquals(totalCost, 5257.5) || !FloatEquals(avgPrice, 175.25) {
		t.Errorf("Reverse Split failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 5b. Reverse Split invalid (zero qty)
	qNo2, cNo2, aNo2 := UpdatePositionOnTransaction(0, 0, 0, "REVERSE_SPLIT", 0, 0, 1.0, 0.0)
	if qNo2 != 0 || cNo2 != 0 || aNo2 != 0 {
		t.Errorf("Reverse Split invalid failed")
	}

	// 6. BONUS transaction on empty position
	qB1, cB1, aB1 := UpdatePositionOnTransaction(0, 0, 0, "BONUS", 10, 15.0, 1.0, 0.0)
	if qB1 != 10 || cB1 != 150.0 || aB1 != 15.0 {
		t.Errorf("Bonus 1 failed: got qty=%.2f, cost=%.2f, avg=%.2f", qB1, cB1, aB1)
	}

	// 6b. BONUS transaction on existing position
	qB2, cB2, aB2 := UpdatePositionOnTransaction(10, 150.0, 15.0, "BONUS", 10, 20.0, 1.0, 0.0)
	if qB2 != 20 || cB2 != 350.0 || aB2 != 17.5 {
		t.Errorf("Bonus 2 failed: got qty=%.2f, cost=%.2f, avg=%.2f", qB2, cB2, aB2)
	}

	// 6c. BONUS transaction with 0 new qty
	qB3, cB3, aB3 := UpdatePositionOnTransaction(0, 0, 10.0, "BONUS", 0, 0.0, 1.0, 0.0)
	if qB3 != 0 || cB3 != 0 || aB3 != 10.0 {
		t.Errorf("Bonus 3 failed: got qty=%.2f, cost=%.2f, avg=%.2f", qB3, cB3, aB3)
	}

	// 7. Unknown transaction type (default branch)
	qDef, cDef, aDef := UpdatePositionOnTransaction(10, 100.0, 10.0, "UNKNOWN", 5, 20.0, 1.0, 0.0)
	if qDef != 10 || cDef != 100.0 || aDef != 10.0 {
		t.Errorf("Unknown type failed")
	}

	// 8. Over-Sell zapping position to zero
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SELL", 500, 200.0, 1.0, 0.0)
	if qty != 0 || totalCost != 0 || avgPrice != 0 {
		t.Errorf("Over-sell failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}
}

func TestCalculatePositionMetrics(t *testing.T) {
	// Qty=100, CurrentPrice=40, TotalCost=3000
	currVal, profit, retPct := CalculatePositionMetrics(100, 40.0, 3000.0, 1.0)
	if currVal != 4000.0 || profit != 1000.0 || !FloatEquals(retPct, 33.333333333333336) {
		t.Errorf("Metrics failed: val=%.2f, profit=%.2f, return=%.2f%%", currVal, profit, retPct)
	}

	// Zero fxRate fallback
	currValFx, profitFx, retPctFx := CalculatePositionMetrics(100, 40.0, 3000.0, 0.0)
	if currValFx != 4000.0 || profitFx != 1000.0 || !FloatEquals(retPctFx, 33.333333333333336) {
		t.Errorf("Metrics zero fx failed")
	}

	// Total cost <= 1e-6 (free bonus / zero cost)
	currValZero, profitZero, retPctZero := CalculatePositionMetrics(100, 40.0, 0.0, 1.0)
	if currValZero != 4000.0 || profitZero != 4000.0 || retPctZero != 0.0 {
		t.Errorf("Metrics zero cost failed: retPct=%.2f", retPctZero)
	}
}

func TestCalculateValuationRatios(t *testing.T) {
	pvp, pe := CalculateValuationRatios(30.0, 15.0, 3.0)
	if pvp != 2.0 || pe != 10.0 {
		t.Errorf("Valuation ratios failed: pvp=%.2f, pe=%.2f", pvp, pe)
	}

	pvpZero, peZero := CalculateValuationRatios(30.0, 0.0, 0.0)
	if pvpZero != 0 || peZero != 0 {
		t.Errorf("Valuation ratios zero check failed: pvp=%.2f, pe=%.2f", pvpZero, peZero)
	}
}

func TestCalculateTWRR(t *testing.T) {
	// Empty slice
	if CalculateTWRR([]float64{}) != 0.0 {
		t.Errorf("Empty TWRR should return 0.0")
	}

	// +10% then -5% -> TWRR = 1.10 * 0.95 - 1 = 1.045 - 1 = 4.5%
	twrr := CalculateTWRR([]float64{0.10, -0.05})
	if !FloatEquals(twrr, 4.5) {
		t.Errorf("TWRR failed: got %.4f, want 4.5000", twrr)
	}
}

func TestCalculateDailyFixedIncomeRate(t *testing.T) {
	// Prefixado 12% a.a.: (1 + 0.12)^(1/252) - 1 ≈ 0.000450 (0.045% ao dia)
	ratePre := CalculateDailyFixedIncomeRate("PREFIXADO", 12.0, 0.0)
	if ratePre <= 0 || !FloatEquals(ratePre, 0.0004500583) {
		t.Errorf("Prefixado failed: got %.8f", ratePre)
	}

	// POS / CDI 110% do CDI com CDI anual = 10.40% -> taxa efetiva = 11.44%
	ratePos := CalculateDailyFixedIncomeRate("CDI", 110.0, 10.40)
	if ratePos <= 0 || !FloatEquals(ratePos, 0.0004300405) {
		t.Errorf("CDI failed: got %.8f", ratePos)
	}

	// Selic com Selic anual = 10.50%
	rateSelic := CalculateDailyFixedIncomeRate("SELIC", 100.0, 10.50)
	if rateSelic <= 0 || !FloatEquals(rateSelic, 0.0003969966) {
		t.Errorf("Selic failed: got %.8f", rateSelic)
	}

	// Selic fallback rate
	rateSelicFallback := CalculateDailyFixedIncomeRate("SELIC", 10.50, 0.0)
	if rateSelicFallback <= 0 {
		t.Errorf("Selic fallback failed: got %.8f", rateSelicFallback)
	}

	// IPCA + 6%
	rateIpca := CalculateDailyFixedIncomeRate("IPCA", 6.0, 0.0)
	if rateIpca <= 0 {
		t.Errorf("IPCA failed: got %.8f", rateIpca)
	}

	// Default fallback
	rateDef := CalculateDailyFixedIncomeRate("OTHER", 10.0, 0.0)
	if rateDef <= 0 {
		t.Errorf("Default fallback failed: got %.8f", rateDef)
	}

	// Negative rate <= -100%
	rateNeg := CalculateDailyFixedIncomeRate("PRE", -150.0, 0.0)
	if rateNeg != 0.0 {
		t.Errorf("Negative rate should return 0.0: got %.4f", rateNeg)
	}
}

func TestCalculateEstimatedDailyGain(t *testing.T) {
	dailyRate := 0.00045 // approx 0.045%
	gain := CalculateEstimatedDailyGain(10000.0, dailyRate)
	if !FloatEquals(gain, 4.50) {
		t.Errorf("Estimated daily gain failed: got %.2f, want 4.50", gain)
	}

	// Zero checks
	if CalculateEstimatedDailyGain(0.0, dailyRate) != 0.0 {
		t.Errorf("Zero netValue should return 0.0")
	}
	if CalculateEstimatedDailyGain(10000.0, 0.0) != 0.0 {
		t.Errorf("Zero dailyRate should return 0.0")
	}
}

func TestUpdatePositionOnTransaction_ForeignCurrencyAndETFs(t *testing.T) {
	// 1. First BUY of 10 VOO at $400.00 with $5.00 fee, USD/BRL = 5.00
	qty, totalCost, avgPrice := UpdatePositionOnTransaction(0, 0, 0, "BUY", 10, 400.0, 5.0, 5.0)
	if !FloatEquals(qty, 10) || !FloatEquals(totalCost, 20025.0) || !FloatEquals(avgPrice, 400.50) {
		t.Errorf("Foreign Buy 1 failed: got qty=%.2f, cost=%.2f, avg=%.2f (expected qty=10, cost=20025.0, avg=400.50)", qty, totalCost, avgPrice)
	}

	// 2. Second BUY of 10 VOO at $400.00 with $5.00 fee, USD/BRL = 6.00 (Average price in USD must stay 400.50 regardless of FX rate changes)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "BUY", 10, 400.0, 6.0, 5.0)
	if !FloatEquals(qty, 20) || !FloatEquals(totalCost, 44055.0) || !FloatEquals(avgPrice, 400.50) {
		t.Errorf("Foreign Buy 2 failed: got qty=%.2f, cost=%.2f, avg=%.4f (expected qty=20, cost=44055.0, avg=400.5000)", qty, totalCost, avgPrice)
	}

	// 3. Third BUY of 5 VOO at $500.00 with $10.00 fee, USD/BRL = 5.50
	// Native cost: (5 * 500) + 10 = $2510.00
	// New avg price: (20 * 400.50 + 2510) / 25 = 10520 / 25 = $420.80
	// New BRL cost: 44055.0 + (2510 * 5.50) = 44055.0 + 13805.0 = 57860.0
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "BUY", 5, 500.0, 5.5, 10.0)
	if !FloatEquals(qty, 25) || !FloatEquals(totalCost, 57860.0) || !FloatEquals(avgPrice, 420.80) {
		t.Errorf("Foreign Buy 3 failed: got qty=%.2f, cost=%.2f, avg=%.2f (expected qty=25, cost=57860.0, avg=420.80)", qty, totalCost, avgPrice)
	}

	// 4. Partial SELL of 10 units at $600.00 with $5.00 fee, USD/BRL = 7.00
	// Remaining qty: 15
	// Avg price remains $420.80 in USD
	// Total cost in BRL reduced proportionally: 57860.0 * (15/25) = 34716.0
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SELL", 10, 600.0, 7.0, 5.0)
	if !FloatEquals(qty, 15) || !FloatEquals(totalCost, 34716.0) || !FloatEquals(avgPrice, 420.80) {
		t.Errorf("Foreign Partial Sell failed: got qty=%.2f, cost=%.2f, avg=%.2f (expected qty=15, cost=34716.0, avg=420.80)", qty, totalCost, avgPrice)
	}

	// 5. BONUS of 5 shares at $100.00 with USD/BRL = 5.00
	// New qty: 20
	// Native bonus cost: 5 * 100 = 500.0 USD
	// New avg price: (15 * 420.80 + 500) / 20 = (6312 + 500) / 20 = 6812 / 20 = 340.60 USD
	// New BRL cost: 34716.0 + (500 * 5.0) = 37216.0 BRL
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "BONUS", 5, 100.0, 5.0, 0.0)
	if !FloatEquals(qty, 20) || !FloatEquals(totalCost, 37216.0) || !FloatEquals(avgPrice, 340.60) {
		t.Errorf("Foreign Bonus failed: got qty=%.2f, cost=%.2f, avg=%.2f (expected qty=20, cost=37216.0, avg=340.60)", qty, totalCost, avgPrice)
	}

	// 6. Complete SELL of all 20 units
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SELL", 20, 700.0, 5.5, 0.0)
	if !FloatEquals(qty, 0) || !FloatEquals(totalCost, 0) || !FloatEquals(avgPrice, 0) {
		t.Errorf("Foreign Complete Sell failed: got qty=%.2f, cost=%.2f, avg=%.2f (expected 0, 0, 0)", qty, totalCost, avgPrice)
	}
}

func TestDomainConstants(t *testing.T) {
	if FinancialEpsilon != 1e-6 {
		t.Errorf("expected FinancialEpsilon to be 1e-6, got %v", FinancialEpsilon)
	}
	if BusinessDaysPerYear != 252.0 {
		t.Errorf("expected BusinessDaysPerYear to be 252.0, got %v", BusinessDaysPerYear)
	}
	if FuzzyMatchGrossAmountThreshold != 0.05 {
		t.Errorf("expected FuzzyMatchGrossAmountThreshold to be 0.05, got %v", FuzzyMatchGrossAmountThreshold)
	}
	if USWithholdingTaxRate != 0.30 || !FloatEquals(USWithholdingNetFactor, 0.70) {
		t.Errorf("expected US tax rates to be 0.30 and 0.70, got %v and %v", USWithholdingTaxRate, USWithholdingNetFactor)
	}
	if B3WithholdingTaxRate != 0.15 || !FloatEquals(B3WithholdingNetFactor, 0.85) {
		t.Errorf("expected B3 tax rates to be 0.15 and 0.85, got %v and %v", B3WithholdingTaxRate, B3WithholdingNetFactor)
	}
}

