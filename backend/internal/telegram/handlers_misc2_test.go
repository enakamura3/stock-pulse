package telegram

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_HandleHistory(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("err fetch txs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", uID.String()).Return(([]portfolio.Transaction)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no txs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("") // pageStr
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", uID.String()).Return([]portfolio.Transaction{}, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("0") // pageStr
		mCtx.On("Message").Return((*telebot.Message)(nil))
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		txs := []portfolio.Transaction{
			{Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, TotalCost: 1500, ExecutedAt: time.Now()},
			{Ticker: "MSFT", Type: "SELL", Quantity: 5, UnitPrice: 200, TotalCost: 1000, ExecutedAt: time.Now()},
		}
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", uID.String()).Return(txs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success pagination", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("1") // page 1
		mCtx.On("Message").Return(&telebot.Message{Text: "Test"})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		txs := make([]portfolio.Transaction, 15) // > 10
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", uID.String()).Return(txs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleFixedIncome(t *testing.T) {
	h, svc, pSvc, _, fiSvc := setupHandlersTest()

	t.Run("not active", func(t *testing.T) {
		hNoFi, _, _, _, _ := setupHandlersTest()
		hNoFi.fiSvc = nil
		mCtx := new(MockTelebotContext)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := hNoFi.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("err fetch positions", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(([]fixedincome.Position)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no positions", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		pos := []fixedincome.Position{
			{NetValue: 100, GrossValue: 110, TotalInvested: 95, NetReturnPercent: 5.2, IsMatured: true, DaysToMaturity: 0, Asset: fixedincome.Asset{DebtType: "POS", Rate: 100, Indexer: "CDI"}},
			{NetValue: 50, GrossValue: 55, TotalInvested: 40, NetReturnPercent: 25, IsMatured: false, DaysToMaturity: 15, Asset: fixedincome.Asset{DebtType: "PRE", Rate: 10, Indexer: ""}},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(pos, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})
}
