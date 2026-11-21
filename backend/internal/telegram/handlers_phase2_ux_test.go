package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/onigiri/stock-pulse/backend/internal/watchlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestPhase2_TickerHelpers(t *testing.T) {
	t.Run("isCryptoPairTicker", func(t *testing.T) {
		assert.True(t, isCryptoPairTicker("BTC-USD"))
		assert.True(t, isCryptoPairTicker("eth-brl"))
		assert.True(t, isCryptoPairTicker("1INCH-USDT"))

		assert.False(t, isCryptoPairTicker("BTCUSD"))
		assert.False(t, isCryptoPairTicker("A-USD"))
		assert.False(t, isCryptoPairTicker("ABCDEFGHIJK-USD"))
		assert.False(t, isCryptoPairTicker("BTC-US"))
		assert.False(t, isCryptoPairTicker("BTC-USDTX"))
		assert.False(t, isCryptoPairTicker("BT$C-USD"))
		assert.False(t, isCryptoPairTicker("BTC-US1"))
	})

	t.Run("parseExplicitDollarTicker", func(t *testing.T) {
		clean, ok := parseExplicitDollarTicker("$aapl")
		assert.True(t, ok)
		assert.Equal(t, "AAPL", clean)

		clean, ok = parseExplicitDollarTicker("$BRK.B")
		assert.True(t, ok)
		assert.Equal(t, "BRK.B", clean)

		clean, ok = parseExplicitDollarTicker("$BTC-USD")
		assert.True(t, ok)
		assert.Equal(t, "BTC-USD", clean)

		_, ok = parseExplicitDollarTicker("AAPL")
		assert.False(t, ok)
		_, ok = parseExplicitDollarTicker("$")
		assert.False(t, ok)
		_, ok = parseExplicitDollarTicker("$ABCDEFGHIJKLM")
		assert.False(t, ok)
		_, ok = parseExplicitDollarTicker("$AA!L")
		assert.False(t, ok)
		_, ok = parseExplicitDollarTicker("$1234")
		assert.False(t, ok)
	})

	t.Run("isPlainTickerCandidate", func(t *testing.T) {
		assert.True(t, isPlainTickerCandidate("AAPL"))
		assert.True(t, isPlainTickerCandidate("BRK.B"))
		assert.True(t, isPlainTickerCandidate("voo"))

		assert.False(t, isPlainTickerCandidate(""))
		assert.False(t, isPlainTickerCandidate("ABCDEFGHIJK"))
		assert.False(t, isPlainTickerCandidate("ola?"))
		assert.False(t, isPlainTickerCandidate("1234"))
	})

	t.Run("tickersMatch", func(t *testing.T) {
		assert.True(t, tickersMatch("PETR4", "petr4.sa"))
		assert.True(t, tickersMatch(" AAPL ", "aapl"))
		assert.False(t, tickersMatch("PETR4", "VALE3"))
	})
}

