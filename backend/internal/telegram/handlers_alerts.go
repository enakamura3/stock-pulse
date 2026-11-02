package telegram

import (
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleAlerts(c telebot.Context) error {
	defer c.Respond()
	if c.Chat() != nil {
		ctx, cancel := h.getContext(c)
		defer cancel()
		_ = h.svc.ClearConversationState(ctx, c.Chat().ID)
	}
	page := 0
	if rawData := c.Data(); rawData != "" {
		fmt.Sscanf(rawData, "%d", &page)
	}
	return h.renderAlerts(c, page)
}

func (h *Handlers) renderAlerts(c telebot.Context, page int) error {
	if h.alertSvc == nil {
		return h.sendOrEdit(c, "⚠️ Módulo de alertas não está ativo.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	alerts, err := h.alertSvc.GetAlerts(ctx, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch alerts for telegram bot", "error", err)
		return h.sendOrEdit(c, "❌ Ocorreu um erro ao buscar seus alertas.")
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := "🔔 *Meus Alertas de Preço*\n"

	if len(alerts) == 0 {
		msg += "\nVocê não possui alertas cadastrados.\n\nUse o botão abaixo para criar o seu primeiro alerta!"
	} else {
		pageSize := 5
		totalPages := (len(alerts) + pageSize - 1) / pageSize
		if page < 0 {
			page = 0
		}
		if page >= totalPages {
			page = totalPages - 1
		}
		start := page * pageSize
		end := start + pageSize
		if end > len(alerts) {
			end = len(alerts)
		}

		if totalPages > 1 {
			msg += p.Sprintf("_Página %d de %d_\n\n", page+1, totalPages)
		} else {
			msg += "\n"
		}

		for _, a := range alerts[start:end] {
			condStr := "acima de"
			if a.Condition == "BELOW" {
				condStr = "abaixo de"
			}
			curr := a.Currency
			if curr == "" {
				curr = "BRL"
			}

			var emoji, statusLabel string
			switch a.Status {
			case "ACTIVE":
				emoji = "🟢"
			case "TRIGGERED":
				emoji = "🔔"
				statusLabel = " _(disparado)_"
			case "DISABLED":
				emoji = "⚪"
				statusLabel = " _(pausado)_"
			default:
				emoji = "⚪"
			}

			msg += p.Sprintf("%s `%s` — %s %s %.2f%s\n",
				emoji, cleanTickerForDisplay(a.Ticker), condStr, getCurrencySymbol(curr), a.TargetPrice, statusLabel)
		}
		msg += p.Sprintf("\n_Total: %d alerta(s)_", len(alerts))

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row

		for i := start; i < end; i++ {
			a := alerts[i]
			tickerClean := cleanTickerForDisplay(a.Ticker)
			toggleLabel := "⏸️ Pausar " + tickerClean
			if a.Status == "DISABLED" || a.Status == "TRIGGERED" {
				toggleLabel = "▶️ Ativar " + tickerClean
			}
			btnToggle := menu.Data(toggleLabel, fmt.Sprintf("btn_alert_toggle_%s:%d", a.ID, page))
			btnDel := menu.Data("🗑️ "+tickerClean, fmt.Sprintf("btn_alert_del_%s:%d", a.ID, page))
			rows = append(rows, menu.Row(btnToggle, btnDel))
		}

		var navBtns []telebot.Btn
		if page > 0 {
			navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_alerts", fmt.Sprintf("%d", page-1)))
		}
		if end < len(alerts) {
			navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_alerts", fmt.Sprintf("%d", page+1)))
		}
		if len(navBtns) > 0 {
			rows = append(rows, menu.Row(navBtns...))
		}

		btnCreate := menu.Data("➕ Criar Alerta", "btn_alert_create")
		btnRefresh := menu.Data("🔄 Atualizar", "btn_alerts", fmt.Sprintf("%d", page))
		btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
		rows = append(rows, menu.Row(btnCreate, btnRefresh), menu.Row(btnBack))
		menu.Inline(rows...)

		return h.sendOrEdit(c, msg, telebot.ModeMarkdown, menu)
	}

	menu := &telebot.ReplyMarkup{}
	btnCreate := menu.Data("➕ Criar Alerta", "btn_alert_create")
	btnRefresh := menu.Data("🔄 Atualizar", "btn_alerts")
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	menu.Inline(menu.Row(btnCreate, btnRefresh), menu.Row(btnBack))

	return h.sendOrEdit(c, msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleAlertCreate(c telebot.Context) error {
	defer c.Respond()

	ctx, cancel := h.getContext(c)
	defer cancel()

	err := h.svc.SetConversationState(ctx, c.Chat().ID, ConversationState{
		Step: "ALERT_EXPECT_TICKER",
	})
	if err != nil {
		return c.Edit("❌ Erro interno ao iniciar criação de alerta.")
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	return c.Edit("🔔 *Criar Alerta de Preço*\n\nDigite o código do ativo:\n_(ex: PETR4, VALE3, AAPL, BTC-USD)_", telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleAlertConditionAbove(c telebot.Context) error {
	return h.handleAlertCondition(c, "ABOVE")
}

func (h *Handlers) HandleAlertConditionBelow(c telebot.Context) error {
	return h.handleAlertCondition(c, "BELOW")
}

func (h *Handlers) handleAlertCondition(c telebot.Context, condition string) error {
	defer c.Respond()

	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "ALERT_EXPECT_COND" {
		return c.Edit("⚠️ Nenhuma criação de alerta em andamento.")
	}

	state.Type = condition
	state.Step = "ALERT_EXPECT_PRICE"
	_ = h.svc.SetConversationState(ctx, c.Chat().ID, *state)

	condLabel := "ACIMA DE"
	if condition == "BELOW" {
		condLabel = "ABAIXO DE"
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	currSymbol := getTickerDefaultCurrency(state.Ticker)
	msg := fmt.Sprintf("🔔 *Alerta para %s* (%s)\n", escapeMarkdown(cleanTickerForDisplay(state.Ticker)), condLabel)

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	if h.marketSvc != nil {
		if quote, errQ := h.getQuoteWithFallback(ctx, state.Ticker); errQ == nil && quote != nil && quote.Price > 1e-6 {
			if quote.Currency != "" {
				currSymbol = getCurrencySymbol(quote.Currency)
			}
			msg += p.Sprintf("💵 Preço Atual: *%s %s*\n", currSymbol, formatFinancialPrice(p, quote.Price))

			if condition == "BELOW" {
				p5 := quote.Price * 0.95
				p10 := quote.Price * 0.90
				btn5 := menu.Data(p.Sprintf("📉 -5%% (%s %.2f)", currSymbol, p5), fmt.Sprintf("btn_alert_price_%.2f", p5))
				btn10 := menu.Data(p.Sprintf("📉 -10%% (%s %.2f)", currSymbol, p10), fmt.Sprintf("btn_alert_price_%.2f", p10))
				rows = append(rows, menu.Row(btn5, btn10))
			} else {
				p5 := quote.Price * 1.05
				p10 := quote.Price * 1.10
				btn5 := menu.Data(p.Sprintf("📈 +5%% (%s %.2f)", currSymbol, p5), fmt.Sprintf("btn_alert_price_%.2f", p5))
				btn10 := menu.Data(p.Sprintf("📈 +10%% (%s %.2f)", currSymbol, p10), fmt.Sprintf("btn_alert_price_%.2f", p10))
				rows = append(rows, menu.Row(btn5, btn10))
			}
		}
	}

	if pos := h.findActiveCustodyPosition(ctx, c, state.Ticker); pos != nil && pos.AveragePrice > 1e-6 {
		msg += p.Sprintf("💼 Seu PM: *%s %s*\n", currSymbol, formatFinancialPrice(p, pos.AveragePrice))
		btnPM := menu.Data(p.Sprintf("🎯 Meu PM (%s %.2f)", currSymbol, pos.AveragePrice), fmt.Sprintf("btn_alert_price_%.2f", pos.AveragePrice))
		rows = append(rows, menu.Row(btnPM))
	}

	msg += "\nQual o preço alvo do alerta? (ex: 35.50)"

	btnBack := menu.Data("⬅️ Voltar", "btn_op_back")
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	rows = append(rows, menu.Row(btnBack, btnCancel))
	menu.Inline(rows...)

	return c.Edit(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleSelectedAlertPrice(c telebot.Context, priceStr string) error {
	defer c.Respond()

	ctx, cancel := h.getContext(c)
	defer cancel()

	state, err := h.svc.GetConversationState(ctx, c.Chat().ID)
	if err != nil || state == nil || state.Step != "ALERT_EXPECT_PRICE" {
		return c.Edit("⚠️ Nenhuma criação de alerta em andamento.")
	}

	var price float64
	if _, err := fmt.Sscanf(priceStr, "%f", &price); err != nil || price <= 1e-6 {
		return c.Edit("⚠️ Preço inválido.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	createdAlert, err := h.alertSvc.CreateAlert(ctx, userIDStr, state.Ticker, price, state.Type)
	if err != nil {
		slog.Error("Erro ao criar alerta via callback telegram", "error", err)
		return c.Edit("❌ Ocorreu um erro ao salvar o alerta. Tente novamente mais tarde.")
	}

	_ = h.svc.ClearConversationState(ctx, c.Chat().ID)

	condStr := "acima de"
	if createdAlert.Condition == "BELOW" {
		condStr = "abaixo de"
	}
	curr := createdAlert.Currency
	if curr == "" {
		curr = "BRL"
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	successMsg := p.Sprintf("✅ *Alerta Criado com Sucesso!*\n\n• Ativo: `%s`\n• Condição: Quando estiver %s %s %.2f",
		cleanTickerForDisplay(createdAlert.Ticker), condStr, getCurrencySymbol(curr), createdAlert.TargetPrice)

	successMenu := &telebot.ReplyMarkup{}
	btnAlertsList := successMenu.Data("🔔 Meus Alertas", "btn_alerts")
	btnMenuBack := successMenu.Data("🏠 Menu", "btn_menu")
	successMenu.Inline(successMenu.Row(btnAlertsList, btnMenuBack))

	return c.Edit(successMsg, telebot.ModeMarkdown, successMenu)
}

func (h *Handlers) handleAlertToggle(c telebot.Context, payload string) error {
	defer c.Respond()

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	alertID := payload
	page := 0
	if parts := strings.SplitN(payload, ":", 2); len(parts) == 2 {
		alertID = parts[0]
		fmt.Sscanf(parts[1], "%d", &page)
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	_, err = h.alertSvc.ToggleAlert(ctx, alertID, userIDStr)
	if err != nil {
		slog.Error("Failed to toggle alert status", "error", err, "alert_id", alertID)
	}

	return h.renderAlerts(c, page)
}

func (h *Handlers) handleAlertDelete(c telebot.Context, payload string) error {
	defer func() {
		_ = c.Respond(&telebot.CallbackResponse{Text: "🗑️ Alerta removido!"})
	}()

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	alertID := payload
	page := 0
	if parts := strings.SplitN(payload, ":", 2); len(parts) == 2 {
		alertID = parts[0]
		fmt.Sscanf(parts[1], "%d", &page)
	}

	ctx, cancel := h.getContext(c)
	defer cancel()

	err = h.alertSvc.DeleteAlert(ctx, alertID, userIDStr)
	if err != nil {
		slog.Error("Failed to delete alert", "error", err, "alert_id", alertID)
	}

	return h.renderAlerts(c, page)
}
