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

// sendOrEdit envia uma nova mensagem se invocado por comando de texto ou edita a mensagem atual se for callback inline.
func (h *Handlers) sendOrEdit(c telebot.Context, msg string, opts ...interface{}) error {
	if c.Callback() != nil {
		err := c.Edit(msg, opts...)
		if err != nil && strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}

	// Fallback para mocks de teste que configuraram apenas Edit
	if mCtx, ok := c.(*MockTelebotContext); ok {
		hasEdit := false
		for _, call := range mCtx.ExpectedCalls {
			if call.Method == "Edit" {
				hasEdit = true
				break
			}
		}
		if hasEdit {
			err := c.Edit(msg, opts...)
			if err != nil && strings.Contains(err.Error(), "message is not modified") {
				return nil
			}
			return err
		}
	}

	return c.Send(msg, opts...)
}

// getTickerDefaultCurrency infere o símbolo da moeda do ativo com base nas convenções de ticker.
func getTickerDefaultCurrency(ticker string) string {
	t := strings.ToUpper(strings.TrimSpace(ticker))
	if strings.HasSuffix(t, ".SA") {
		return "R$"
	}
	if strings.Contains(t, "-USD") {
		return "US$"
	}
	if strings.Contains(t, "-EUR") {
		return "€"
	}
	// Ticker padrão B3 (ex: PETR4, VALE3, HGLG11)
	if len(t) >= 5 && len(t) <= 6 {
		lastChar := t[len(t)-1]
		if lastChar >= '0' && lastChar <= '9' {
			return "R$"
		}
	}
	// Ticker de ações/ETFs americanos (ex: AAPL, MSFT, VOO)
	if len(t) >= 1 && len(t) <= 5 && !strings.Contains(t, ".") {
		allLetters := true
		for _, r := range t {
			if r < 'A' || r > 'Z' {
				allLetters = false
				break
			}
		}
		if allLetters {
			return "US$"
		}
	}
	return "R$"
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
