package telegram

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func TestHandlers_DividendHelpers(t *testing.T) {
	t.Run("getAssetTypeEmoji", func(t *testing.T) {
		assert.Equal(t, "🏢", getAssetTypeEmoji("FII", "HGLG11.SA"))
		assert.Equal(t, "🏢", getAssetTypeEmoji("FIAGRO", "KNCA11.SA"))
		assert.Equal(t, "📊", getAssetTypeEmoji("ETF_BR", "BOVA11.SA"))
		assert.Equal(t, "📊", getAssetTypeEmoji("ETF", "IVVB11.SA"))
		assert.Equal(t, "🌐", getAssetTypeEmoji("BDR", "AAPL34.SA"))
		assert.Equal(t, "🪙", getAssetTypeEmoji("CRYPTO", "BTC.SA"))
		assert.Equal(t, "🏢", getAssetTypeEmoji("UNKNOWN", "XPIN11.SA"))
		assert.Equal(t, "📈", getAssetTypeEmoji("STOCK", "PETR4.SA"))
		assert.Equal(t, "🪙", getAssetTypeEmoji("CRYPTO", "BTC-USD"))
		assert.Equal(t, "🇺🇸", getAssetTypeEmoji("STOCK", "AAPL"))
	})

	t.Run("cleanTickerForDisplay", func(t *testing.T) {
		assert.Equal(t, "PETR4", cleanTickerForDisplay("petr4.sa"))
		assert.Equal(t, "AAPL", cleanTickerForDisplay("AAPL"))
	})

	t.Run("formatQuantity", func(t *testing.T) {
		assert.Equal(t, "100", formatQuantity(100.0))
		assert.Equal(t, "100.50", formatQuantity(100.50))
	})

	t.Run("formatPerShareAmount", func(t *testing.T) {
		p := message.NewPrinter(language.BrazilianPortuguese)
		assert.Equal(t, "1,50", formatPerShareAmount(p, 1.5000))
		assert.Equal(t, "1,235", formatPerShareAmount(p, 1.2350))
		assert.Equal(t, "1,2345", formatPerShareAmount(p, 1.2345))
	})

	t.Run("getMonthNamePT", func(t *testing.T) {
		assert.Equal(t, "Janeiro", getMonthNamePT(time.January))
		assert.Equal(t, "Dezembro", getMonthNamePT(time.December))
	})
}
