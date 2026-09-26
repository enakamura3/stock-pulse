package telegram

import (
	"math"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

// escapeMarkdown escapa caracteres especiais reservados do modo Markdown legado do Telegram (*, _, `, [, \).
func escapeMarkdown(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '_', '*', '`', '[', ']', '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatFinancialPrice formata um valor monetário respeitando a precisão necessária para ativos padrão e fracionários/criptoativos.
// Valores >= 1.0 utilizam 2 casas decimais.
// Valores entre 0.01 e 1.0 utilizam 4 casas decimais.
// Valores menores que 0.01 utilizam até 6 casas decimais.
func formatFinancialPrice(p *message.Printer, val float64) string {
	absVal := math.Abs(val)
	if absVal < 1e-6 {
		if p != nil {
			return p.Sprintf("%.2f", 0.0)
		}
		return "0,00"
	}

	printer := p
	if printer == nil {
		printer = message.NewPrinter(language.BrazilianPortuguese)
	}

	if absVal >= 1.0 {
		return printer.Sprintf("%.2f", val)
	}
	if absVal >= 0.01 {
		return printer.Sprintf("%.4f", val)
	}
	return printer.Sprintf("%.6f", val)
}

// isBlockedByUser detecta se o erro retornado pela API do Telegram indica que o usuário bloqueou o bot.
func isBlockedByUser(err error) bool {
	if err == nil {
		return false
	}
	if telebot.ErrIs(err.Error(), telebot.ErrBlockedByUser) || strings.Contains(strings.ToLower(err.Error()), "blocked by the user") {
		return true
	}
	return false
}
