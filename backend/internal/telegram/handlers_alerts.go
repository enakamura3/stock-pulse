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
	page := 0
	if rawData := c.Data(); rawData != "" {
		fmt.Sscanf(rawData, "%d", &page)
	}
	return h.renderAlerts(c, page)
}

func (h *Handlers) renderAlerts(c telebot.Context, page int) error {
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
				emoji, a.Ticker, condStr, getCurrencySymbol(curr), a.TargetPrice, statusLabel)
		}
		msg += p.Sprintf("\n_Total: %d alerta(s)_", len(alerts))

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row

		for i := start; i < end; i++ {
			a := alerts[i]
			toggleLabel := "⏸️ Pausar " + a.Ticker
			if a.Status == "DISABLED" || a.Status == "TRIGGERED" {
				toggleLabel = "▶️ Ativar " + a.Ticker
			}
			btnToggle := menu.Data(toggleLabel, fmt.Sprintf("btn_alert_toggle_%s:%d", a.ID, page))
			btnDel := menu.Data("🗑️ "+a.Ticker, fmt.Sprintf("btn_alert_del_%s:%d", a.ID, page))
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

		err = c.Edit(msg, telebot.ModeMarkdown, menu)
		if err != nil && strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}

	menu := &telebot.ReplyMarkup{}
	btnCreate := menu.Data("➕ Criar Alerta", "btn_alert_create")
	btnRefresh := menu.Data("🔄 Atualizar", "btn_alerts")
	btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
	menu.Inline(menu.Row(btnCreate, btnRefresh), menu.Row(btnBack))

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

	_, err = h.alertSvc.ToggleAlert(context.Background(), alertID, userIDStr)
	if err != nil {
		slog.Error("Failed to toggle alert status", "error", err, "alert_id", alertID)
	}

	return h.renderAlerts(c, page)
}

func (h *Handlers) handleAlertDelete(c telebot.Context, payload string) error {
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

	err = h.alertSvc.DeleteAlert(context.Background(), alertID, userIDStr)
	if err != nil {
		slog.Error("Failed to delete alert", "error", err, "alert_id", alertID)
	}

	return h.renderAlerts(c, page)
}