func TestPhase2_ZeroClickQuote(t *testing.T) {
	t.Run("HandleText with B3 single ticker opens quote directly", func(t *testing.T) {
		h, svc, _, mSvc, _, _ := setupHandlersTest()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("petr4")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{
			Ticker: "PETR4.SA", Name: "Petrobras", Price: 37.50, Currency: "BRL",
		}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4") && strings.Contains(msg, "R$ 37,50")
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleText(mCtx))
	})

	t.Run("HandleText with multiple tickers ($AAPL, BTC-USD, VALE3) opens multi-quote", func(t *testing.T) {
		h, svc, _, mSvc, _, _ := setupHandlersTest()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("vale3, $aapl; btc-usd")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{Ticker: "VALE3.SA", Price: 60.0, Currency: "BRL"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Ticker: "AAPL", Price: 200.0, Currency: "USD"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "BTC-USD").Return(&market.Quote{Ticker: "BTC-USD", Price: 90000.0, Currency: "USD"}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Painel de Cotações") &&
				strings.Contains(msg, "VALE3") &&
				strings.Contains(msg, "AAPL") &&
				strings.Contains(msg, "BTC-USD")
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleText(mCtx))
	})

	t.Run("HandleText resolves plain US ticker when present in active portfolio or watchlist", func(t *testing.T) {
		h, svc, pSvc, mSvc, _, _, wlSvc := setupHandlersTestWithWatchlist()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("aapl nvda")
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "Global"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{
			{Ticker: "AAPL", Quantity: 5},
		}, nil).Once()

		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID:    "wl-1",
			Items: []watchlist.Item{{Ticker: "NVDA"}},
		}, nil).Once()

		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Ticker: "AAPL", Price: 210.0, Currency: "USD"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "NVDA").Return(&market.Quote{Ticker: "NVDA", Price: 130.0, Currency: "USD"}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Painel de Cotações") && strings.Contains(msg, "AAPL") && strings.Contains(msg, "NVDA")
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleText(mCtx))
	})

	t.Run("HandleText ignores Portuguese words, punctuation, commands, and >10 tokens", func(t *testing.T) {
		h, svc, _, _, _, _ := setupHandlersTest()
		cases := []string{
			"bom dia",
			"oi tudo bem?",
			"/desconhecido",
			"PETR4 VALE3 ITUB4 BBAS3 WEGE3 ABEV3 MGLU3 B3SA3 RENT3 SUZB3 JBSS3",
			" , ; ",
		}
		for _, txt := range cases {
			mCtx := new(MockTelebotContext)
			mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
			mCtx.On("Text").Return(txt)
			svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
			mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
				return strings.Contains(msg, "Não entendi esse comando ou mensagem solta")
			}), mock.Anything).Return(nil).Once()

			assert.NoError(t, h.HandleText(mCtx))
		}
	})

	t.Run("collectUserKnownTickers nil context and invalid user_id", func(t *testing.T) {
		h, _, _, _, _, _ := setupHandlersTest()
		assert.Empty(t, h.collectUserKnownTickers(context.Background(), nil))

		mCtxInvalid := new(MockTelebotContext)
		mCtxInvalid.Set("user_id", "")
		assert.Empty(t, h.collectUserKnownTickers(context.Background(), mCtxInvalid))
	})
}

func TestPhase2_QuickPortfolioAssetButtons(t *testing.T) {
	t.Run("buildPortfolioQuickAssetRows edge cases, sorting, cap at 10, odd row, and >63B skip", func(t *testing.T) {
		hNil := &Handlers{}
		assert.Nil(t, hNil.buildPortfolioQuickAssetRows(context.Background(), nil, &telebot.ReplyMarkup{}, "btn_quote_", 10))

		h, svc, pSvc, _, _, _ := setupHandlersTest()
		mCtxBadUser := new(MockTelebotContext)
		mCtxBadUser.Set("user_id", "")
		assert.Nil(t, h.buildPortfolioQuickAssetRows(context.Background(), mCtxBadUser, &telebot.ReplyMarkup{}, "btn_quote_", 10))

		// Empty positions
		mCtxEmpty := new(MockTelebotContext)
		mCtxEmpty.On("Chat").Return(&telebot.Chat{ID: 123})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{}, nil).Once()
		assert.Nil(t, h.buildPortfolioQuickAssetRows(context.Background(), mCtxEmpty, &telebot.ReplyMarkup{}, "btn_quote_", 10))

		// 12 positions (cap to 10) + 1 ticker exceeding 63B callback_data (skipped -> 9 buttons = 5 rows with odd last row)
		var positions []portfolio.Position
		positions = append(positions, portfolio.Position{Ticker: strings.Repeat("Z", 60), CurrentValue: 999999})
		for i := 1; i <= 11; i++ {
			positions = append(positions, portfolio.Position{Ticker: fmt.Sprintf("T%02d.SA", i), CurrentValue: float64(i * 100)})
		}
		mCtxCap := new(MockTelebotContext)
		mCtxCap.On("Chat").Return(&telebot.Chat{ID: 123})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, positions, nil).Once()

		rows := h.buildPortfolioQuickAssetRows(context.Background(), mCtxCap, &telebot.ReplyMarkup{}, "btn_quote_", 10)
		assert.Equal(t, 5, len(rows))
		assert.Equal(t, 2, len(rows[0]))
		assert.Equal(t, 1, len(rows[4]))
		assert.Equal(t, "T11", rows[0][0].Text)
	})

	t.Run("HandleQuoteStart, HandleAnalysisStart and HandleAlertCreate render quick portfolio buttons", func(t *testing.T) {
		h, svc, pSvc, _, _, _ := setupHandlersTest()
		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}
		positions := []portfolio.Position{
			{Ticker: "PETR4.SA", CurrentValue: 5000},
			{Ticker: "VALE3.SA", CurrentValue: 3000},
		}

		// HandleQuoteStart
		mCtxQ := new(MockTelebotContext)
		mCtxQ.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxQ.On("Callback").Return(&telebot.Callback{})
		mCtxQ.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "QUOTE_EXPECT_TICKER"}).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()
		var qMenu *telebot.ReplyMarkup
		mCtxQ.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Escolha um ativo da sua carteira abaixo")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					qMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleQuoteStart(mCtxQ))
		assert.Equal(t, "btn_quote_PETR4.SA", qMenu.InlineKeyboard[0][0].Unique)

		// HandleAnalysisStart
		mCtxA := new(MockTelebotContext)
		mCtxA.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxA.On("Callback").Return(&telebot.Callback{})
		mCtxA.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ANALYSIS_EXPECT_TICKER"}).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()
		var aMenu *telebot.ReplyMarkup
		mCtxA.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Escolha um ativo da sua carteira abaixo")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					aMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleAnalysisStart(mCtxA))
		assert.Equal(t, "btn_analise_PETR4.SA", aMenu.InlineKeyboard[0][0].Unique)

		// HandleAlertCreate
		mCtxAl := new(MockTelebotContext)
		mCtxAl.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAl.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "ALERT_EXPECT_TICKER"}).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()
		var alMenu *telebot.ReplyMarkup
		mCtxAl.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Escolha um ativo da sua carteira abaixo")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					alMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleAlertCreate(mCtxAl))
		assert.Equal(t, "btn_alert_create_PETR4.SA", alMenu.InlineKeyboard[0][0].Unique)
	})
}

