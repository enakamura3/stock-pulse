package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleAlerts(c telebot.Context) error {
	defer c.Respond()
	if h.alertSvc == nil {
		return c.Edit("⚠️ Módulo de alertas não está ativo.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	alerts, err := h.alertSvc.GetAlerts(context.Background(), userIDStr)
	if err != nil {
		slog.Error("Failed to fetch alerts for telegram bot", "error", err)
		return c.Edit("❌ Ocorreu um erro ao buscar seus alertas.")
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	msg := "🔔 *Meus Alertas de Preço*\n\n"

	if len(alerts) == 0 {
		msg += "Você não possui alertas cadastrados.\n\nUse o botão abaixo para criar o seu primeiro alerta!"
	} else {
		for _, a := range alerts {
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
				emoji, a.Ticker, condStr, getCurrencySymbol(curr), a.TargetPrice, statusLabel)
		}
		msg += p.Sprintf("\n_Total: %d alerta(s)_", len(alerts))
	}

	menu := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	// Botões de ação por alerta (máximo 5 para não sobrecarregar o inline keyboard)
	limit := 5
	if len(alerts) < limit {
		limit = len(alerts)
	}
	for i := 0; i < limit; i++ {
		a := alerts[i]
		toggleLabel := "⏸️ Pausar " + a.Ticker
		if a.Status == "DISABLED" || a.Status == "TRIGGERED" {
			toggleLabel = "▶️ Ativar " + a.Ticker
		}
		btnToggle := menu.Data(toggleLabel, "btn_alert_toggle_"+a.ID)
		btnDel := menu.Data("🗑️ "+a.Ticker, "btn_alert_del_"+a.ID)
		rows = append(rows, menu.Row(btnToggle, btnDel))
	}

	btnCreate := menu.Data("➕ Criar Alerta", "btn_alert_create")
	btnRefresh := menu.Data("🔄 Atualizar", "btn_alerts")
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnCreate, btnRefresh), menu.Row(btnBack))
	menu.Inline(rows...)

	err = c.Edit(msg, telebot.ModeMarkdown, menu)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}

func (h *Handlers) HandleAlertCreate(c telebot.Context) error {
	defer c.Respond()

	err := h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
		Step: "ALERT_EXPECT_TICKER",
	})
	if err != nil {
		return c.Edit("❌ Erro interno ao iniciar criação de alerta.")
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	return c.Edit("🔔 *Criar Alerta de Preço*\n\nDigite o código do ativo:\n_(ex: PETR4, VALE3.SA, AAPL, BTC-USD)_", telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleAlertConditionAbove(c telebot.Context) error {
	return h.handleAlertCondition(c, "ABOVE")
}

func (h *Handlers) HandleAlertConditionBelow(c telebot.Context) error {
	return h.handleAlertCondition(c, "BELOW")
}

func (h *Handlers) handleAlertCondition(c telebot.Context, condition string) error {
	defer c.Respond()

	state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
	if err != nil || state == nil || state.Step != "ALERT_EXPECT_COND" {
		return c.Edit("⚠️ Nenhuma criação de alerta em andamento.")
	}

	state.Type = condition
	state.Step = "ALERT_EXPECT_PRICE"
	_ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

	condLabel := "ACIMA DE"
	if condition == "BELOW" {
		condLabel = "ABAIXO DE"
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	return c.Edit(fmt.Sprintf("🔔 *Alerta para %s* (%s)\n\nQual o preço alvo do alerta? (ex: 35.50)", state.Ticker, condLabel), telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleAlertToggle(c telebot.Context, alertID string) error {
	defer c.Respond()

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	_, err = h.alertSvc.ToggleAlert(context.Background(), alertID, userIDStr)
	if err != nil {
		slog.Error("Failed to toggle alert status", "error", err, "alert_id", alertID)
	}

	return h.HandleAlerts(c)
}

func (h *Handlers) handleAlertDelete(c telebot.Context, alertID string) error {
	defer c.Respond()

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	err = h.alertSvc.DeleteAlert(context.Background(), alertID, userIDStr)
	if err != nil {
		slog.Error("Failed to delete alert", "error", err, "alert_id", alertID)
	}

	return h.HandleAlerts(c)
}
