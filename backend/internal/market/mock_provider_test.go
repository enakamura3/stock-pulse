package market

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	t.Run("GetQuote - default", func(t *testing.T) {
		quote, err := provider.GetQuote(ctx, "PETR4.SA")
		assert.NoError(t, err)
		assert.NotNil(t, quote)
		assert.Equal(t, "PETR4.SA", quote.Symbol)
		assert.Equal(t, "PETR4.SA Mocked Corp", quote.Name)
		assert.Equal(t, 50.00, quote.Price)
	})

	t.Run("GetQuote - special tickers", func(t *testing.T) {
		tickers := []string{"^BVSP", "^GSPC", "BRL=X", "IFIX.SA"}
		for _, sym := range tickers {
			q, err := provider.GetQuote(ctx, sym)
			assert.NoError(t, err)
			assert.NotNil(t, q)
			assert.Equal(t, sym, q.Symbol)
		}
	})

	t.Run("SearchAssets", func(t *testing.T) {
		results, err := provider.SearchAssets(ctx, "PETR")
		assert.NoError(t, err)
		assert.NotEmpty(t, results)
		assert.Equal(t, "PETR4.SA", results[0].Symbol)
	})

	t.Run("HistoricalPrices", func(t *testing.T) {
		hist, err := provider.GetHistoricalPrices(ctx, "PETR4.SA", "1mo")
		assert.NoError(t, err)
		assert.Empty(t, hist)

		histBetween, err := provider.GetHistoricalPricesBetween(ctx, "PETR4.SA", 0, 100)
		assert.NoError(t, err)
		assert.Empty(t, histBetween)
	})
}