func TestPhase2_SmartWatchlistToggle(t *testing.T) {
	t.Run("findWatchlistItem edge cases", func(t *testing.T) {
		hNil := &Handlers{}
		wlID, stored, found := hNil.findWatchlistItem(context.Background(), nil, "PETR4")
		assert.Empty(t, wlID)
		assert.Empty(t, stored)
		assert.False(t, found)

		h, _, _, _, _, _, wlSvc := setupHandlersTestWithWatchlist()
		mCtxBadUser := new(MockTelebotContext)
		mCtxBadUser.Set("user_id", "")
		wlID, _, found = h.findWatchlistItem(context.Background(), mCtxBadUser, "PETR4")
		assert.Empty(t, wlID)
		assert.False(t, found)

		// GetWatchlist error still returns wlID
		mCtxWlErr := new(MockTelebotContext)
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return((*watchlist.Watchlist)(nil), errors.New("err")).Once()
		wlID, stored, found = h.findWatchlistItem(context.Background(), mCtxWlErr, "PETR4")
		assert.Equal(t, "wl-1", wlID)
		assert.Empty(t, stored)
		assert.False(t, found)

		// Found with .SA normalization
		mCtxFound := new(MockTelebotContext)
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID:    "wl-1",
			Items: []watchlist.Item{{Ticker: "PETR4.SA"}},
		}, nil).Once()
		wlID, stored, found = h.findWatchlistItem(context.Background(), mCtxFound, "petr4")
		assert.Equal(t, "wl-1", wlID)
		assert.Equal(t, "PETR4.SA", stored)
		assert.True(t, found)
	})

	t.Run("handleQuickWatchlistToggle errors, add with B3 fallback on Quote, and remove on Analysis", func(t *testing.T) {
		// 1. watchlistSvc nil
		hNil := &Handlers{watchlistSvc: nil}
		mCtxNil := new(MockTelebotContext)
		mCtxNil.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && strings.Contains(resps[0].Text, "não está ativo")
		})).Return(nil).Once()
		assert.NoError(t, hNil.handleQuickWatchlistToggle(mCtxNil, "q", "PETR4"))

		// 2. getUserID error
		h, svc, _, mSvc, _, _, wlSvc := setupHandlersTestWithWatchlist()
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.store = map[string]interface{}{"user_id": 123}
		mCtxUserErr.On("Send", mock.Anything, mock.Anything).Return(nil).Once()
		assert.Error(t, h.handleQuickWatchlistToggle(mCtxUserErr, "q", "PETR4"))

		// 3. No watchlist found
		mCtxNoWl := new(MockTelebotContext)
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{}, nil).Once()
		mCtxNoWl.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && strings.Contains(resps[0].Text, "Erro ao buscar lista")
		})).Return(nil).Once()
		assert.NoError(t, h.handleQuickWatchlistToggle(mCtxNoWl, "q", "PETR4"))

		// 4. Add error
		mCtxAddErr := new(MockTelebotContext)
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{ID: "wl-1"}, nil).Once()
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "INVALID").Return(nil, errors.New("fail")).Once()
		mCtxAddErr.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && strings.Contains(resps[0].Text, "Não foi possível adicionar")
		})).Return(nil).Once()
		assert.NoError(t, h.handleQuickWatchlistToggle(mCtxAddErr, "q", "INVALID"))

		// 5. Remove error
		mCtxRmErr := new(MockTelebotContext)
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID: "wl-1", Items: []watchlist.Item{{Ticker: "PETR4.SA"}},
		}, nil).Once()
		wlSvc.On("RemoveAssetFromWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "PETR4.SA").Return(errors.New("db err")).Once()
		mCtxRmErr.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && strings.Contains(resps[0].Text, "Erro ao remover")
		})).Return(nil).Once()
		assert.NoError(t, h.handleQuickWatchlistToggle(mCtxRmErr, "q", "PETR4"))

		// 6. Add success with B3 .SA fallback via DynamicCallback (btn_wl_tog_q_PETR4) -> re-renders Quote with Remover Favorito
		mCtxAddOk := new(MockTelebotContext)
		mCtxAddOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAddOk.On("Callback").Return(&telebot.Callback{Data: "\fbtn_wl_tog_q_PETR4"})
		// First check (toggle): not in watchlist
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{ID: "wl-1"}, nil).Once()
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "PETR4").Return(nil, errors.New("try sa")).Once()
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "PETR4.SA").Return(&watchlist.Item{Ticker: "PETR4.SA"}, nil).Once()
		mCtxAddOk.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && strings.Contains(resps[0].Text, "Adicionado aos Favoritos")
		})).Return(nil).Once()
		// Re-render quote card: now present in watchlist
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Price: 36.0, Currency: "BRL"}, nil).Once()
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID: "wl-1", Items: []watchlist.Item{{Ticker: "PETR4.SA"}},
		}, nil).Once()
		var updatedQuoteMenu *telebot.ReplyMarkup
		mCtxAddOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					updatedQuoteMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleDynamicCallback(mCtxAddOk))
		assert.Equal(t, "💔 Remover Favorito", updatedQuoteMenu.InlineKeyboard[2][0].Text)

		// 7. Remove success via DynamicCallback (btn_wl_tog_a_PETR4) -> re-renders Analysis with ⭐ Favoritar
		mCtxRmOk := new(MockTelebotContext)
		mCtxRmOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxRmOk.On("Callback").Return(&telebot.Callback{Data: "\fbtn_wl_tog_a_PETR4"})
		// First check (toggle): found in watchlist
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID: "wl-1", Items: []watchlist.Item{{Ticker: "PETR4.SA"}},
		}, nil).Once()
		wlSvc.On("RemoveAssetFromWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "PETR4.SA").Return(nil).Once()
		mCtxRmOk.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && strings.Contains(resps[0].Text, "Removido dos Favoritos")
		})).Return(nil).Once()
		// Re-render analysis card: now removed from watchlist
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mSvc.On("GetFundamentals", mock.Anything, "PETR4").Return(&market.Fundamentals{Ticker: "PETR4.SA", EPS: 5, BookValue: 25}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Price: 36.0, Currency: "BRL"}, nil).Once()
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{ID: "wl-1"}, nil).Once()
		var updatedAnalysisMenu *telebot.ReplyMarkup
		mCtxRmOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Análise Fundamentalista")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					updatedAnalysisMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleDynamicCallback(mCtxRmOk))
		assert.Equal(t, "⭐ Favoritar", updatedAnalysisMenu.InlineKeyboard[2][0].Text)

		// 8. Analysis card when already in watchlist shows 💔 Remover Favorito
		mCtxAnInWl := new(MockTelebotContext)
		mCtxAnInWl.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAnInWl.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mSvc.On("GetFundamentals", mock.Anything, "VALE3").Return(&market.Fundamentals{Ticker: "VALE3.SA", EPS: 8, BookValue: 40}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{Ticker: "VALE3.SA", Price: 60.0, Currency: "BRL"}, nil).Once()
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("GetWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000").Return(&watchlist.Watchlist{
			ID: "wl-1", Items: []watchlist.Item{{Ticker: "VALE3.SA"}},
		}, nil).Once()
		var anMenuInWl *telebot.ReplyMarkup
		mCtxAnInWl.On("Send", mock.Anything, mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					anMenuInWl = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.renderAnalysis(mCtxAnInWl, "VALE3"))
		assert.Equal(t, "💔 Remover Favorito", anMenuInWl.InlineKeyboard[2][0].Text)
	})
}
