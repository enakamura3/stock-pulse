package telegram

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
