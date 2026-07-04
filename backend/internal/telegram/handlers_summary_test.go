package telegram

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_HandlePortfolioSummary(t *testing.T) {
	h, svc, pSvc, _, fiSvc := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("err fetch details", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "AAPL", CurrentValue: 100, TotalCost: 90, DailyChange: 2, DailyChangePercent: 2, Quantity: 1, CurrentPrice: 100},
			{Ticker: "MSFT", CurrentValue: 50, TotalCost: 60, DailyChange: -5, DailyChangePercent: -10, Quantity: 1, CurrentPrice: 50},
			{Ticker: "FLAT", CurrentValue: 10, TotalCost: 10, DailyChange: 0, DailyChangePercent: 0, Quantity: 1, CurrentPrice: 10},
		}

		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return(&portfolio.Portfolio{}, positions, nil).Once()

		fiPositions := []fixedincome.Position{
			{NetValue: 100, TotalInvested: 95, DaysToMaturity: 15, IsMatured: false, Asset: fixedincome.Asset{Institution: "Bank", Type: "CDB"}},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPositions, nil).Once()

		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success negative global", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "AAPL", CurrentValue: 90, TotalCost: 100, DailyChange: -2, DailyChangePercent: -2, Quantity: 1, CurrentPrice: 90},
			{Ticker: "MSFT", CurrentValue: 50, TotalCost: 60, DailyChange: -5, DailyChangePercent: -10, Quantity: 1, CurrentPrice: 50},
		}

		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return(&portfolio.Portfolio{}, positions, nil).Once()

		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()

		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success positive global", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		
		positions := []portfolio.Position{
			{Ticker: "AAPL", CurrentValue: 120, TotalCost: 100, DailyChange: 20, DailyChangePercent: 20, Quantity: 1, CurrentPrice: 120},
		}
		
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return(&portfolio.Portfolio{}, positions, nil).Once()
		
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, errors.New("none")).Once()
		
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		
		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})
}
