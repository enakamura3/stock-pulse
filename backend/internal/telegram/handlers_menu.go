package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"gopkg.in/telebot.v3"
)

func (h *Handlers) HandleStart(c telebot.Context) error {
	args := c.Args()
	if len(args) > 0 {
		token := args[0]
		err := h.svc.LinkAccountWithToken(context.Background(), token, c.Chat().ID)
		if err != nil {
			if strings.Contains(err.Error(), "inválido ou expirado") {
				return c.Send("❌ O link de vinculação é inválido ou expirou. Gere um novo no Stock Pulse.")
			}
			slog.Error("Erro ao vincular conta telegram", "error", err, "chat_id", c.Chat().ID)
			return c.Send("❌ Ocorreu um erro interno ao vincular sua conta. Tente novamente.")
		}
		return c.Send("✅ Conta vinculada com sucesso! Bem-vindo ao Stock Pulse.\n\nEnvie /menu para ver as opções.")
	}

	return c.Send("Bem-vindo ao bot do Stock Pulse! Para usar este bot, vá até as configurações no sistema web e clique em 'Vincular Telegram'.")
}

func (h *Handlers) HandleMenu(c telebot.Context) error {
	return h.sendOrEditMenu(c)
}

func (h *Handlers) HandleMenuCallback(c telebot.Context) error {
	defer c.Respond()
	return h.sendOrEditMenu(c)
}

func (h *Handlers) sendOrEditMenu(c telebot.Context) error {
	userIDStr := c.Get("user_id").(string)

	// Se houver estado pendente, vamos limpar
	_ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)

	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		if c.Callback() != nil {
			return c.Edit("⚠️ Nenhuma carteira encontrada na sua conta.")
		}
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	menu := &telebot.ReplyMarkup{}
	btnResumo := menu.Data("📊 Resumo da Carteira", "btn_resumo")
	btnProventos := menu.Data("💸 Ver Proventos", "btn_proventos")
	btnHistory := menu.Data("📜 Histórico", "btn_history")
	btnRendaFixa := menu.Data("🏛️ Renda Fixa", "btn_renda_fixa")
	btnOperacao := menu.Data("💵 Lançar Operação", "btn_operacao")

	rows := []telebot.Row{
		menu.Row(btnResumo),
		menu.Row(btnProventos),
		menu.Row(btnHistory),
		menu.Row(btnRendaFixa),
		menu.Row(btnOperacao),
	}

	if len(portfolios) > 1 {
		btnTrocarCarteira := menu.Data("🔄 Trocar Carteira", "btn_change_portfolio")
		rows = append(rows, menu.Row(btnTrocarCarteira))
	}

	menu.Inline(rows...)

	_, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	msgText := fmt.Sprintf("🏢 *Carteira Ativa:* %s\nEscolha uma opção:", portfolioName)

	if c.Callback() != nil {
		return c.Edit(msgText, telebot.ModeMarkdown, menu)
	}
	return c.Send(msgText, telebot.ModeMarkdown, menu)
}

func (h *Handlers) resolveActivePortfolio(ctx context.Context, chatID int64, portfolios []portfolio.Portfolio) (string, string) {
	if len(portfolios) == 0 {
		return "", ""
	}

	activeID, err := h.svc.GetActivePortfolio(ctx, chatID)
	if err == nil && activeID != "" {
		for _, p := range portfolios {
			if p.ID == activeID {
				return p.ID, p.Name
			}
		}
	}
	// Fallback para a primeira
	return portfolios[0].ID, portfolios[0].Name
}
