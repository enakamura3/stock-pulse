package market

import (
	"context"
	"time"
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

func (m *MockProvider) GetDividends(ctx context.Context, symbol string, assetType string) ([]DividendEvent, error) {
	// Retorna proventos fixos para facilitar testes de performance do portfólio
	return []DividendEvent{
		{
			Date:        time.Now().Add(-30 * 24 * time.Hour),
			PaymentDate: time.Now().Add(-15 * 24 * time.Hour),
			Amount:      1.50,
			Type:        "Dividendo",
		},
		{
			Date:        time.Now().Add(-90 * 24 * time.Hour),
			PaymentDate: time.Now().Add(-80 * 24 * time.Hour),
			Amount:      2.00,
			Type:        "JCP",
		},
	}, nil
}
