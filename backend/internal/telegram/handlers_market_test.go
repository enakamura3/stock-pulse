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

	t.Run("renderQuote - callback branches and error handling", func(t *testing.T) {
		// Callback error with "message is not modified"
		mCtx1 := new(MockTelebotContext)
		mCtx1.On("Callback").Return(&telebot.Callback{}).Twice()
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{
			Symbol: "VALE3", Price: 60.0, Currency: "BRL",
		}, nil).Once()
		mCtx1.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()
		err := h.renderQuote(mCtx1, "VALE3")
		assert.NoError(t, err)

		// Callback error with other error
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Callback").Return(&telebot.Callback{}).Twice()
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{
			Symbol: "VALE3", Price: 60.0, Currency: "BRL",
		}, nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("fatal edit error")).Once()
		err = h.renderQuote(mCtx2, "VALE3")
		assert.Error(t, err)

		// Callback with quote error
		mCtx3 := new(MockTelebotContext)
		mCtx3.On("Callback").Return(&telebot.Callback{}).Once()
		mSvc.On("GetQuote", mock.Anything, "ERR").Return((*market.Quote)(nil), errors.New("err")).Once()
		mCtx3.On("Edit", "⚠️ Ativo não encontrado. Verifique o código e tente novamente.", mock.Anything).Return(nil).Once()
		err = h.renderQuote(mCtx3, "ERR")
		assert.NoError(t, err)
	})

	t.Run("HandleAnalysis - with and without args", func(t *testing.T) {
		// with args
		mCtxWithArgs := new(MockTelebotContext)
		mCtxWithArgs.On("Args").Return([]string{"petr4"}).Once()
		mSvc.On("GetFundamentals", mock.Anything, "PETR4").Return(&market.Fundamentals{
			Symbol: "PETR4", EPS: 5.0, BookValue: 30.0, DividendYield: 8.5, GrahamValue: 40.0, BazinValue: 45.0,
		}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Price: 35.0, Currency: "BRL"}, nil).Once()
		mCtxWithArgs.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Análise Fundamentalista: PETR4")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAnalysis(mCtxWithArgs)
		assert.NoError(t, err)

		// without args delegates to HandleAnalysisStart
		mCtxNoArgs := new(MockTelebotContext)
		mCtxNoArgs.On("Args").Return([]string{""}).Once()
		mCtxNoArgs.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ANALYSIS_EXPECT_TICKER"}).Return(nil).Once()
		mCtxNoArgs.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Análise Fundamentalista")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleAnalysis(mCtxNoArgs)
		assert.NoError(t, err)
	})

	t.Run("HandleAnalysisStart - variations", func(t *testing.T) {
		// Callback with error
		mCtx1 := new(MockTelebotContext)
		mCtx1.On("Callback").Return(&telebot.Callback{}).Twice()
		mCtx1.On("Respond", mock.Anything).Return(nil).Once()
		mCtx1.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ANALYSIS_EXPECT_TICKER"}).Return(errors.New("err")).Once()
		mCtx1.On("Edit", "❌ Erro interno ao iniciar análise fundamentalista.", mock.Anything).Return(nil).Once()
		err := h.HandleAnalysisStart(mCtx1)
		assert.NoError(t, err)

		// Message with error
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ANALYSIS_EXPECT_TICKER"}).Return(errors.New("err")).Once()
		mCtx2.On("Send", "❌ Erro interno ao iniciar análise fundamentalista.", mock.Anything).Return(nil).Once()
		err = h.HandleAnalysisStart(mCtx2)
		assert.NoError(t, err)

		// Callback success
		mCtx3 := new(MockTelebotContext)
		mCtx3.On("Callback").Return(&telebot.Callback{}).Twice()
		mCtx3.On("Respond", mock.Anything).Return(nil).Once()
		mCtx3.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ANALYSIS_EXPECT_TICKER"}).Return(nil).Once()
		mCtx3.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Análise Fundamentalista")
		}), mock.Anything, mock.Anything).Return(nil).Once()
		err = h.HandleAnalysisStart(mCtx3)
		assert.NoError(t, err)
	})

	t.Run("renderAnalysis - variations and branches", func(t *testing.T) {
		// Fundamentals error non-callback
		mCtx1 := new(MockTelebotContext)
		mSvc.On("GetFundamentals", mock.Anything, "FAIL").Return((*market.Fundamentals)(nil), errors.New("not found")).Once()
		mCtx1.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Dados fundamentalistas não encontrados")
		}), mock.Anything).Return(nil).Once()
		err := h.renderAnalysis(mCtx1, "FAIL")
		assert.NoError(t, err)

		// Fundamentals error callback
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Callback").Return(&telebot.Callback{}).Once()
		mSvc.On("GetFundamentals", mock.Anything, "FAIL").Return((*market.Fundamentals)(nil), nil).Once()
		mCtx2.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Dados fundamentalistas não encontrados")
		}), mock.Anything).Return(nil).Once()
		err = h.renderAnalysis(mCtx2, "FAIL")
		assert.NoError(t, err)

		// Success callback with "message is not modified"
		mCtx3 := new(MockTelebotContext)
		mCtx3.On("Callback").Return(&telebot.Callback{}).Twice()
		mSvc.On("GetFundamentals", mock.Anything, "BBAS3").Return(&market.Fundamentals{
			Symbol: "BBAS3", EPS: 6.0, BookValue: 40.0, DividendYield: 10.0, GrahamValue: 50.0, BazinValue: 45.0,
		}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "BBAS3").Return((*market.Quote)(nil), errors.New("quote fail")).Once()
		mCtx3.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("message is not modified")).Once()
		err = h.renderAnalysis(mCtx3, "BBAS3")
		assert.NoError(t, err)

		// Success callback with other error
		mCtx4 := new(MockTelebotContext)
		mCtx4.On("Callback").Return(&telebot.Callback{}).Twice()
		mSvc.On("GetFundamentals", mock.Anything, "BBAS3").Return(&market.Fundamentals{
			Symbol: "BBAS3", EPS: 6.0, BookValue: 40.0, DividendYield: 10.0, GrahamValue: 50.0, BazinValue: 45.0,
		}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "BBAS3").Return(&market.Quote{Price: 28.0, Currency: "BRL"}, nil).Once()
		mCtx4.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("other edit error")).Once()
		err = h.renderAnalysis(mCtx4, "BBAS3")
		assert.Error(t, err)
	})

	t.Run("formatAnalysisMessage - valuation margins and edge cases", func(t *testing.T) {
		// Negative margin (price > Graham & Bazin) and negative EPS
		fundOver := &market.Fundamentals{
			Symbol:        "OVER3",
			EPS:           -1.5,
			BookValue:     10.0,
			DividendYield: 2.0,
			GrahamValue:   15.0,
			BazinValue:    12.0,
		}
		msg := formatAnalysisMessage("OVER3", fundOver, 30.0, "BRL")
		assert.Contains(t, msg, "Preço Atual: *R$ 30,00*")
		assert.Contains(t, msg, "• P/L: _N/D_")
		assert.Contains(t, msg, "• P/VP: *3,00*")
		assert.Contains(t, msg, "• LPA (Lucro/Ação): *R$ -1,50*")
		assert.Contains(t, msg, "• VPA (Patrimônio/Ação): *R$ 10,00*")
		assert.Contains(t, msg, "🔴 -50,0% do teto")
		assert.Contains(t, msg, "🔴 -60,0% do teto")

		// All zero / N/D
		fundZero := &market.Fundamentals{
			Symbol: "ZERO3",
		}
		msgZero := formatAnalysisMessage("ZERO3", fundZero, 0, "")
		assert.NotContains(t, msgZero, "Preço Atual:")
		assert.Contains(t, msgZero, "• P/L: _N/D_")
		assert.Contains(t, msgZero, "• P/VP: _N/D_")
		assert.Contains(t, msgZero, "• Preço Justo de Graham: _N/D_")
		assert.Contains(t, msgZero, "• Preço Teto Bazin (6%): _N/D_")
	})

	t.Run("HandleText - ANALYSIS_EXPECT_TICKER", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("itub4")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ANALYSIS_EXPECT_TICKER"}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mSvc.On("GetFundamentals", mock.Anything, "ITUB4").Return(&market.Fundamentals{
			Symbol: "ITUB4", EPS: 3.5, BookValue: 20.0, DividendYield: 6.5, GrahamValue: 35.0, BazinValue: 32.0,
		}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "ITUB4").Return(&market.Quote{Price: 30.0, Currency: "BRL"}, nil).Once()
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Análise Fundamentalista: ITUB4")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})
}
