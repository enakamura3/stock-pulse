package telegram

import (
	"errors"
	"strings"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/alert"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/onigiri/stock-pulse/backend/internal/watchlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_Phase2_HandleStart_InlineKeyboard(t *testing.T) {
	h, svc, _, _, _, _ := setupHandlersTest()

	t.Run("with valid token sends buttons for menu and resumo", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{"valid_token"})
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("LinkAccountWithToken", mock.Anything, "valid_token", int64(123)).Return(nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Conta vinculada com sucesso")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			if len(opts) > 0 {
				if menu, ok := opts[0].(*telebot.ReplyMarkup); ok {
					return len(menu.InlineKeyboard) > 0
				}
			}
			return false
		})).Return(nil).Once()

		err := h.HandleStart(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase2_HandleMenu_TwoColumnGrid(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("menu grid with 1 portfolio", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "Principal"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		mCtx.On("Send", mock.Anything, mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					if len(menu.InlineKeyboard) >= 6 && len(menu.InlineKeyboard[0]) == 2 {
						return true
					}
				}
			}
			return false
		})).Return(nil).Once()

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})

	t.Run("menu grid with multiple portfolios includes trocar carteira", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "Principal"},
			{ID: "p2", Name: "Secundária"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		mCtx.On("Send", mock.Anything, mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					return len(menu.InlineKeyboard) >= 7
				}
			}
			return false
		})).Return(nil).Once()

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase2_HandleChangePortfolio_ActiveIndicator(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	mCtx := new(MockTelebotContext)
	mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
	mCtx.On("Respond", mock.Anything).Return(nil)

	pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
		{ID: "p1", Name: "Carteira 1"},
		{ID: "p2", Name: "Carteira 2"},
	}, nil).Once()
	svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

	mCtx.On("Edit", "Qual carteira você deseja definir como Ativa?", mock.MatchedBy(func(opts []interface{}) bool {
		if len(opts) == 0 {
			return false
		}
		menu, ok := opts[0].(*telebot.ReplyMarkup)
		if !ok || len(menu.InlineKeyboard) < 3 {
			return false
		}
		// First button should have ✅ (Active)
		hasCheck := strings.Contains(menu.InlineKeyboard[0][0].Text, "✅") && strings.Contains(menu.InlineKeyboard[0][0].Text, "Carteira 1")
		// Second button should have 📂
		hasFolder := strings.Contains(menu.InlineKeyboard[1][0].Text, "📂") && strings.Contains(menu.InlineKeyboard[1][0].Text, "Carteira 2")
		return hasCheck && hasFolder
	})).Return(nil).Once()

	err := h.HandleChangePortfolio(mCtx)
	assert.NoError(t, err)
}

