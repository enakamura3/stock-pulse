package telegram

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_HandleLaunchOperation(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("err details", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("set state err", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return(&portfolio.Portfolio{}, []portfolio.Position{{Ticker: "AAPL", Name: "Apple"}}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(1), mock.Anything).Return(errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return(&portfolio.Portfolio{}, []portfolio.Position{{Ticker: "AAPL", Name: "Apple"}}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(1), mock.Anything).Return(nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_handleSelectedTicker(t *testing.T) {
	h, svc, _, _, _ := setupHandlersTest()

	t.Run("no state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return((*ConversationState)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.handleSelectedTicker(mCtx, "AAPL")
		assert.NoError(t, err)
	})

	t.Run("valid state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(1), mock.Anything).Return(nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := h.handleSelectedTicker(mCtx, "AAPL")
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleNewAsset(t *testing.T) {
	h, svc, _, _, _ := setupHandlersTest()
	t.Run("no state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return((*ConversationState)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleNewAsset(mCtx)
		assert.NoError(t, err)
	})
	t.Run("valid state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{}, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleNewAsset(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleSetType(t *testing.T) {
	h, svc, _, _, _ := setupHandlersTest()

	t.Run("HandleSetTypeBuy no state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return((*ConversationState)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleSetTypeBuy(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleSetTypeSell valid state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(1), mock.Anything).Return(nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleSetTypeSell(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleText(t *testing.T) {
	h, svc, pSvc, mSvc, _ := setupHandlersTest()

	t.Run("no state", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetConversationState", mock.Anything, int64(1)).Return((*ConversationState)(nil), errors.New("err")).Once()

		// Fallbacks to Menu
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_TICKER invalid", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("INVALID")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "INVALID").Return(nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_TICKER valid", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("AAPL")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_TICKER"}, nil).Twice()
		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(nil, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(1), mock.Anything).Return(nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_QTY invalid", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("abc")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_QTY valid", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("10,5")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(1), mock.Anything).Return(nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_PRICE invalid", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("-5")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_PRICE"}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_PRICE valid err save", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("150.5")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_PRICE", Quantity: 10, Ticker: "AAPL"}, nil).Once()
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("AddTransaction", mock.Anything, uID.String(), mock.Anything).Return((*portfolio.Transaction)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_PRICE valid success SELL", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("150.5")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_PRICE", Quantity: 10, Ticker: "AAPL", Type: "SELL"}, nil).Once()
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("AddTransaction", mock.Anything, uID.String(), mock.Anything).Return(&portfolio.Transaction{TotalCost: 1505, ExecutedAt: time.Now()}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(1)).Return(nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("EXPECT_PRICE valid success BUY", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("150.5")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "EXPECT_PRICE", Quantity: 10, Ticker: "AAPL", Type: "BUY"}, nil).Once()
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("AddTransaction", mock.Anything, uID.String(), mock.Anything).Return(&portfolio.Transaction{TotalCost: 1505, ExecutedAt: time.Now()}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(1)).Return(nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("OTHER_STEP", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Text").Return("150.5")
		svc.On("GetConversationState", mock.Anything, int64(1)).Return(&ConversationState{Step: "UNKNOWN"}, nil).Once()
		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})
}
