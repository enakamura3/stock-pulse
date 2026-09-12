package telegram

import (
	"context"
	"log/slog"
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
		return h.renderQuote(c, ticker)
	}

	return h.HandleQuoteStart(c)
}

func (h *Handlers) renderQuote(c telebot.Context, ticker string) error {
	quote, err := h.marketSvc.GetQuote(context.Background(), ticker)
	if err != nil {
		msgErr := "⚠️ Ativo não encontrado. Verifique o código e tente novamente."
		if c.Callback() != nil {
			return c.Edit(msgErr)
		}
		return c.Send(msgErr)
	}

	msg := formatQuoteMessage(ticker, quote)

	replyMenu := &telebot.ReplyMarkup{}
	btnRefresh := replyMenu.Data("🔄 Atualizar", "btn_quote_"+ticker)
	btnAnalise := replyMenu.Data("🔍 Análise Fundamentalista", "btn_analise_"+ticker)
	btnNew := replyMenu.Data("🔍 Consultar Outro", "btn_cotacao")
	btnMenuBtn := replyMenu.Data("🏠 Menu", "btn_menu")
	replyMenu.Inline(replyMenu.Row(btnRefresh, btnAnalise), replyMenu.Row(btnNew, btnMenuBtn))

	if c.Callback() != nil {
		err = c.Edit(msg, telebot.ModeMarkdown, replyMenu)
		if err != nil && strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}
	return c.Send(msg, telebot.ModeMarkdown, replyMenu)
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

func (h *Handlers) HandleAnalysis(c telebot.Context) error {
	args := c.Args()
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		ticker := strings.ToUpper(strings.TrimSpace(args[0]))
		return h.renderAnalysis(c, ticker)
	}
	return h.HandleAnalysisStart(c)
}

func (h *Handlers) HandleAnalysisStart(c telebot.Context) error {
	if c.Callback() != nil {
		defer c.Respond()
	}

	err := h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
		Step: "ANALYSIS_EXPECT_TICKER",
	})
	if err != nil {
		if c.Callback() != nil {
			return c.Edit("❌ Erro interno ao iniciar análise fundamentalista.")
		}
		return c.Send("❌ Erro interno ao iniciar análise fundamentalista.")
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	msg := "🔍 *Análise Fundamentalista*\n\nDigite o código do ativo que deseja analisar:\n_(ex: PETR4, VALE3.SA, AAPL)_"
	if c.Callback() != nil {
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) renderAnalysis(c telebot.Context, ticker string) error {
	fund, err := h.marketSvc.GetFundamentals(context.Background(), ticker)
	if err != nil || fund == nil {
		slog.Error("Failed to fetch fundamentals for telegram", "ticker", ticker, "error", err)
		msgErr := "⚠️ Dados fundamentalistas não encontrados para `" + ticker + "`. Verifique o código e tente novamente."
		if c.Callback() != nil {
			return c.Edit(msgErr, telebot.ModeMarkdown)
		}
		return c.Send(msgErr, telebot.ModeMarkdown)
	}

	quote, _ := h.marketSvc.GetQuote(context.Background(), ticker)
	currentPrice := 0.0
	curr := "BRL"
	if quote != nil {
		currentPrice = quote.Price
		if quote.Currency != "" {
			curr = quote.Currency
		}
	}

	msg := formatAnalysisMessage(ticker, fund, currentPrice, curr)

	menu := &telebot.ReplyMarkup{}
	btnRefresh := menu.Data("🔄 Atualizar", "btn_analise_"+ticker)
	btnQuote := menu.Data("📈 Ver Cotação", "btn_quote_"+ticker)
	btnNew := menu.Data("🔍 Outra Análise", "btn_analise")
	btnMenu := menu.Data("🏠 Menu", "btn_menu")
	menu.Inline(menu.Row(btnRefresh, btnQuote), menu.Row(btnNew, btnMenu))

	if c.Callback() != nil {
		err = c.Edit(msg, telebot.ModeMarkdown, menu)
		if err != nil && strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func formatAnalysisMessage(ticker string, fund *market.Fundamentals, price float64, currency string) string {
	p := message.NewPrinter(language.BrazilianPortuguese)
	curr := getCurrencySymbol(currency)

	msg := p.Sprintf("🔍 *Análise Fundamentalista: %s*\n", ticker)
	if price > 1e-6 {
		msg += p.Sprintf("💵 Preço Atual: *%s %.2f*\n\n", curr, price)
	} else {
		msg += "\n"
	}

	msg += "📊 *Múltiplos e Indicadores:*\n"
	if fund.EPS > 1e-6 && price > 1e-6 {
		pe := price / fund.EPS
		msg += p.Sprintf("• P/L: *%.2f*\n", pe)
	} else {
		msg += "• P/L: _N/D_\n"
	}

	if fund.BookValue > 1e-6 && price > 1e-6 {
		pvp := price / fund.BookValue
		msg += p.Sprintf("• P/VP: *%.2f*\n", pvp)
	} else {
		msg += "• P/VP: _N/D_\n"
	}

	msg += p.Sprintf("• Dividend Yield: *%.2f%%*\n", fund.DividendYield)
	if fund.EPS > 1e-6 || fund.EPS < -1e-6 {
		msg += p.Sprintf("• LPA (Lucro/Ação): *%s %.2f*\n", curr, fund.EPS)
	}
	if fund.BookValue > 1e-6 {
		msg += p.Sprintf("• VPA (Patrimônio/Ação): *%s %.2f*\n", curr, fund.BookValue)
	}

	msg += "\n🎯 *Modelos de Valuation:*\n"
	if fund.GrahamValue > 1e-6 {
		grahamStr := p.Sprintf("• Preço Justo de Graham: *%s %.2f*", curr, fund.GrahamValue)
		if price > 1e-6 {
			margin := ((fund.GrahamValue - price) / price) * 100
			if margin > 1e-6 {
				grahamStr += p.Sprintf(" _(🟢 +%.1f%% de margem)_", margin)
			} else if margin < -1e-6 {
				grahamStr += p.Sprintf(" _(🔴 %.1f%% do teto)_", margin)
			}
		}
		msg += grahamStr + "\n"
	} else {
		msg += "• Preço Justo de Graham: _N/D_\n"
	}

	if fund.BazinValue > 1e-6 {
		bazinStr := p.Sprintf("• Preço Teto Bazin (6%%): *%s %.2f*", curr, fund.BazinValue)
		if price > 1e-6 {
			margin := ((fund.BazinValue - price) / price) * 100
			if margin > 1e-6 {
				bazinStr += p.Sprintf(" _(🟢 +%.1f%% de margem)_", margin)
			} else if margin < -1e-6 {
				bazinStr += p.Sprintf(" _(🔴 %.1f%% do teto)_", margin)
			}
		}
		msg += bazinStr + "\n"
	} else {
		msg += "• Preço Teto Bazin (6%): _N/D_\n"
	}

	return msg
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
