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
		{"PETR4.SA", "Petroleo Brasileiro SA", "BRL", "STOCK_BR"},
		{"VALE3.SA", "Vale SA", "BRL", "STOCK_BR"},
		{"HGLG11.SA", "CSHG Logistica FII", "BRL", "FII"},
		{"MXRF11.SA", "Maxi Renda FII", "BRL", "FII"},
		{"IVVB11.SA", "iShares S&P 500 Fundo de Indice", "BRL", "ETF_BR"},
		{"AAPL34.SA", "Apple Inc BDR", "BRL", "BDR"},
	}

	for _, tt := range tests {
		got := DetermineAssetType(tt.ticker, tt.name, tt.currency)
		if got != tt.expected {
			t.Errorf("DetermineAssetType(%s, %s) = %s, want %s", tt.ticker, tt.name, got, tt.expected)
		}
	}
}

func TestUpdatePositionOnTransaction(t *testing.T) {
	// 1. Initial Buy
	qty, totalCost, avgPrice := UpdatePositionOnTransaction(0, 0, 0, "BUY", 100, 30.0, 1.0)
	if qty != 100 || totalCost != 3000.0 || avgPrice != 30.0 {
		t.Errorf("Buy 1 failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 2. Second Buy at higher price
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "BUY", 100, 40.0, 1.0)
	if qty != 200 || totalCost != 7000.0 || avgPrice != 35.0 {
		t.Errorf("Buy 2 failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 3. Partial Sell (Average price should remain 35.0)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SELL", 50, 45.0, 1.0)
	if qty != 150 || !FloatEquals(totalCost, 5250.0) || !FloatEquals(avgPrice, 35.0) {
		t.Errorf("Sell failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 4. Split 2:1 (Quantity doubles, avg price halves)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SPLIT", 2.0, 0, 1.0)
	if qty != 300 || !FloatEquals(totalCost, 5250.0) || !FloatEquals(avgPrice, 17.5) {
		t.Errorf("Split failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 5. Reverse Split 10:1 (300 -> 30)
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "REVERSE_SPLIT", 10.0, 0, 1.0)
	if qty != 30 || !FloatEquals(totalCost, 5250.0) || !FloatEquals(avgPrice, 175.0) {
		t.Errorf("Reverse Split failed: got qty=%.2f, cost=%.2f, avg=%.2f", qty, totalCost, avgPrice)
	}

	// 6. Over-Sell zapping position to zero
	qty, totalCost, avgPrice = UpdatePositionOnTransaction(qty, totalCost, avgPrice, "SELL", 500, 200.0, 1.0)
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
	// +10% then -5% -> TWRR = 1.10 * 0.95 - 1 = 1.045 - 1 = 4.5%
	twrr := CalculateTWRR([]float64{0.10, -0.05})
	if !FloatEquals(twrr, 4.5) {
		t.Errorf("TWRR failed: got %.4f, want 4.5000", twrr)
	}
}