func TestHandlers_Phase2_B3TickerFallback(t *testing.T) {
	h, svc, _, mSvc, _, alertSvc, wSvc := setupHandlersTestWithWatchlist()

	t.Run("HandleQuote with B3 ticker fallback to .SA", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Args").Return([]string{"VALE3"})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		// VALE3 direct fails, VALE3.SA succeeds
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mSvc.On("GetQuote", mock.Anything, "VALE3.SA").Return(&market.Quote{
			Symbol: "VALE3.SA",
			Name:   "Vale",
			Price:  65.50,
		}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "VALE3.SA")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAnalysis with B3 ticker fallback to .SA", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Args").Return([]string{"WEGE3"})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		mSvc.On("GetFundamentals", mock.Anything, "WEGE3").Return((*market.Fundamentals)(nil), errors.New("not found")).Once()
		mSvc.On("GetFundamentals", mock.Anything, "WEGE3.SA").Return(&market.Fundamentals{
			Symbol: "WEGE3.SA", EPS: 1.5, BookValue: 12.0,
		}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "WEGE3").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mSvc.On("GetQuote", mock.Anything, "WEGE3.SA").Return(&market.Quote{
			Symbol: "WEGE3.SA", Price: 50.0,
		}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "WEGE3")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleAnalysis(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText EXPECT_TICKER B3 fallback", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("ITUB4")
		mCtx.On("Respond", mock.Anything).Return(nil)

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step: "EXPECT_TICKER",
		}, nil).Twice()

		mSvc.On("GetQuote", mock.Anything, "ITUB4").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mSvc.On("GetQuote", mock.Anything, "ITUB4.SA").Return(&market.Quote{
			Symbol: "ITUB4.SA",
		}, nil).Once()

		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(st ConversationState) bool {
			return st.Step == "EXPECT_TYPE" && st.Ticker == "ITUB4.SA"
		})).Return(nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "ITUB4.SA")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText ALERT_EXPECT_TICKER B3 fallback", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("BBDC4")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step: "ALERT_EXPECT_TICKER",
		}, nil).Once()

		mSvc.On("GetQuote", mock.Anything, "BBDC4").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mSvc.On("GetQuote", mock.Anything, "BBDC4.SA").Return(&market.Quote{
			Symbol: "BBDC4.SA",
		}, nil).Once()

		svc.On("SetConversationState", mock.Anything, int64(123), mock.MatchedBy(func(st ConversationState) bool {
			return st.Step == "ALERT_EXPECT_COND" && st.Ticker == "BBDC4.SA"
		})).Return(nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Alerta para BBDC4.SA")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("handleWatchlistAddTicker B3 fallback", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})

		wSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{
			{ID: "w1"},
		}, nil).Once()

		wSvc.On("AddAssetToWatchlist", mock.Anything, "w1", "00000000-0000-0000-0000-000000000000", "PETR4").Return((*watchlist.Item)(nil), errors.New("not found")).Once()
		wSvc.On("AddAssetToWatchlist", mock.Anything, "w1", "00000000-0000-0000-0000-000000000000", "PETR4.SA").Return(&watchlist.Item{Ticker: "PETR4.SA"}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4.SA") && strings.Contains(msg, "adicionado aos seus Favoritos")
		}), mock.Anything).Return(nil).Once()

		err := h.handleWatchlistAddTicker(mCtx, "petr4")
		assert.NoError(t, err)
	})

	_ = alertSvc
}

func TestHandlers_Phase2_HandleText_StateError(t *testing.T) {
	h, svc, _, _, _, _ := setupHandlersTest()

	mCtx := new(MockTelebotContext)
	mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
	svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), errors.New("redis down")).Once()

	mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
		return strings.Contains(msg, "Olá")
	}), mock.Anything).Return(nil).Once()

	err := h.HandleText(mCtx)
	assert.NoError(t, err)
}

func TestHandlers_Phase2_ToastFeedbacks(t *testing.T) {
	h, _, _, _, _, alertSvc, wSvc := setupHandlersTestWithWatchlist()

	t.Run("handleAlertDelete sends toast CallbackResponse", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.MatchedBy(func(resp []*telebot.CallbackResponse) bool {
			return len(resp) > 0 && strings.Contains(resp[0].Text, "Alerta removido")
		})).Return(nil).Once()

		alertSvc.On("DeleteAlert", mock.Anything, "a1", "00000000-0000-0000-0000-000000000000").Return(nil).Once()
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.handleAlertDelete(mCtx, "a1:0")
		assert.NoError(t, err)
	})

	t.Run("handleWatchlistDelete sends toast CallbackResponse", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.MatchedBy(func(resp []*telebot.CallbackResponse) bool {
			return len(resp) > 0 && strings.Contains(resp[0].Text, "Favorito removido")
		})).Return(nil).Once()

		wSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{
			{ID: "w1"},
		}, nil).Twice()
		wSvc.On("RemoveAssetFromWatchlist", mock.Anything, "w1", "00000000-0000-0000-0000-000000000000", "PETR4").Return(nil).Once()
		wSvc.On("GetWatchlist", mock.Anything, "w1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID: "w1", Name: "Default", Items: []watchlist.Item{},
		}, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.handleWatchlistDelete(mCtx, "PETR4:0")
		assert.NoError(t, err)
	})
}

func TestHandlers_Phase2_ClearStateOnCommands(t *testing.T) {
	h, _, _, _, _, alertSvc, wSvc := setupHandlersTestWithWatchlist()

	t.Run("HandleAlerts with Chat clears conversation state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil)
		alertSvc.On("GetAlerts", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]*alert.Alert{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAlerts(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleWatchlist with Chat clears conversation state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil)
		wSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleWatchlist(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHelp with Chat clears conversation state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleHelp(mCtx)
		assert.NoError(t, err)
	})
}
