package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_getCurrencySymbol(t *testing.T) {
	assert.Equal(t, "US$", getCurrencySymbol("USD"))
	assert.Equal(t, "€", getCurrencySymbol("EUR"))
	assert.Equal(t, "R$", getCurrencySymbol("BRL"))
	assert.Equal(t, "R$", getCurrencySymbol("OTHER"))
}

func TestHandlers_abbreviateDividendType(t *testing.T) {
	assert.Equal(t, "DIV", abbreviateDividendType("DIVIDENDO"))
	assert.Equal(t, "JCP", abbreviateDividendType("JUROS SOBRE CAPITAL PRÓPRIO"))
	assert.Equal(t, "REND", abbreviateDividendType("RENDIMENTO"))
	assert.Equal(t, "AMORT", abbreviateDividendType("AMORTIZAÇÃO"))
	assert.Equal(t, "TEST", abbreviateDividendType("TESTING"))
	assert.Equal(t, "ABC", abbreviateDividendType("abc"))
}

func TestHandlers_resolveActivePortfolio(t *testing.T) {
	h, svc, _, _, _ := setupHandlersTest()

	// 0 portfolios
	id, name := h.resolveActivePortfolio(context.Background(), 1, []portfolio.Portfolio{})
	assert.Empty(t, id)
	assert.Empty(t, name)

	// active portfolio found
	svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("p2", nil).Once()
	id, name = h.resolveActivePortfolio(context.Background(), 1, []portfolio.Portfolio{
		{ID: "p1", Name: "P1"},
		{ID: "p2", Name: "P2"},
	})
	assert.Equal(t, "p2", id)
	assert.Equal(t, "P2", name)

	// active portfolio not found, fallback to 0
	svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("px", nil).Once()
	id, name = h.resolveActivePortfolio(context.Background(), 1, []portfolio.Portfolio{
		{ID: "p1", Name: "P1"},
		{ID: "p2", Name: "P2"},
	})
	assert.Equal(t, "p1", id)
	assert.Equal(t, "P1", name)

	// GetActivePortfolio error
	svc.On("GetActivePortfolio", mock.Anything, int64(1)).Return("", errors.New("err")).Once()
	id, name = h.resolveActivePortfolio(context.Background(), 1, []portfolio.Portfolio{
		{ID: "p1", Name: "P1"},
	})
	assert.Equal(t, "p1", id)
	assert.Equal(t, "P1", name)
}

func TestHandlers_HandleDynamicCallback(t *testing.T) {
	h, svc, _, _, _ := setupHandlersTest()

	t.Run("btn_ticker_", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fbtn_ticker_AAPL"})
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})

		svc.On("GetConversationState", mock.Anything, int64(1)).Return((*ConversationState)(nil), errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})

	t.Run("btn_sel_port_", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fbtn_sel_port_p1"})
		mCtx.On("Chat").Return(&telebot.Chat{ID: 1})

		svc.On("GetUserIDByChatID", mock.Anything, int64(1)).Return(uuid.Nil, errors.New("err")).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})

	t.Run("other", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fother"})

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})
}
