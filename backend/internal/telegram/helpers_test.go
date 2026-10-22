package telegram

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "string without special chars",
			input:    "Carteira Principal",
			expected: "Carteira Principal",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with underscores",
			input:    "Minha_Carteira_XP",
			expected: `Minha\_Carteira\_XP`,
		},
		{
			name:     "string with asterisks",
			input:    "Top*Assets*",
			expected: `Top\*Assets\*`,
		},
		{
			name:     "string with backticks and brackets",
			input:    "[CDB] `100% CDI`",
			expected: "\\[CDB\\] \\`100% CDI\\`",
		},
		{
			name:     "string with backslashes",
			input:    `A\B`,
			expected: `A\\B`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFinancialPrice(t *testing.T) {
	p := message.NewPrinter(language.BrazilianPortuguese)

	t.Run("zero value with printer and nil printer", func(t *testing.T) {
		assert.Equal(t, "0,00", formatFinancialPrice(p, 0.0))
		assert.Equal(t, "0,00", formatFinancialPrice(nil, 0.0))
		assert.Equal(t, "0,00", formatFinancialPrice(p, 1e-7))
	})

	t.Run("standard value >= 1.0", func(t *testing.T) {
		assert.Equal(t, "38,50", formatFinancialPrice(p, 38.5))
		assert.Equal(t, "1.234,56", formatFinancialPrice(nil, 1234.56))
		assert.Equal(t, "-15,20", formatFinancialPrice(p, -15.2))
	})

	t.Run("intermediate fractional value 0.01 to 1.0", func(t *testing.T) {
		assert.Equal(t, "0,1234", formatFinancialPrice(p, 0.1234))
		assert.Equal(t, "-0,0567", formatFinancialPrice(p, -0.0567))
		assert.Equal(t, "0,0100", formatFinancialPrice(nil, 0.01))
	})

	t.Run("tiny micro/crypto value < 0.01", func(t *testing.T) {
		assert.Equal(t, "0,000034", formatFinancialPrice(p, 0.000034))
		assert.Equal(t, "0,000034", formatFinancialPrice(nil, 0.000034))
		assert.Equal(t, "-0,005678", formatFinancialPrice(p, -0.005678))
	})
}

func TestIsBlockedByUser(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.False(t, isBlockedByUser(nil))
	})

	t.Run("telebot ErrBlockedByUser", func(t *testing.T) {
		assert.True(t, isBlockedByUser(telebot.ErrBlockedByUser))
	})

	t.Run("string containing blocked by the user", func(t *testing.T) {
		err := errors.New("telegram: Forbidden: bot was blocked by the user")
		assert.True(t, isBlockedByUser(err))
	})

	t.Run("other error", func(t *testing.T) {
		err := errors.New("network timeout")
		assert.False(t, isBlockedByUser(err))
	})
}

func TestGetTickerDefaultCurrency(t *testing.T) {
	tests := []struct {
		ticker   string
		expected string
	}{
		{"PETR4.SA", "R$"},
		{"VALE3.SA", "R$"},
		{"BTC-USD", "US$"},
		{"ETH-USD", "US$"},
		{"SAP-EUR", "€"},
		{"PETR4", "R$"},
		{"HGLG11", "R$"},
		{"AAPL", "US$"},
		{"VOO", "US$"},
		{"123", "R$"},
		{"unknown-ticker-999", "R$"},
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			assert.Equal(t, tt.expected, getTickerDefaultCurrency(tt.ticker))
		})
	}
}

func TestHandlers_SendOrEdit(t *testing.T) {
	h := &Handlers{}

	t.Run("callback is not nil calls Edit success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Edit", "test message", mock.Anything).Return(nil).Once()

		err := h.sendOrEdit(mCtx, "test message")
		assert.NoError(t, err)
	})

	t.Run("callback is not nil calls Edit message not modified returns nil", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Edit", "test message", mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err := h.sendOrEdit(mCtx, "test message")
		assert.NoError(t, err)
	})

	t.Run("callback is not nil calls Edit generic error returns error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Edit", "test message", mock.Anything).Return(errors.New("network fail")).Once()

		err := h.sendOrEdit(mCtx, "test message")
		assert.EqualError(t, err, "network fail")
	})

	t.Run("callback is nil calls Send", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Send", "test message", mock.Anything).Return(nil).Once()

		err := h.sendOrEdit(mCtx, "test message")
		assert.NoError(t, err)
	})
}

func TestIsB3TickerAndNormalize(t *testing.T) {
	assert.True(t, isB3Ticker("PETR4"))
	assert.True(t, isB3Ticker("vale3"))
	assert.True(t, isB3Ticker("HGLG11"))
	assert.True(t, isB3Ticker("PETR4.SA"))
	assert.True(t, isB3Ticker("PETR4F"))
	assert.True(t, isB3Ticker("petr4f.sa"))

	assert.False(t, isB3Ticker("AAPL"))
	assert.False(t, isB3Ticker("BTC-USD"))
	assert.False(t, isB3Ticker("P1"))
	assert.False(t, isB3Ticker("TOOLONG11"))
	assert.False(t, isB3Ticker("12345"))
	assert.False(t, isB3Ticker("PETRA"))
	assert.False(t, isB3Ticker("PETR4X"))

	assert.Equal(t, "PETR4.SA", normalizeB3Ticker("petr4"))
	assert.Equal(t, "VALE3.SA", normalizeB3Ticker("VALE3.SA"))
	assert.Equal(t, "AAPL", normalizeB3Ticker("AAPL"))
	assert.Equal(t, "BTC-USD", normalizeB3Ticker("BTC-USD"))
}
