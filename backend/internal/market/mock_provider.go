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
	// Retorna cotação fixa para qualquer ticker
	return &Quote{
		Symbol:        symbol,
		Name:          symbol + " Mocked Corp",
		Price:         50.00,
		Change:        1.50,
		ChangePercent: 3.09,
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
