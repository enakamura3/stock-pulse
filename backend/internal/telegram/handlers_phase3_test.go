package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/onigiri/stock-pulse/backend/internal/watchlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_Phase3_LaunchOperation_TwoColumn(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("HandleLaunchOperation with odd number of positions and long names", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Principal"},
		}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "PETR4", Name: "Petroleo Brasileiro S.A."}, // > 12 chars
			{Ticker: "VALE3", Name: "Vale"},
			{Ticker: "WEGE3", Name: "WEG"}, // Odd count: 3 items
		}
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_TICKER" && s.PortfolioID == "p1"
		})).Return(nil).Once()

		var sentMenu *telebot.ReplyMarkup
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Principal") && strings.Contains(msg, "digite o ticker")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					sentMenu = menu
					return true
				}
			}
			return false
		})).Return(nil).Once()

		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
		assert.NotNil(t, sentMenu)
		// Row 0: PETR4, VALE3 (2 columns)
		// Row 1: WEGE3 (1 column)
		// Row 2: Novo Ativo, Cancelar (2 columns)
		assert.Equal(t, 3, len(sentMenu.InlineKeyboard))
		assert.Equal(t, 2, len(sentMenu.InlineKeyboard[0]))
		assert.Equal(t, 1, len(sentMenu.InlineKeyboard[1]))
		assert.Equal(t, 2, len(sentMenu.InlineKeyboard[2]))
	})
}

