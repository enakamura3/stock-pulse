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
		ctx, cancel := h.getContext(c)
		defer cancel()
		err := h.svc.LinkAccountWithToken(ctx, token, c.Chat().ID)
		if err != nil {
			if strings.Contains(err.Error(), "inválido ou expirado") {
				return c.Send("❌ O link de vinculação é inválido ou expirou. Gere um novo no Stock Pulse.")
			}
			slog.Error("Erro ao vincular conta telegram", "error", err, "chat_id", c.Chat().ID)
			return c.Send("❌ Ocorreu um erro interno ao vincular sua conta. Tente novamente.")
		}
		menu := &telebot.ReplyMarkup{}
		btnMenu := menu.Data("🏠 Abrir Menu Principal", "btn_menu")
		btnResumo := menu.Data("📊 Ver Resumo", "btn_resumo")
		menu.Inline(menu.Row(btnMenu, btnResumo))
		return c.Send("✅ Conta vinculada com sucesso! Bem-vindo ao Stock Pulse.\n\nEscolha uma opção para começar:", menu)
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

	ctx, cancel := h.getContext(c)
	defer cancel()

	var chatID int64
	if c.Chat() != nil {
		chatID = c.Chat().ID
		_ = h.svc.ClearConversationState(ctx, chatID)
	}

	portfolios, err := h.portfolioSvc.GetPortfolios(ctx, userIDStr)
	if err != nil || len(portfolios) == 0 {
		if c.Callback() != nil {
			return c.Edit("⚠️ Nenhuma carteira encontrada na sua conta.")
		}
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	menu := &telebot.ReplyMarkup{}
	btnResumo := menu.Data("📊 Resumo", "btn_resumo")
	btnAtivos := menu.Data("📋 Meus Ativos", "btn_ativos")
	btnProventos := menu.Data("💸 Proventos", "btn_proventos")
	btnAgenda := menu.Data("📅 Agenda", "btn_agenda")
	btnRendaFixa := menu.Data("🏛️ Renda Fixa", "btn_renda_fixa")
	btnHistory := menu.Data("📜 Histórico", "btn_history")
	btnOperacao := menu.Data("💵 Lançar Op.", "btn_operacao")
	btnAlertas := menu.Data("🔔 Alertas", "btn_alerts")
	btnFavoritos := menu.Data("⭐ Favoritos", "btn_watchlist")
	btnCotacao := menu.Data("📈 Cotação", "btn_cotacao")
	btnAnalise := menu.Data("🔍 Análise", "btn_analise")
	btnHelp := menu.Data("❓ Ajuda e Comandos", "btn_help")

	rows := []telebot.Row{
		menu.Row(btnResumo, btnAtivos),
		menu.Row(btnProventos, btnAgenda),
		menu.Row(btnRendaFixa, btnHistory),
		menu.Row(btnOperacao, btnAlertas),
		menu.Row(btnFavoritos, btnCotacao),
	}

	if len(portfolios) > 1 {
		btnTrocarCarteira := menu.Data("🔄 Trocar Carteira", "btn_change_portfolio")
		rows = append(rows, menu.Row(btnAnalise, btnTrocarCarteira))
		rows = append(rows, menu.Row(btnHelp))
	} else {
		rows = append(rows, menu.Row(btnAnalise, btnHelp))
	}

	menu.Inline(rows...)

	_, portfolioName := h.resolveActivePortfolio(ctx, chatID, portfolios)
	msgText := fmt.Sprintf("🏢 *Carteira Ativa:* %s\nEscolha uma opção:", escapeMarkdown(portfolioName))

	if c.Callback() != nil {
		return c.Edit(msgText, telebot.ModeMarkdown, menu)
	}
	return c.Send(msgText, telebot.ModeMarkdown, menu)
}

func (h *Handlers) HandleHelp(c telebot.Context) error {
	ctx, cancel := h.getContext(c)
	defer cancel()

	if c.Chat() != nil {
		_ = h.svc.ClearConversationState(ctx, c.Chat().ID)
	}
	msg := "🤖 *Comandos do Stock Pulse Bot*\n\n"
	msg += "• /menu — Abre o menu principal interativo\n"
	msg += "• /resumo — Resumo consolidado da carteira ativa\n"
	msg += "• /ativos — Lista seus ativos, preço médio e rentabilidade\n"
	msg += "• /proventos — Extrato e totais de proventos (ou `/proventos VIVT3`)\n"
	msg += "• /agenda — Exibe os proventos previstos para os próximos 30 dias\n"
	msg += "• /rendafixa — Posições de Renda Fixa e Tesouro Direto\n"
	msg += "• /historico — Histórico de operações (ou `/historico PETR4`)\n"
	msg += "• /operacao — Lançar operação (ou `/operacao PETR4` / `/operacao C PETR4 100 36,50`)\n"
	msg += "• /desfazer — Desfaz a última transação lançada na carteira ativa\n"
	msg += "• /cotacao `<ticker>` — Consulta cotação rápida (ex: `/cotacao PETR4`)\n"
	msg += "• /analise `<ticker>` — Análise fundamentalista completa (ex: `/analise WEGE3`)\n"
	msg += "• /alertas — Gerencia seus alertas de preço\n"
	msg += "• /watchlist — Lista e gerencia seus ativos favoritos\n"
	msg += "• /help — Exibe esta mensagem de ajuda\n\n"
	msg += "💡 *Dicas:*\n"
	msg += "• Digite um ativo direto no chat (ex: `PETR4`, `PETR4 VALE3`, `BTC-USD`, `$AAPL`) para ver a cotação na hora.\n"
	msg += "• No lançamento rápido, você pode informar data (`AAAA-MM-DD`) e taxa: `/operacao C PETR4 100 36,50 2026-03-15 4,50`.\n"
	msg += "• Todos os recursos também estão nos botões interativos do /menu."

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
	return resolveActivePortfolio(ctx, h.svc, chatID, portfolios)
}

func resolveActivePortfolio(ctx context.Context, svc Service, chatID int64, portfolios []portfolio.Portfolio) (string, string) {
	if len(portfolios) == 0 {
		return "", ""
	}

	if svc != nil && chatID != 0 {
		activeID, err := svc.GetActivePortfolio(ctx, chatID)
		if err == nil && activeID != "" {
			for _, p := range portfolios {
				if p.ID == activeID {
					return p.ID, p.Name
				}
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
