package telegram

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_HandleMenu(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})

	t.Run("1 portfolio", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})

	t.Run("2 portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}, {ID: "p2", Name: "P2"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p1", nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HandleChangePortfolio(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleChangePortfolio(mCtx)
		assert.NoError(t, err)
	})

	t.Run("no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleChangePortfolio(mCtx)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.HandleChangePortfolio(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_handleSelectedPortfolio(t *testing.T) {
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("not linked", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})

	t.Run("err fetch", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{}, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})

	t.Run("invalid portfolio", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p2", Name: "P2"}}, nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})

	t.Run("err set active", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("SetActivePortfolio", mock.Anything, int64(1), "p1").Return(errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uID, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("SetActivePortfolio", mock.Anything, int64(1), "p1").Return(nil).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})
}
