package market

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	t.Run("GetQuote", func(t *testing.T) {
		quote, err := provider.GetQuote(ctx, "PETR4.SA")
		assert.NoError(t, err)
		assert.NotNil(t, quote)
		assert.Equal(t, "PETR4.SA", quote.Symbol)
		assert.Equal(t, "PETR4.SA Mocked Corp", quote.Name)
		assert.Equal(t, 50.00, quote.Price)
	})

	t.Run("SearchAssets", func(t *testing.T) {
		results, err := provider.SearchAssets(ctx, "PETR")
		assert.NoError(t, err)
		assert.NotEmpty(t, results)
		assert.Equal(t, "PETR4.SA", results[0].Symbol)
	})
}
