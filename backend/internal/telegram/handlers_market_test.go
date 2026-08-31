package telegram

import (
	"errors"
	"strings"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_Market(t *testing.T) {
	h, svc, _, mSvc, _, _ := setupHandlersTest()

	t.Run("HandleQuote - with args success and all fields formatted", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{"vale3.sa"}).Once()
		mSvc.On("GetQuote", mock.Anything, "VALE3.SA").Return(&market.Quote{
			Symbol:        "VALE3.SA",
			Name:          "Vale S.A.",
			Price:         62.50,
			Change:        1.25,
			ChangePercent: 2.04,
			High:          63.00,
			Low:           61.50,
			PreviousClose: 61.25,
			Volume:        1500000,
			Currency:      "BRL",
		}, nil).Once()

		var sentMsg string
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "VALE3.SA") &&
				strings.Contains(msg, "Vale S.A.") &&
				strings.Contains(msg, "Preço:") &&
				strings.Contains(msg, "R$ 62,50") &&
				strings.Contains(msg, "1.500.000")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "VALE3.SA")
	})

	t.Run("HandleQuote - with args quote error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{"INVALID"}).Once()
		mSvc.On("GetQuote", mock.Anything, "INVALID").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mCtx.On("Send", "⚠️ Ativo não encontrado. Verifique o código e tente novamente.", mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleQuote - without args delegates to HandleQuoteStart", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Args").Return([]string{}).Once()
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "QUOTE_EXPECT_TICKER"}).Return(nil).Once()
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Cotação Rápida")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleQuoteStart - callback with state error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{}).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "QUOTE_EXPECT_TICKER"}).Return(errors.New("redis error")).Once()
		mCtx.On("Edit", "❌ Erro interno ao iniciar consulta de cotação.", mock.Anything).Return(nil).Once()

		err := h.HandleQuoteStart(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleQuoteStart - message with state error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil)).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "QUOTE_EXPECT_TICKER"}).Return(errors.New("redis error")).Once()
		mCtx.On("Send", "❌ Erro interno ao iniciar consulta de cotação.", mock.Anything).Return(nil).Once()

		err := h.HandleQuoteStart(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleQuoteStart - callback success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{}).Twice()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "QUOTE_EXPECT_TICKER"}).Return(nil).Once()
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Cotação Rápida")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleQuoteStart(mCtx)
		assert.NoError(t, err)
	})

	t.Run("formatQuoteMessage - negative change and zero/fallback fields", func(t *testing.T) {
		quote := &market.Quote{
			Symbol:        "", // fallback to ticker
			Name:          "Apple Inc.",
			Price:         180.00,
			Change:        -2.50,
			ChangePercent: -1.37,
			Currency:      "USD",
		}
		msg := formatQuoteMessage("AAPL", quote)
		assert.Contains(t, msg, "AAPL")
		assert.Contains(t, msg, "🔴 *Variação:* -2,50 (-1,37%)")
		assert.Contains(t, msg, "US$ 180,00")
		assert.NotContains(t, msg, "Volume:")
		assert.NotContains(t, msg, "Mín / Máx")
		assert.NotContains(t, msg, "Fechamento Anterior")
	})

	t.Run("formatQuoteMessage - zero change", func(t *testing.T) {
		quote := &market.Quote{
			Symbol:        "NEUT3",
			Name:          "Neutral Corp",
			Price:         10.00,
			Change:        0.0,
			ChangePercent: 0.0,
			Currency:      "BRL",
		}
		msg := formatQuoteMessage("NEUT3", quote)
		assert.Contains(t, msg, "⚪ *Variação:* 0,00 (0,00%)")
	})

	t.Run("HandleText - QUOTE_EXPECT_TICKER error and success", func(t *testing.T) {
		// Error
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxErr.On("Text").Return("INVALID")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "QUOTE_EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "INVALID").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mCtxErr.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo não encontrado na bolsa")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtxErr)
		assert.NoError(t, err)

		// Success
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Text").Return("petr4")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "QUOTE_EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{
			Symbol:        "PETR4.SA",
			Name:          "Petrobras",
			Price:         38.20,
			Change:        0.50,
			ChangePercent: 1.32,
			Currency:      "BRL",
		}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mCtxOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4.SA") &&
				strings.Contains(msg, "Petrobras") &&
				strings.Contains(msg, "R$ 38,20")
		}), mock.Anything).Return(nil).Once()

		err = h.HandleText(mCtxOk)
		assert.NoError(t, err)
	})
}
