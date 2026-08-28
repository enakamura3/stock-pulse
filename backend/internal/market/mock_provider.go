package market

import (
	"context"
)

// MockProvider implementa a interface QuoteProvider retornando dados constantes
// para fins de testes (especialmente testes E2E), garantindo independência de APIs externas.
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	switch symbol {
	case "^BVSP":
		return &Quote{
			Symbol:        "^BVSP",
			Name:          "Ibovespa",
			Price:         130000.00,
			Change:        1300.00,
			ChangePercent: 1.01,
			PreviousClose: 128700.00,
			High:          130500.00,
			Low:           128600.00,
			Volume:        10000000,
			Currency:      "BRL",
		}, nil
	case "^GSPC":
		return &Quote{
			Symbol:        "^GSPC",
			Name:          "S&P 500",
			Price:         5500.00,
			Change:        25.00,
			ChangePercent: 0.45,
			PreviousClose: 5475.00,
			High:          5510.00,
			Low:           5460.00,
			Volume:        20000000,
			Currency:      "USD",
		}, nil
	case "BRL=X":
		return &Quote{
			Symbol:        "BRL=X",
			Name:          "Dólar Comercial",
			Price:         5.45,
			Change:        -0.02,
			ChangePercent: -0.37,
			PreviousClose: 5.47,
			High:          5.48,
			Low:           5.44,
			Volume:        0,
			Currency:      "BRL",
		}, nil
	case "IFIX.SA":
		return &Quote{
			Symbol:        "IFIX.SA",
			Name:          "IFIX",
			Price:         3350.00,
			Change:        5.00,
			ChangePercent: 0.15,
			PreviousClose: 3345.00,
			High:          3355.00,
			Low:           3340.00,
			Volume:        500000,
			Currency:      "BRL",
		}, nil
	}

	// Retorna cotação fixa para qualquer ticker
	return &Quote{
		Symbol:        symbol,
		Name:          symbol + " Mocked Corp",
		Price:         50.00,
		Change:        1.50,
		ChangePercent: 3.09,
		PreviousClose: 48.50,
		High:          51.00,
		Low:           49.50,
		Volume:        1000000,
		Currency:      "BRL",
	}, nil
}

func (m *MockProvider) SearchAssets(ctx context.Context, query string) ([]SearchResult, error) {
	// Retorna resultado genérico
	return []SearchResult{
		{
			Symbol:   "PETR4.SA",
			Name:     "Petróleo Brasileiro S.A. - Petrobras",
			Exchange: "SAO",
			Type:     "EQUITY",
		},
	}, nil
}

func (m *MockProvider) GetHistoricalPrices(ctx context.Context, symbol string, rangePeriod string) ([]HistoricalPrice, error) {
	return []HistoricalPrice{}, nil
}

func (m *MockProvider) GetHistoricalPricesBetween(ctx context.Context, symbol string, period1, period2 int64) ([]HistoricalPrice, error) {
	return []HistoricalPrice{}, nil
}

