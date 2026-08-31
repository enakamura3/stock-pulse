package telegram

import (
	"context"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleQuote(c telebot.Context) error {
	args := c.Args()
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		ticker := strings.ToUpper(strings.TrimSpace(args[0]))
		quote, err := h.marketSvc.GetQuote(context.Background(), ticker)
		if err != nil {
			return c.Send("⚠️ Ativo não encontrado. Verifique o código e tente novamente.")
		}

		msg := formatQuoteMessage(ticker, quote)

		replyMenu := &telebot.ReplyMarkup{}
		btnNew := replyMenu.Data("🔍 Consultar Outro", "btn_cotacao")
		btnMenuBtn := replyMenu.Data("🏠 Menu", "btn_menu")
		replyMenu.Inline(replyMenu.Row(btnNew, btnMenuBtn))

		return c.Send(msg, telebot.ModeMarkdown, replyMenu)
	}

	return h.HandleQuoteStart(c)
}

func (h *Handlers) HandleQuoteStart(c telebot.Context) error {
	defer c.Respond()

	err := h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
		Step: "QUOTE_EXPECT_TICKER",
	})
	if err != nil {
		if c.Callback() != nil {
			return c.Edit("❌ Erro interno ao iniciar consulta de cotação.")
		}
		return c.Send("❌ Erro interno ao iniciar consulta de cotação.")
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	msg := "📈 *Cotação Rápida*\n\nDigite o código do ativo:\n_(ex: VALE3.SA, AAPL, BTC-USD)_"
	if c.Callback() != nil {
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func formatQuoteMessage(ticker string, quote *market.Quote) string {
	changeEmoji := "⚪"
	changeSign := ""
	if quote.Change > 1e-6 {
		changeEmoji = "🟢"
		changeSign = "+"
	} else if quote.Change < -1e-6 {
		changeEmoji = "🔴"
	}

	curr := getCurrencySymbol(quote.Currency)
	title := quote.Symbol
	if title == "" {
		title = ticker
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := p.Sprintf("📈 *%s*\n_%s_\n\n", title, quote.Name)
	msg += p.Sprintf("💵 *Preço:* %s %.2f\n", curr, quote.Price)
	msg += p.Sprintf("%s *Variação:* %s%.2f (%s%.2f%%)\n",
		changeEmoji, changeSign, quote.Change, changeSign, quote.ChangePercent)

	if quote.High > 1e-6 || quote.Low > 1e-6 {
		msg += p.Sprintf("📊 *Mín / Máx (Dia):* %s %.2f / %s %.2f\n", curr, quote.Low, curr, quote.High)
	}
	if quote.PreviousClose > 1e-6 {
		msg += p.Sprintf("⏮️ *Fechamento Anterior:* %s %.2f\n", curr, quote.PreviousClose)
	}
	if quote.Volume > 0 {
		msg += p.Sprintf("📦 *Volume:* %d\n", quote.Volume)
	}
	return msg
}
