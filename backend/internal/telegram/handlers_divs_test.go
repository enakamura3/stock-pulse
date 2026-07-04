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

func TestHandlers_fetchDividends(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()

		_, _, err := h.fetchDividends(mCtx)
		assert.ErrorContains(t, err, "conta não vinculada")
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()

		_, _, err := h.fetchDividends(mCtx)
		assert.ErrorContains(t, err, "nenhuma carteira")
	})

	t.Run("err fetch", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(([]portfolio.CalculatedDividend)(nil), errors.New("err")).Once()

		_, _, err := h.fetchDividends(mCtx)
		assert.ErrorContains(t, err, "erro ao buscar")
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return([]portfolio.CalculatedDividend{{NetAmount: 10}}, nil).Once()

		divs, name, err := h.fetchDividends(mCtx)
		assert.NoError(t, err)
		assert.Equal(t, "P1", name)
		assert.Len(t, divs, 1)
	})
}

func TestHandlers_HandleDividends(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("err fetch divs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(([]portfolio.CalculatedDividend)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		now := time.Now()
		past := now.Add(-2 * time.Hour)
		future := now.Add(2 * time.Hour)

		divs := []portfolio.CalculatedDividend{
			{NetAmount: 10, PaymentDate: past, Ticker: "AAPL", Type: "DIVIDENDO"},
			{NetAmount: 20, PaymentDate: future, Ticker: "MSFT", Type: ""}, // Empty type test
		}

		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(divs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Message").Return((*telebot.Message)(nil))
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no divs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return([]portfolio.CalculatedDividend{}, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleDividendsByYear(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("err fetch divs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividendsByYear(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		divs := []portfolio.CalculatedDividend{
			{NetAmount: 10, PaymentDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			{NetAmount: 20, PaymentDate: time.Time{}}, // A Definir (year <= 1)
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(divs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividendsByYear(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleDividendsByMonth(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("err fetch divs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no divs", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("") // pageStr
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return([]portfolio.CalculatedDividend{}, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("0") // pageStr
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		divs := []portfolio.CalculatedDividend{
			{NetAmount: 10, PaymentDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Ticker: "AAPL"},
			{NetAmount: 20, PaymentDate: time.Time{}, Ticker: "MSFT"}, // 0000-00
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(divs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Message").Return((*telebot.Message)(nil))
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success pagination", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		mCtx.On("Data").Return("1") // page 1
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()

		// 4 different months to trigger pagination
		divs := []portfolio.CalculatedDividend{
			{NetAmount: 10, PaymentDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			{NetAmount: 10, PaymentDate: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)},
			{NetAmount: 10, PaymentDate: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)},
			{NetAmount: 10, PaymentDate: time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC)},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", uID.String()).Return(divs, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Message").Return((*telebot.Message)(nil))
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})
}
