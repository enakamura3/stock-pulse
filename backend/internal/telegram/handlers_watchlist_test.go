package telegram

import (
	"errors"
	"strings"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/watchlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_Watchlist(t *testing.T) {
	t.Run("HandleWatchlist - nil service", func(t *testing.T) {
		hNil := NewHandlers(nil, nil, nil, nil, nil)
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Edit", "⚠️ Módulo de Favoritos não está ativo.", mock.Anything).Return(nil).Once()

		err := hNil.HandleWatchlist(mCtx)
		assert.NoError(t, err)

		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))
		mCtxSend.On("Send", "⚠️ Módulo de Favoritos não está ativo.", mock.Anything).Return(nil).Once()

		err = hNil.HandleWatchlist(mCtxSend)
		assert.NoError(t, err)
	})

	t.Run("HandleWatchlist - getUserID error", func(t *testing.T) {
		h, _, _, _, _, _, _ := setupHandlersTestWithWatchlist()
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.Set("user_id", nil)
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		mCtx.On("Send", "⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.", mock.Anything).Return(nil).Once()

		err := h.HandleWatchlist(mCtx)
		assert.Error(t, err)
	})

	t.Run("HandleWatchlist - GetWatchlists error", func(t *testing.T) {
		h, _, _, _, _, _, wSvc := setupHandlersTestWithWatchlist()
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Get", "user_id").Return("u1")
		mCtx.On("Callback").Return(&telebot.Callback{})

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(([]watchlist.Watchlist)(nil), errors.New("db error")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar sua lista de favoritos.", mock.Anything).Return(nil).Once()

		err := h.HandleWatchlist(mCtx)
		assert.NoError(t, err)

		// Send branch
		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Get", "user_id").Return("u1")
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return([]watchlist.Watchlist{}, nil).Once()
		mCtxSend.On("Send", "❌ Ocorreu um erro ao buscar sua lista de favoritos.", mock.Anything).Return(nil).Once()

		err = h.HandleWatchlist(mCtxSend)
		assert.NoError(t, err)
	})

	t.Run("HandleWatchlist - GetWatchlist error", func(t *testing.T) {
		h, _, _, _, _, _, wSvc := setupHandlersTestWithWatchlist()
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Get", "user_id").Return("u1")
		mCtx.On("Callback").Return(&telebot.Callback{})

		lists := []watchlist.Watchlist{{ID: "wl1", Name: "Favoritos"}}
		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return((*watchlist.Watchlist)(nil), errors.New("not found")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar itens dos favoritos.", mock.Anything).Return(nil).Once()

		err := h.HandleWatchlist(mCtx)
		assert.NoError(t, err)

		// Send branch
		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Get", "user_id").Return("u1")
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return((*watchlist.Watchlist)(nil), errors.New("not found")).Once()
		mCtxSend.On("Send", "❌ Ocorreu um erro ao buscar itens dos favoritos.", mock.Anything).Return(nil).Once()

		err = h.HandleWatchlist(mCtxSend)
		assert.NoError(t, err)
	})

	t.Run("HandleWatchlist - empty items", func(t *testing.T) {
		h, _, _, _, _, _, wSvc := setupHandlersTestWithWatchlist()

		// Callback context
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Get", "user_id").Return("u1")
		mCtx.On("Callback").Return(&telebot.Callback{})

		lists := []watchlist.Watchlist{{ID: "wl1", Name: "Minha_Lista*"}}
		wl := &watchlist.Watchlist{ID: "wl1", Name: "Minha_Lista*", Items: []watchlist.Item{}}

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, `Minha\_Lista\*`)
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleWatchlist(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, `Minha\_Lista\*`)

		// Send context
		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Get", "user_id").Return("u1")
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()
		mCtxSend.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleWatchlist(mCtxSend)
		assert.NoError(t, err)

		// Message not modified on empty
		mCtxNotMod := new(MockTelebotContext)
		mCtxNotMod.On("Respond", mock.Anything).Return(nil).Once()
		mCtxNotMod.On("Get", "user_id").Return("u1")
		mCtxNotMod.On("Callback").Return(&telebot.Callback{})

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()
		mCtxNotMod.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err = h.HandleWatchlist(mCtxNotMod)
		assert.NoError(t, err)
	})

	t.Run("HandleWatchlist - items with pagination and status emojis", func(t *testing.T) {
		h, _, _, _, _, _, wSvc := setupHandlersTestWithWatchlist()

		items := []watchlist.Item{
			{Ticker: "PETR4", Price: 35.5, ChangePercent: 1.5, Currency: "BRL"},
			{Ticker: "VALE3", Price: 60.0, ChangePercent: -0.8, Currency: "BRL"},
			{Ticker: "BBAS3", Price: 28.0, ChangePercent: 0.0, Currency: ""},
			{Ticker: "AAPL", Price: 180.0, ChangePercent: 2.1, Currency: "USD"},
			{Ticker: "MSFT", Price: 420.0, ChangePercent: -1.2, Currency: "USD"},
			{Ticker: "NVDA", Price: 120.0, ChangePercent: 3.5, Currency: "USD"},
		}
		lists := []watchlist.Watchlist{{ID: "wl1", Name: "Favoritos"}}
		wl := &watchlist.Watchlist{ID: "wl1", Name: "Favoritos", Items: items}

		// Page 0
		mCtx0 := new(MockTelebotContext)
		mCtx0.On("Respond", mock.Anything).Return(nil).Once()
		mCtx0.On("Get", "user_id").Return("u1")
		mCtx0.On("Callback").Return(&telebot.Callback{})
		mCtx0.On("Data").Return("0").Once()

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()

		var msg0 string
		mCtx0.On("Edit", mock.MatchedBy(func(msg string) bool {
			msg0 = msg
			return strings.Contains(msg, "Página 1 de 2") && strings.Contains(msg, "PETR4")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleWatchlist(mCtx0)
		assert.NoError(t, err)
		assert.Contains(t, msg0, "🟢 `PETR4`")
		assert.Contains(t, msg0, "🔴 `VALE3`")
		assert.Contains(t, msg0, "⚪ `BBAS3`")

		// Page 1
		mCtx1 := new(MockTelebotContext)
		mCtx1.On("Respond", mock.Anything).Return(nil).Once()
		mCtx1.On("Get", "user_id").Return("u1")
		mCtx1.On("Callback").Return(&telebot.Callback{})
		mCtx1.On("Data").Return("1").Once()

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()

		var msg1 string
		mCtx1.On("Edit", mock.MatchedBy(func(msg string) bool {
			msg1 = msg
			return strings.Contains(msg, "Página 2 de 2") && strings.Contains(msg, "NVDA")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleWatchlist(mCtx1)
		assert.NoError(t, err)
		assert.Contains(t, msg1, "NVDA")

		// Bounds clamping: page < 0
		mCtxNeg := new(MockTelebotContext)
		mCtxNeg.On("Respond", mock.Anything).Return(nil).Once()
		mCtxNeg.On("Get", "user_id").Return("u1")
		mCtxNeg.On("Callback").Return((*telebot.Callback)(nil))
		mCtxNeg.On("Data").Return("-1").Once()

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()
		mCtxNeg.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleWatchlist(mCtxNeg)
		assert.NoError(t, err)

		// Bounds clamping: page >= totalPages
		mCtxOver := new(MockTelebotContext)
		mCtxOver.On("Respond", mock.Anything).Return(nil).Once()
		mCtxOver.On("Get", "user_id").Return("u1")
		mCtxOver.On("Callback").Return(&telebot.Callback{})
		mCtxOver.On("Data").Return("99").Once()

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()
		mCtxOver.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleWatchlist(mCtxOver)
		assert.NoError(t, err)

		// Message not modified
		mCtxNotMod := new(MockTelebotContext)
		mCtxNotMod.On("Respond", mock.Anything).Return(nil).Once()
		mCtxNotMod.On("Get", "user_id").Return("u1")
		mCtxNotMod.On("Callback").Return(&telebot.Callback{})
		mCtxNotMod.On("Data").Return("0").Once()

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()
		mCtxNotMod.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err = h.HandleWatchlist(mCtxNotMod)
		assert.NoError(t, err)
	})

	t.Run("HandleWatchlistAdd", func(t *testing.T) {
		hNil := NewHandlers(nil, nil, nil, nil, nil)
		mCtxNil := new(MockTelebotContext)
		mCtxNil.On("Respond", mock.Anything).Return(nil).Once()
		mCtxNil.On("Edit", "⚠️ Módulo de Favoritos não está ativo.", mock.Anything).Return(nil).Once()

		err := hNil.HandleWatchlistAdd(mCtxNil)
		assert.NoError(t, err)

		h, svc, _, _, _, _, _ := setupHandlersTestWithWatchlist()

		// Error setting state
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxErr.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "WL_EXPECT_TICKER"
		})).Return(errors.New("redis err")).Once()
		mCtxErr.On("Edit", "❌ Erro interno ao iniciar adição aos favoritos.", mock.Anything).Return(nil).Once()

		err = h.HandleWatchlistAdd(mCtxErr)
		assert.NoError(t, err)

		// Callback success
		mCtxCb := new(MockTelebotContext)
		mCtxCb.On("Respond", mock.Anything).Return(nil).Once()
		mCtxCb.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxCb.On("Callback").Return(&telebot.Callback{})
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "WL_EXPECT_TICKER"
		})).Return(nil).Once()
		mCtxCb.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleWatchlistAdd(mCtxCb)
		assert.NoError(t, err)

		// Send success
		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "WL_EXPECT_TICKER"
		})).Return(nil).Once()
		mCtxSend.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleWatchlistAdd(mCtxSend)
		assert.NoError(t, err)
	})

	t.Run("handleWatchlistAddTicker and HandleText routing", func(t *testing.T) {
		h, svc, _, _, _, _, wSvc := setupHandlersTestWithWatchlist()

		// Empty text
		mCtxEmpty := new(MockTelebotContext)
		mCtxEmpty.On("Text").Return("   ").Once()
		mCtxEmpty.On("Send", "⚠️ Código de ativo inválido. Envie um código válido:", mock.Anything).Return(nil).Once()

		err := h.handleWatchlistAddTicker(mCtxEmpty, "   ")
		assert.NoError(t, err)

		// getUserID error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Text").Return("PETR4").Once()
		mCtxUserErr.Set("user_id", nil)
		mCtxUserErr.On("Callback").Return((*telebot.Callback)(nil))
		mCtxUserErr.On("Send", "⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.", mock.Anything).Return(nil).Once()

		err = h.handleWatchlistAddTicker(mCtxUserErr, "PETR4")
		assert.Error(t, err)

		// GetWatchlists error
		mCtxListsErr := new(MockTelebotContext)
		mCtxListsErr.On("Text").Return("PETR4").Once()
		mCtxListsErr.On("Get", "user_id").Return("u1")
		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(([]watchlist.Watchlist)(nil), errors.New("err")).Once()
		mCtxListsErr.On("Send", "❌ Erro ao buscar lista de favoritos.", mock.Anything).Return(nil).Once()

		err = h.handleWatchlistAddTicker(mCtxListsErr, "PETR4")
		assert.NoError(t, err)

		// AddAssetToWatchlist error
		mCtxAddErr := new(MockTelebotContext)
		mCtxAddErr.On("Text").Return("INVALID").Once()
		mCtxAddErr.On("Get", "user_id").Return("u1")
		lists := []watchlist.Watchlist{{ID: "wl1"}}
		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("AddAssetToWatchlist", mock.Anything, "wl1", "u1", "INVALID").Return((*watchlist.Item)(nil), errors.New("ativo não existe")).Once()
		mCtxAddErr.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Não foi possível adicionar")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.handleWatchlistAddTicker(mCtxAddErr, "INVALID")
		assert.NoError(t, err)

		// Success through HandleText
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Text").Return("PETR4")
		mCtxOk.On("Get", "user_id").Return("u1")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step: "WL_EXPECT_TICKER",
		}, nil).Once()
		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Once()
		wSvc.On("AddAssetToWatchlist", mock.Anything, "wl1", "u1", "PETR4").Return(&watchlist.Item{Ticker: "PETR4"}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mCtxOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "adicionado aos seus Favoritos")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxOk)
		assert.NoError(t, err)
	})

	t.Run("handleWatchlistDelete and HandleDynamicCallback routing", func(t *testing.T) {
		hNil := NewHandlers(nil, nil, nil, nil, nil)
		mCtxNil := new(MockTelebotContext)
		mCtxNil.On("Respond", mock.Anything).Return(nil).Once()
		mCtxNil.On("Edit", "⚠️ Módulo de Favoritos não está ativo.", mock.Anything).Return(nil).Once()

		err := hNil.handleWatchlistDelete(mCtxNil, "PETR4:0")
		assert.NoError(t, err)

		h, _, _, _, _, _, wSvc := setupHandlersTestWithWatchlist()

		// getUserID error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Respond", mock.Anything).Return(nil).Twice()
		mCtxUserErr.Set("user_id", nil)
		mCtxUserErr.On("Callback").Return(&telebot.Callback{})
		mCtxUserErr.On("Edit", "⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.", mock.Anything).Return(nil).Once()

		err = h.handleWatchlistDelete(mCtxUserErr, "PETR4:0")
		assert.Error(t, err)

		// Success through HandleDynamicCallback
		mCtxDel := new(MockTelebotContext)
		mCtxDel.On("Respond", mock.Anything).Return(nil).Once()
		mCtxDel.On("Callback").Return(&telebot.Callback{Data: "\fbtn_wl_del_PETR4:0"})
		mCtxDel.On("Get", "user_id").Return("u1")
		mCtxDel.On("Data").Return("").Maybe()

		lists := []watchlist.Watchlist{{ID: "wl1", Name: "Favoritos"}}
		wl := &watchlist.Watchlist{ID: "wl1", Name: "Favoritos", Items: []watchlist.Item{}}
		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Twice()
		wSvc.On("RemoveAssetFromWatchlist", mock.Anything, "wl1", "u1", "PETR4").Return(nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()

		mCtxDel.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtxDel)
		assert.NoError(t, err)

		// Removal error logged and recovered
		mCtxDelErr := new(MockTelebotContext)
		mCtxDelErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxDelErr.On("Callback").Return(&telebot.Callback{Data: "\fbtn_wl_del_VALE3:0"})
		mCtxDelErr.On("Get", "user_id").Return("u1")
		mCtxDelErr.On("Data").Return("").Maybe()

		wSvc.On("GetWatchlists", mock.Anything, "u1").Return(lists, nil).Twice()
		wSvc.On("RemoveAssetFromWatchlist", mock.Anything, "wl1", "u1", "VALE3").Return(errors.New("remove error")).Once()
		wSvc.On("GetWatchlist", mock.Anything, "wl1", "u1").Return(wl, nil).Once()
		mCtxDelErr.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtxDelErr)
		assert.NoError(t, err)
	})
}
