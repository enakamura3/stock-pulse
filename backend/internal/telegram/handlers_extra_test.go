package telegram

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_ExtraCoverage(t *testing.T) {
	h, svc, pSvc, _, fiSvc := setupHandlersTest()

	t.Run("HandlePortfolioSummary flat daily change", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		
		// 0% daily change
		positions := []portfolio.Position{
			{Ticker: "FLAT", CurrentValue: 100, TotalCost: 100, DailyChange: 0, DailyChangePercent: 0, Quantity: 1, CurrentPrice: 100},
		}
		
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", uID.String()).Return(&portfolio.Portfolio{}, positions, nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()
		
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)
		
		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividends sorting and types", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		
		now := time.Now()
		
		// Unsorted past and future dividends to trigger sort branches
		divs := []portfolio.CalculatedDividend{
			{NetAmount: 10, PaymentDate: now.AddDate(0, 0, -5), Ticker: "B", Type: "JCP"},
			{NetAmount: 15, PaymentDate: now.AddDate(0, 0, -10), Ticker: "A", Type: "DIVIDENDO"},
			{NetAmount: 20, PaymentDate: now.AddDate(0, 0, 5), Ticker: "Z", Type: ""},
			{NetAmount: 25, PaymentDate: now.AddDate(0, 0, 10), Ticker: "Y", Type: "RENDIMENTO"},
		}
		
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(divs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		
		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth same date deduplication and end pagination", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("1") // Page 1
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		
		// 15 months to trigger keys > 12 -> page 1 has end < len(keys)
		var divs []portfolio.CalculatedDividend
		for i := 1; i <= 15; i++ {
			d1 := time.Date(2023, time.Month(i%12+1), 1, 0, 0, 0, 0, time.UTC)
			divs = append(divs, portfolio.CalculatedDividend{
				NetAmount: 10, PaymentDate: d1, Ticker: "AAPL", Type: "JCP",
			})
			// Add duplicate date for the same month/ticker to hit foundDate = true
			divs = append(divs, portfolio.CalculatedDividend{
				NetAmount: 5, PaymentDate: d1, Ticker: "AAPL", Type: "JCP",
			})
		}
		
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(divs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Message").Return(&telebot.Message{Text: "Test"})
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		
		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory end pagination", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("1") // Page 1
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		
		// 25 txs to trigger txs > 20 -> page 1 has end < len(txs)
		txs := make([]portfolio.Transaction, 25)
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", uID.String()).Return(txs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Message").Return(&telebot.Message{Text: "Test"})
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		
		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})
}
