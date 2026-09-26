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

func (h *Handlers) HandleWatchlist(c telebot.Context) error {
	defer c.Respond()
	page := 0
	if rawData := c.Data(); rawData != "" {
		fmt.Sscanf(rawData, "%d", &page)
	}
	return h.renderWatchlist(c, page)
}

func (h *Handlers) renderWatchlist(c telebot.Context, page int) error {
	if h.watchlistSvc == nil {
		if c.Callback() != nil {
			return c.Edit("⚠️ Módulo de Favoritos não está ativo.")
		}
		return c.Send("⚠️ Módulo de Favoritos não está ativo.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	lists, err := h.watchlistSvc.GetWatchlists(context.Background(), userIDStr)
	if err != nil || len(lists) == 0 {
		slog.Error("Falha ao buscar listas de favoritos", "error", err, "user_id", userIDStr)
		if c.Callback() != nil {
			return c.Edit("❌ Ocorreu um erro ao buscar sua lista de favoritos.")
		}
		return c.Send("❌ Ocorreu um erro ao buscar sua lista de favoritos.")
	}

	activeList := lists[0]
	wl, err := h.watchlistSvc.GetWatchlist(context.Background(), activeList.ID, userIDStr)
	if err != nil {
		slog.Error("Falha ao buscar itens da lista de favoritos", "error", err, "list_id", activeList.ID)
		if c.Callback() != nil {
			return c.Edit("❌ Ocorreu um erro ao buscar itens dos favoritos.")
		}
		return c.Send("❌ Ocorreu um erro ao buscar itens dos favoritos.")
	}

	p := message.NewPrinter(language.BrazilianPortuguese)
	menu := &telebot.ReplyMarkup{}

	if len(wl.Items) == 0 {
		msg := fmt.Sprintf("⭐ *Meus Favoritos (%s)*\n\nVocê ainda não possui nenhum ativo na sua lista de favoritos.\n\nUse o botão abaixo para adicionar ativos!", escapeMarkdown(wl.Name))
		btnAdd := menu.Data("➕ Adicionar Ativo", "btn_wl_add")
		btnMenu := menu.Data("🏠 Voltar ao Menu", "btn_menu")
		menu.Inline(menu.Row(btnAdd), menu.Row(btnMenu))

		if c.Callback() != nil {
			err = c.Edit(msg, telebot.ModeMarkdown, menu)
		} else {
			err = c.Send(msg, telebot.ModeMarkdown, menu)
		}
		if err != nil && strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return err
	}

	pageSize := 5
	totalPages := (len(wl.Items) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	start := page * pageSize
	end := start + pageSize
	if end > len(wl.Items) {
		end = len(wl.Items)
	}

	msg := fmt.Sprintf("⭐ *Meus Favoritos (%s)*\n", escapeMarkdown(wl.Name))
	if totalPages > 1 {
		msg += p.Sprintf("_Página %d de %d_\n\n", page+1, totalPages)
	} else {
		msg += "\n"
	}

	var rows []telebot.Row
	for _, item := range wl.Items[start:end] {
		emoji := "⚪"
		if item.ChangePercent > 1e-6 {
			emoji = "🟢"
		} else if item.ChangePercent < -1e-6 {
			emoji = "🔴"
		}
		curr := item.Currency
		if curr == "" {
			curr = "BRL"
		}
		msg += p.Sprintf("%s `%s` — %s %s (%+.2f%%)\n",
			emoji, item.Ticker, getCurrencySymbol(curr), formatFinancialPrice(nil, item.Price), item.ChangePercent)

		btnQuote := menu.Data("📈 "+item.Ticker, "btn_quote_"+item.Ticker)
		btnDel := menu.Data("🗑️ "+item.Ticker, fmt.Sprintf("btn_wl_del_%s:%d", item.Ticker, page))
		rows = append(rows, menu.Row(btnQuote, btnDel))
	}
	msg += p.Sprintf("\n_Total: %d ativo(s)_", len(wl.Items))

	var navBtns []telebot.Btn
	if page > 0 {
		navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_watchlist", fmt.Sprintf("%d", page-1)))
	}
	if end < len(wl.Items) {
		navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_watchlist", fmt.Sprintf("%d", page+1)))
	}
	if len(navBtns) > 0 {
		rows = append(rows, menu.Row(navBtns...))
	}

	btnAdd := menu.Data("➕ Adicionar Ativo", "btn_wl_add")
	btnRefresh := menu.Data("🔄 Atualizar", "btn_watchlist", fmt.Sprintf("%d", page))
	btnMenu := menu.Data("🏠 Voltar ao Menu", "btn_menu")
	rows = append(rows, menu.Row(btnAdd, btnRefresh), menu.Row(btnMenu))
	menu.Inline(rows...)

	if c.Callback() != nil {
		err = c.Edit(msg, telebot.ModeMarkdown, menu)
	} else {
		err = c.Send(msg, telebot.ModeMarkdown, menu)
	}
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	return err
}

func (h *Handlers) HandleWatchlistAdd(c telebot.Context) error {
	defer c.Respond()
	if h.watchlistSvc == nil {
		return c.Edit("⚠️ Módulo de Favoritos não está ativo.")
	}

	err := h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
		Step: "WL_EXPECT_TICKER",
	})
	if err != nil {
		return c.Edit("❌ Erro interno ao iniciar adição aos favoritos.")
	}

	menu := &telebot.ReplyMarkup{}
	btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
	menu.Inline(menu.Row(btnCancel))

	msg := "⭐ *Adicionar aos Favoritos*\n\nDigite o código do ativo que deseja favoritar:\n_(ex: PETR4, VALE3.SA, AAPL, BTC-USD)_"
	if c.Callback() != nil {
		return c.Edit(msg, telebot.ModeMarkdown, menu)
	}
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleWatchlistAddTicker(c telebot.Context, text string) error {
	ticker := strings.ToUpper(strings.TrimSpace(text))
	if ticker == "" {
		return c.Send("⚠️ Código de ativo inválido. Envie um código válido:")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	lists, err := h.watchlistSvc.GetWatchlists(context.Background(), userIDStr)
	if err != nil || len(lists) == 0 {
		return c.Send("❌ Erro ao buscar lista de favoritos.")
	}

	_, err = h.watchlistSvc.AddAssetToWatchlist(context.Background(), lists[0].ID, userIDStr, ticker)
	if err != nil {
		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
		menu.Inline(menu.Row(btnCancel))
		return c.Send(fmt.Sprintf("⚠️ Não foi possível adicionar *%s*: %s\n\nVerifique se o código está correto ou tente novamente:", escapeMarkdown(ticker), err.Error()), telebot.ModeMarkdown, menu)
	}

	_ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)

	menu := &telebot.ReplyMarkup{}
	btnWatchlist := menu.Data("⭐ Ver Favoritos", "btn_watchlist")
	btnMenu := menu.Data("🏠 Menu", "btn_menu")
	menu.Inline(menu.Row(btnWatchlist, btnMenu))

	msg := fmt.Sprintf("✅ Ativo *`%s`* adicionado aos seus Favoritos com sucesso!", escapeMarkdown(ticker))
	return c.Send(msg, telebot.ModeMarkdown, menu)
}

func (h *Handlers) handleWatchlistDelete(c telebot.Context, payload string) error {
	defer c.Respond()
	if h.watchlistSvc == nil {
		return c.Edit("⚠️ Módulo de Favoritos não está ativo.")
	}

	userIDStr, err := h.getUserID(c)
	if err != nil {
		return err
	}

	ticker := payload
	page := 0
	if parts := strings.SplitN(payload, ":", 2); len(parts) == 2 {
		ticker = parts[0]
		fmt.Sscanf(parts[1], "%d", &page)
	}

	lists, err := h.watchlistSvc.GetWatchlists(context.Background(), userIDStr)
	if err == nil && len(lists) > 0 {
		err = h.watchlistSvc.RemoveAssetFromWatchlist(context.Background(), lists[0].ID, userIDStr, ticker)
		if err != nil {
			slog.Error("Falha ao remover ativo dos favoritos", "error", err, "ticker", ticker)
		}
	}

	return h.renderWatchlist(c, page)
}