func TestHandlers_Phase3_QuickOperation(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("handleQuickOperation with user ID error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.store = map[string]interface{}{"user_id": 12345} // invalid type for getUserID
		mCtx.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err := h.handleQuickOperation(mCtx, "PETR4")
		assert.Error(t, err)
	})

	t.Run("handleQuickOperation with 0 portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil).Once()

		err := h.handleQuickOperation(mCtx, "PETR4")
		assert.NoError(t, err)
	})

	t.Run("handleQuickOperation success and dynamic routing", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fbtn_op_quick_PETR4"})

		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Step:        "EXPECT_TYPE",
		}).Return(nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Você selecionou *PETR4*")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase3_QuickAlertCreate(t *testing.T) {
	h, svc, _, mSvc, _, _ := setupHandlersTest()

	t.Run("handleQuickAlertCreate with quote error and state error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		mSvc.On("GetQuote", mock.Anything, "UNKNOWN").Return((*market.Quote)(nil), errors.New("not found")).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(errors.New("redis error")).Once()
		mCtx.On("Send", "❌ Erro interno ao iniciar criação de alerta.", mock.Anything).Return(nil).Once()

		err := h.handleQuickAlertCreate(mCtx, "UNKNOWN")
		assert.NoError(t, err)
	})

	t.Run("handleQuickAlertCreate success text send and callback edit", func(t *testing.T) {
		// Send path
		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))

		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{
			Ticker: "PETR4.SA",
			Step:   "ALERT_EXPECT_COND",
		}).Return(nil).Once()
		mCtxSend.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Alerta para PETR4.SA")
		}), mock.Anything).Return(nil).Once()

		err := h.handleQuickAlertCreate(mCtxSend, "PETR4")
		assert.NoError(t, err)

		// Callback edit path via HandleDynamicCallback
		mCtxCb := new(MockTelebotContext)
		mCtxCb.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxCb.On("Respond", mock.Anything).Return(nil).Once()
		mCtxCb.On("Callback").Return(&telebot.Callback{Data: "\fbtn_alert_create_VALE3"})

		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{Ticker: "VALE3"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{
			Ticker: "VALE3",
			Step:   "ALERT_EXPECT_COND",
		}).Return(nil).Once()
		mCtxCb.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Alerta para VALE3")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtxCb)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase3_QuickWatchlistAdd(t *testing.T) {
	t.Run("watchlistSvc is nil", func(t *testing.T) {
		h := &Handlers{watchlistSvc: nil}
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Send", "⚠️ Módulo de Favoritos não está ativo.", mock.Anything).Return(nil).Once()

		err := h.handleQuickWatchlistAdd(mCtx, "PETR4")
		assert.NoError(t, err)
	})

	t.Run("userID error", func(t *testing.T) {
		h, _, _, _, _, _, _ := setupHandlersTestWithWatchlist()
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.store = map[string]interface{}{"user_id": 999}
		mCtx.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err := h.handleQuickWatchlistAdd(mCtx, "PETR4")
		assert.Error(t, err)
	})

	t.Run("getWatchlists error or empty", func(t *testing.T) {
		h, _, _, _, _, _, wlSvc := setupHandlersTestWithWatchlist()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{}, nil).Once()
		mCtx.On("Send", "❌ Erro ao buscar lista de favoritos.", mock.Anything).Return(nil).Once()

		err := h.handleQuickWatchlistAdd(mCtx, "PETR4")
		assert.NoError(t, err)
	})

	t.Run("AddAssetToWatchlist error and success with B3 fallback", func(t *testing.T) {
		h, _, _, _, _, _, wlSvc := setupHandlersTestWithWatchlist()

		// Error path
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxErr.On("Respond", mock.Anything).Return(nil).Once()

		wlLists := []watchlist.Watchlist{{ID: "wl-1", Name: "Minha Lista"}}
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(wlLists, nil).Once()
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "INVALID").Return(nil, errors.New("invalid asset")).Once()
		mCtxErr.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Não foi possível adicionar")
		}), mock.Anything).Return(nil).Once()

		err := h.handleQuickWatchlistAdd(mCtxErr, "INVALID")
		assert.NoError(t, err)

		// Success path with B3 .SA fallback via HandleDynamicCallback
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Respond", mock.Anything).Return(nil).Once()
		mCtxOk.On("Callback").Return(&telebot.Callback{Data: "\fbtn_wl_add_PETR4"})

		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(wlLists, nil).Once()
		// First fails on "PETR4", second succeeds on "PETR4.SA"
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "PETR4").Return(nil, errors.New("not found")).Once()
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "PETR4.SA").Return(&watchlist.Item{Ticker: "PETR4.SA"}, nil).Once()
		mCtxOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4.SA") && strings.Contains(msg, "adicionado aos seus Favoritos")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtxOk)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase3_SellValidation_And_ShortSellingGapF7(t *testing.T) {
	h, svc, pSvc, mSvc, _, _ := setupHandlersTest()

	t.Run("handleSetType with user ID error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.store = map[string]interface{}{"user_id": 12345}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TYPE"}, nil).Once()
		mCtx.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err := h.handleSetType(mCtx, "BUY")
		assert.Error(t, err)
	})

	t.Run("handleSetType SELL with no custody returns warning", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_TYPE",
			PortfolioID: "p1",
			Ticker:      "PETR4",
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		// Custody position empty
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{}, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Você não possui PETR4 em custódia")
		}), mock.Anything).Return(nil).Once()

		err := h.handleSetType(mCtx, "SELL")
		assert.NoError(t, err)
	})

	t.Run("handleSetType SELL with custody renders sell shortcuts", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_TYPE",
			PortfolioID: "p1",
			Ticker:      "PETR4",
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{
			{Ticker: "PETR4", Quantity: 100, AveragePrice: 30.0},
		}, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Posição em Custódia") && strings.Contains(msg, "PM: R$ 30,00")
		}), mock.Anything).Return(nil).Once()

		err := h.handleSetType(mCtx, "SELL")
		assert.NoError(t, err)
	})

	t.Run("HandleQtyAll and HandleQtyHalf validations and success", func(t *testing.T) {
		// HandleQtyAll invalid step
		mCtxInvalid := new(MockTelebotContext)
		mCtxInvalid.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxInvalid.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TYPE"}, nil).Once()
		mCtxInvalid.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err := h.HandleQtyAll(mCtxInvalid)
		assert.NoError(t, err)

		// HandleQtyAll userID error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUserErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxUserErr.store = map[string]interface{}{"user_id": 12345}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtxUserErr.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err = h.HandleQtyAll(mCtxUserErr)
		assert.Error(t, err)

		// HandleQtyAll zero custody
		mCtxZero := new(MockTelebotContext)
		mCtxZero.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxZero.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
		}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{}, nil).Once()
		mCtxZero.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "não possui custódia")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleQtyAll(mCtxZero)
		assert.NoError(t, err)

		// HandleQtyAll success
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Type:        "SELL",
		}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{
			{Ticker: "PETR4", Quantity: 80, AveragePrice: 32.0},
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_PRICE" && s.Quantity == 80
		})).Return(nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Price: 35.0, Currency: "BRL"}, nil).Once()
		mCtxOk.On("Edit", "Qual o preço unitário da transação? (ex: 15.50)", mock.Anything).Return(nil).Once()

		err = h.HandleQtyAll(mCtxOk)
		assert.NoError(t, err)

		// HandleQtyHalf invalid step
		mCtxHalfInvalid := new(MockTelebotContext)
		mCtxHalfInvalid.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxHalfInvalid.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TYPE"}, nil).Once()
		mCtxHalfInvalid.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err = h.HandleQtyHalf(mCtxHalfInvalid)
		assert.NoError(t, err)

		// HandleQtyHalf user error
		mCtxHalfUserErr := new(MockTelebotContext)
		mCtxHalfUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxHalfUserErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxHalfUserErr.store = map[string]interface{}{"user_id": 12345}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtxHalfUserErr.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err = h.HandleQtyHalf(mCtxHalfUserErr)
		assert.Error(t, err)

		// HandleQtyHalf zero custody
		mCtxHalfZero := new(MockTelebotContext)
		mCtxHalfZero.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxHalfZero.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
		}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{}, nil).Once()
		mCtxHalfZero.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "não possui custódia")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleQtyHalf(mCtxHalfZero)
		assert.NoError(t, err)

		// HandleQtyHalf success
		mCtxHalfOk := new(MockTelebotContext)
		mCtxHalfOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxHalfOk.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Type:        "SELL",
		}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{
			{Ticker: "PETR4", Quantity: 80, AveragePrice: 32.0},
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_PRICE" && s.Quantity == 40
		})).Return(nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Price: 35.0, Currency: "BRL"}, nil).Once()
		mCtxHalfOk.On("Edit", "Qual o preço unitário da transação? (ex: 15.50)", mock.Anything).Return(nil).Once()

		err = h.HandleQtyHalf(mCtxHalfOk)
		assert.NoError(t, err)
	})

	t.Run("handleSelectedQty validation for SELL exceeding custody", func(t *testing.T) {
		// Invalid qty <= 1e-6
		mCtxInv := new(MockTelebotContext)
		mCtxInv.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxInv.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtxInv.On("Edit", "⚠️ Quantidade inválida.", mock.Anything).Return(nil).Once()

		err := h.handleSelectedQty(mCtxInv, "0")
		assert.NoError(t, err)

		// SELL user error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUserErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxUserErr.store = map[string]interface{}{"user_id": 12345}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step: "EXPECT_QTY",
			Type: "SELL",
		}, nil).Once()
		mCtxUserErr.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err = h.handleSelectedQty(mCtxUserErr, "10")
		assert.Error(t, err)

		// SELL qty > custody
		mCtxExceed := new(MockTelebotContext)
		mCtxExceed.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxExceed.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Type:        "SELL",
		}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{
			{Ticker: "PETR4", Quantity: 50},
		}, nil).Once()
		mCtxExceed.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Quantidade indisponível para venda") && strings.Contains(msg, "50")
		}), mock.Anything).Return(nil).Once()

		err = h.handleSelectedQty(mCtxExceed, "60")
		assert.NoError(t, err)
	})

	t.Run("HandleText EXPECT_QTY for SELL with exceed custody and user error", func(t *testing.T) {
		// User error on SELL
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUserErr.On("Text").Return("50")
		mCtxUserErr.store = map[string]interface{}{"user_id": 12345}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step: "EXPECT_QTY",
			Type: "SELL",
		}, nil).Once()
		mCtxUserErr.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtxUserErr)
		assert.Error(t, err)

		// Exceed custody on SELL via HandleText
		mCtxExceed := new(MockTelebotContext)
		mCtxExceed.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxExceed.On("Text").Return("150")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Type:        "SELL",
		}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{}, []portfolio.Position{
			{Ticker: "PETR4", Quantity: 100},
		}, nil).Once()
		mCtxExceed.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Quantidade indisponível para venda")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxExceed)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase3_CurrentPriceShortcut(t *testing.T) {
	h, svc, _, mSvc, _, _ := setupHandlersTest()

	t.Run("askPrice with valid USD quote", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		state := &ConversationState{Ticker: "AAPL"}
		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{
			Price:    220.50,
			Currency: "USD",
		}, nil).Once()

		var sentMenu *telebot.ReplyMarkup
		mCtx.On("Send", "Qual o preço unitário da transação? (ex: 15.50)", mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					sentMenu = menu
					return true
				}
			}
			return false
		})).Return(nil).Once()

		err := h.askPrice(mCtx, state, false)
		assert.NoError(t, err)
		assert.NotNil(t, sentMenu)
		assert.True(t, len(sentMenu.InlineKeyboard) >= 2)
		// First row contains btn_price_cur_220.50
		assert.Contains(t, sentMenu.InlineKeyboard[0][0].Text, "US$ 220,50")
	})

	t.Run("handleSelectedPrice errors and success", func(t *testing.T) {
		// State invalid
		mCtxStateErr := new(MockTelebotContext)
		mCtxStateErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxStateErr.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtxStateErr.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err := h.handleSelectedPrice(mCtxStateErr, "10.0")
		assert.NoError(t, err)

		// Price invalid <= 1e-6
		mCtxPriceErr := new(MockTelebotContext)
		mCtxPriceErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxPriceErr.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_PRICE"}, nil).Once()
		mCtxPriceErr.On("Edit", "⚠️ Preço inválido.", mock.Anything).Return(nil).Once()

		err = h.handleSelectedPrice(mCtxPriceErr, "0")
		assert.NoError(t, err)

		// Success via HandleDynamicCallback
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Respond", mock.Anything).Return(nil).Once()
		mCtxOk.On("Callback").Return(&telebot.Callback{Data: "\fbtn_price_cur_38.50"})

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:   "EXPECT_PRICE",
			Ticker: "PETR4",
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_DATE" && s.UnitPrice == 38.50
		})).Return(nil).Once()
		mCtxOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Data da Operação")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtxOk)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase3_YesterdayDateShortcut(t *testing.T) {
	h, svc, _, _, _, _ := setupHandlersTest()

	t.Run("HandleDateYesterday invalid step and success", func(t *testing.T) {
		// Invalid step
		mCtxInv := new(MockTelebotContext)
		mCtxInv.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxInv.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtxInv.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err := h.HandleDateYesterday(mCtxInv)
		assert.NoError(t, err)

		// Success
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:   "EXPECT_DATE",
			Ticker: "PETR4",
		}, nil).Once()
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_FEE" && s.ExecutedAt == yesterday
		})).Return(nil).Once()
		mCtxOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Taxas / Corretagem")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleDateYesterday(mCtxOk)
		assert.NoError(t, err)
	})

	t.Run("HandleText EXPECT_DATE ontem, yesterday and invalid format", func(t *testing.T) {
		// Ontem
		mCtxOntem := new(MockTelebotContext)
		mCtxOntem.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOntem.On("Text").Return("ontem")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:   "EXPECT_DATE",
			Ticker: "PETR4",
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_FEE"
		})).Return(nil).Once()
		mCtxOntem.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Taxas / Corretagem")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtxOntem)
		assert.NoError(t, err)

		// Yesterday
		mCtxYest := new(MockTelebotContext)
		mCtxYest.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxYest.On("Text").Return("yesterday")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:   "EXPECT_DATE",
			Ticker: "PETR4",
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(s ConversationState) bool {
			return s.Step == "EXPECT_FEE"
		})).Return(nil).Once()
		mCtxYest.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Taxas / Corretagem")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxYest)
		assert.NoError(t, err)

		// Invalid format
		mCtxInv := new(MockTelebotContext)
		mCtxInv.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxInv.On("Text").Return("data-invalida")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:   "EXPECT_DATE",
			Ticker: "PETR4",
		}, nil).Once()
		mCtxInv.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Formato de data inválido")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxInv)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase3_HistoryDeletion_And_CustodyHelpers(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("HandleHistory renders delete buttons for displayed transactions", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("")

		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		txs := []portfolio.Transaction{
			{ID: "tx-1", Ticker: "PETR4", Type: "BUY", Quantity: 100, UnitPrice: 30, ExecutedAt: time.Now()},
			{ID: "tx-2", Ticker: "VALE3", Type: "SELL", Quantity: 50, UnitPrice: 60, ExecutedAt: time.Now()},
			{ID: "tx-3", Ticker: "WEGE3", Type: "BUY", Quantity: 10, UnitPrice: 40, ExecutedAt: time.Now()}, // Odd count
		}
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()

		var sentMenu *telebot.ReplyMarkup
		mCtx.On("Send", mock.Anything, mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					sentMenu = menu
					return true
				}
			}
			return false
		})).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
		assert.NotNil(t, sentMenu)
		// Check that delete buttons are in rows
		foundDelBtn := false
		for _, row := range sentMenu.InlineKeyboard {
			for _, btn := range row {
				if strings.HasPrefix(btn.Unique, "btn_tx_del_") || strings.HasPrefix(btn.Data, "btn_tx_del_") {
					foundDelBtn = true
				}
			}
		}
		assert.True(t, foundDelBtn)
	})

	t.Run("handleDeleteTransaction errors and success via DynamicCallback", func(t *testing.T) {
		// User ID error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUserErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxUserErr.store = map[string]interface{}{"user_id": 12345}
		mCtxUserErr.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()

		err := h.handleDeleteTransaction(mCtxUserErr, "tx-1")
		assert.Error(t, err)

		// 0 Portfolios
		mCtxZero := new(MockTelebotContext)
		mCtxZero.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxZero.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtxZero.On("Edit", "⚠️ Carteira não encontrada.", mock.Anything).Return(nil).Once()

		err = h.handleDeleteTransaction(mCtxZero, "tx-1")
		assert.NoError(t, err)

		// Delete error
		mCtxDelErr := new(MockTelebotContext)
		mCtxDelErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxDelErr.On("Respond", mock.Anything).Return(nil).Once()
		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("DeleteTransaction", mock.Anything, "tx-1", "p1", "00000000-0000-0000-0000-000000000000").Return(errors.New("db error")).Once()
		mCtxDelErr.On("Edit", "❌ Erro ao excluir a operação. Ela pode já ter sido removida.", mock.Anything).Return(nil).Once()

		err = h.handleDeleteTransaction(mCtxDelErr, "tx-1")
		assert.NoError(t, err)

		// Success via HandleDynamicCallback
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Respond", mock.Anything).Return(nil).Once()
		mCtxOk.On("Callback").Return(&telebot.Callback{Data: "\fbtn_tx_del_tx-99"})

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("DeleteTransaction", mock.Anything, "tx-99", "p1", "00000000-0000-0000-0000-000000000000").Return(nil).Once()
		mCtxOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Operação desfeita com sucesso")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleDynamicCallback(mCtxOk)
		assert.NoError(t, err)
	})

	t.Run("getCustodyPosition error and B3 normalization", func(t *testing.T) {
		ctx := context.Background()

		// Details error
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-1").Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("query error")).Once()
		pos, err := h.getCustodyPosition(ctx, "p1", "user-1", "PETR4")
		assert.Error(t, err)
		assert.Nil(t, pos)

		// B3 ticker match when stored with .SA
		positions := []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 100},
			{Ticker: "VALE3", Quantity: 50},
		}
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-1").Return(&portfolio.Portfolio{}, positions, nil).Twice()

		pos, err = h.getCustodyPosition(ctx, "p1", "user-1", "petr4")
		assert.NoError(t, err)
		assert.NotNil(t, pos)
		assert.Equal(t, "PETR4.SA", pos.Ticker)

		// Not found
		pos, err = h.getCustodyPosition(ctx, "p1", "user-1", "ITUB4")
		assert.NoError(t, err)
		assert.Nil(t, pos)
	})
}
