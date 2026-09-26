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
	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

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
	btnAlertas := menu.Data("🔔 Meus Alertas", "btn_alerts")
	btnCotacao := menu.Data("📈 Cotação Rápida", "btn_cotacao")

	btnHelp := menu.Data("❓ Ajuda e Comandos", "btn_help")

	rows := []telebot.Row{
		menu.Row(btnResumo),
		menu.Row(btnProventos),
		menu.Row(btnHistory),
		menu.Row(btnRendaFixa),
		menu.Row(btnOperacao),
		menu.Row(btnAlertas),
		menu.Row(btnCotacao),
	}

	if len(portfolios) > 1 {
		btnTrocarCarteira := menu.Data("🔄 Trocar Carteira", "btn_change_portfolio")
		rows = append(rows, menu.Row(btnTrocarCarteira))
	}
	rows = append(rows, menu.Row(btnHelp))

	menu.Inline(rows...)

	_, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
	msgText := fmt.Sprintf("🏢 *Carteira Ativa:* %s\nEscolha uma opção:", escapeMarkdown(portfolioName))

	if c.Callback() != nil {
		return c.Edit(msgText, telebot.ModeMarkdown, menu)
	}
	return c.Send(msgText, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleHelp(c telebot.Context) error {
	msg := "🤖 *Comandos do Stock Pulse Bot*\n\n"
	msg += "• /menu — Abre o menu principal interativo\n"
	msg += "• /cotacao `<ticker>` — Consulta cotação rápida (ex: `/cotacao PETR4`)\n"
	msg += "• /analise `<ticker>` — Análise fundamentalista completa (ex: `/analise WEGE3`)\n"
	msg += "• /agenda — Exibe os proventos previstos para os próximos 30 dias\n"
	msg += "• /desfazer — Desfaz a última transação lançada na carteira ativa\n"
	msg += "• /help — Exibe esta mensagem de ajuda\n\n"
	msg += "💡 *Dica:* Você também pode usar todos os recursos clicando nos botões interativos do /menu."

	menu := &telebot.ReplyMarkup{}
	btnMenu := menu.Data("🏠 Menu Principal", "btn_menu")
	menu.Inline(menu.Row(btnMenu))

	if c.Callback() != nil {
		defer c.Respond()
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
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
	// Fallback para a carteira padrão, se definida, ou a primeira
	for _, p := range portfolios {
		if p.IsDefault {
			return p.ID, p.Name
		}
	}
	return portfolios[0].ID, portfolios[0].Name
}
