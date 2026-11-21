package telegram

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/onigiri/stock-pulse/backend/internal/watchlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestPhase1_UnitsClassificationAndAssetEmoji(t *testing.T) {
	t.Run("getMacroCategoryKey respects STOCK_BR units and FIAGRO", func(t *testing.T) {
		assert.Equal(t, "STOCK", getMacroCategoryKey("STOCK_BR", "TAEE11.SA"))
		assert.Equal(t, "STOCK", getMacroCategoryKey("STOCK_BR", "SANB11"))
		assert.Equal(t, "STOCK", getMacroCategoryKey("STOCK_US", "AAPL"))
		assert.Equal(t, "FII", getMacroCategoryKey("FII", "HGLG11.SA"))
		assert.Equal(t, "FII", getMacroCategoryKey("FIAGRO", "KNCA11.SA"))
		assert.Equal(t, "ETF", getMacroCategoryKey("ETF_BR", "BOVA11.SA"))
		assert.Equal(t, "ETF", getMacroCategoryKey("ETF_US", "VOO"))
		assert.Equal(t, "BDR", getMacroCategoryKey("BDR", "AAPL34.SA"))
		assert.Equal(t, "CRYPTO", getMacroCategoryKey("CRYPTO", "BTC-USD"))
		assert.Equal(t, "RF", getMacroCategoryKey("FIXED_INCOME", "CDB"))
		assert.Equal(t, "FII", getMacroCategoryKey("", "HGLG11.SA"))
		assert.Equal(t, "FII", getMacroCategoryKey("", "MXRF11"))
		assert.Equal(t, "STOCK", getMacroCategoryKey("", "PETR4.SA"))
	})

	t.Run("getAssetTypeEmoji respects STOCK_BR units, STOCK_US, ETF_US, and fallbacks", func(t *testing.T) {
		assert.Equal(t, "📈", getAssetTypeEmoji("STOCK_BR", "TAEE11.SA"))
		assert.Equal(t, "📈", getAssetTypeEmoji("STOCK_BR", "PETR4"))
		assert.Equal(t, "🇺🇸", getAssetTypeEmoji("STOCK_US", "AAPL"))
		assert.Equal(t, "📊", getAssetTypeEmoji("ETF_US", "VOO"))
		assert.Equal(t, "🪙", getAssetTypeEmoji("", "BTC-USD"))
		assert.Equal(t, "📈", getAssetTypeEmoji("", "PETR4"))
		assert.Equal(t, "🏢", getAssetTypeEmoji("", "HGLG11"))
	})

	t.Run("getCurrencySymbol handles both ISO codes and symbols", func(t *testing.T) {
		assert.Equal(t, "R$", getCurrencySymbol("BRL"))
		assert.Equal(t, "R$", getCurrencySymbol("R$"))
		assert.Equal(t, "US$", getCurrencySymbol("USD"))
		assert.Equal(t, "US$", getCurrencySymbol("US$"))
		assert.Equal(t, "€", getCurrencySymbol("EUR"))
		assert.Equal(t, "€", getCurrencySymbol("€"))
		assert.Equal(t, "R$", getCurrencySymbol("OTHER"))
	})
}

func TestPhase1_PortfolioBaseCurrencyAndUnitsAllocation(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("HandlePortfolioSummary uses BaseCurrency and groups TAEE11 under Ações", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		port := portfolio.Portfolio{ID: "p-usd", Name: "Global", BaseCurrency: "USD"}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{port}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p-usd", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "TAEE11.SA", Type: "STOCK_BR", Quantity: 100, CurrentPrice: 35, CurrentValue: 3500, TotalCost: 3000, DailyChange: 0.5, DailyChangePercent: 1.5, Currency: "USD"},
		}
		pSvc.On("GetPortfolioDetails", mock.Anything, "p-usd", "00000000-0000-0000-0000-000000000000").Return(&port, positions, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Valor Total: *US$ 3.500,00*") &&
				strings.Contains(msg, "📈 Ações: *US$ 3.500,00*") &&
				!strings.Contains(msg, "🏢 FIIs")
		}), mock.Anything).Return(nil).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAssetList uses BaseCurrency in header", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("0:VAL")

		port := portfolio.Portfolio{ID: "p-usd", Name: "Global", BaseCurrency: "USD"}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{port}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p-usd", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "AAPL", Type: "STOCK_US", Quantity: 10, CurrentPrice: 200, CurrentValue: 2000, TotalCost: 1500, Currency: "USD"},
		}
		pSvc.On("GetPortfolioDetails", mock.Anything, "p-usd", "00000000-0000-0000-0000-000000000000").Return(&port, positions, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Patrimônio em Ativos: *US$ 2.000,00* (Custo: US$ 1.500,00)")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)
	})
}

func TestPhase1_MultiQuoteAndMarkdownEscapeFixes(t *testing.T) {
	t.Run("renderMultiQuote formats currency symbol and single plus sign", func(t *testing.T) {
		h, svc, _, mSvc, _, _ := setupHandlersTest()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Args").Return([]string{"PETR4", "AAPL"})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()

		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{
			Ticker: "PETR4.SA", Price: 36.50, ChangePercent: 2.50, Currency: "BRL",
		}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{
			Ticker: "AAPL", Price: 210.00, ChangePercent: -1.10, Currency: "USD",
		}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "R$ 36,50 (🟢 +2,50%)") &&
				!strings.Contains(msg, "🟢 ++") &&
				!strings.Contains(msg, "BRL ") &&
				strings.Contains(msg, "US$ 210,00 (🔴 -1,10%)")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleUndoLastOperation and Watchlist escape markdown in dynamic fields", func(t *testing.T) {
		h, svc, pSvc, _, _, _, wlSvc := setupHandlersTestWithWatchlist()

		mCtxUndo := new(MockTelebotContext)
		mCtxUndo.On("Chat").Return(&telebot.Chat{ID: 123})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "Minha_Carteira"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{
			{ID: "tx-1", Ticker: "PETR4.SA", Type: "BUY", Quantity: 10, UnitPrice: 35, TotalCost: 350, Currency: "BRL", ExecutedAt: time.Now()},
		}, nil).Once()
		mCtxUndo.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, `Minha\_Carteira`)
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleUndoLastOperation(mCtxUndo))

		// Watchlist error with underscore
		mCtxWl := new(MockTelebotContext)
		mCtxWl.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxWl.On("Respond", mock.Anything).Return(nil).Once()
		wlSvc.On("GetWatchlists", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]watchlist.Watchlist{{ID: "wl-1"}}, nil).Once()
		wlSvc.On("AddAssetToWatchlist", mock.Anything, "wl-1", "00000000-0000-0000-0000-000000000000", "BAD_TICKER").Return(nil, errors.New("already_exists")).Once()
		mCtxWl.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, `already\_exists`)
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.handleQuickWatchlistAdd(mCtxWl, "BAD_TICKER"))
	})
}

func TestPhase1_DividendsCumDateFilterAndMonthPagination(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()
	portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}

	t.Run("HandleDividends with ticker filter via Args, CumDate, >5 items and Ver Todos button", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Args").Return([]string{"vivt3"})

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		var divs []portfolio.CalculatedDividend
		for i := 1; i <= 7; i++ {
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:         "VIVT3.SA",
				AssetType:      "STOCK_BR",
				Type:           "JCP",
				Currency:       "BRL",
				CumDate:        now.AddDate(0, 0, -i-5),
				PaymentDate:    now.AddDate(0, 0, -i),
				Quantity:       100,
				PerShareAmount: 0.50,
				NetAmount:      50.0,
			})
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:         "VIVT3.SA",
				AssetType:      "STOCK_BR",
				Type:           "DIVIDENDO",
				Currency:       "BRL",
				CumDate:        now.AddDate(0, 0, i),
				PaymentDate:    now.AddDate(0, 0, i+10),
				Quantity:       100,
				PerShareAmount: 0.30,
				NetAmount:      30.0,
			})
		}
		divs = append(divs, portfolio.CalculatedDividend{
			Ticker:      "PETR4.SA",
			AssetType:   "STOCK_BR",
			Type:        "DIVIDENDO",
			Currency:    "BRL",
			PaymentDate: now.AddDate(0, 0, -2),
			NetAmount:   999.0,
		})
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		var sentMenu *telebot.ReplyMarkup
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo filtrado: *`VIVT3`*") &&
				strings.Contains(msg, "Com: ") &&
				!strings.Contains(msg, "PETR4")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					sentMenu = menu
					return true
				}
			}
			return false
		})).Return(nil).Once()

		assert.NoError(t, h.HandleDividends(mCtx))
		assert.NotNil(t, sentMenu)
		hasVerTodos := false
		for _, row := range sentMenu.InlineKeyboard {
			for _, btn := range row {
				if strings.Contains(btn.Text, "Ver Todos") {
					hasVerTodos = true
				}
			}
		}
		assert.True(t, hasVerTodos)
	})

	t.Run("HandleDividends with ticker filter via callback Data and zero matches", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("WEGE3")

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.CalculatedDividend{
			{Ticker: "PETR4.SA", NetAmount: 10, PaymentDate: time.Now()},
		}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Nenhum provento registrado para `WEGE3`")
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleDividends(mCtx))
	})

	t.Run("HandleAgenda shows CumDate with and without quantity/per-share", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		cum := now.AddDate(0, 0, -2)
		divs := []portfolio.CalculatedDividend{
			{Ticker: "VIVT3.SA", AssetType: "STOCK_BR", Type: "JCP", Currency: "BRL", CumDate: cum, PaymentDate: now.AddDate(0, 0, 5), Quantity: 100, PerShareAmount: 0.4, NetAmount: 40},
			{Ticker: "ITSA4.SA", AssetType: "STOCK_BR", Type: "DIVIDENDO", Currency: "BRL", CumDate: cum, PaymentDate: now.AddDate(0, 0, 6), NetAmount: 25},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		expectedCum := "Com: " + cum.Format("2006-01-02")
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "• "+expectedCum) && strings.Contains(msg, "_"+expectedCum+"_")
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleAgenda(mCtx))
	})

	t.Run("HandleDividendsByYear with ticker filter propagates filter", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("VIVT3")

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.CalculatedDividend{
			{Ticker: "VIVT3.SA", Type: "JCP", Currency: "BRL", PaymentDate: time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC), NetAmount: 100},
			{Ticker: "PETR4.SA", Type: "DIV", Currency: "BRL", PaymentDate: time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC), NetAmount: 500},
		}, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo filtrado: *`VIVT3`*") &&
				strings.Contains(msg, "R$ 100,00") &&
				!strings.Contains(msg, "PETR4")
		}), mock.Anything).Return(nil).Once()

		assert.NoError(t, h.HandleDividendsByYear(mCtx))
	})

	t.Run("HandleDividendsByMonth with ticker filter, page clamping and Página X de Y", func(t *testing.T) {
		divs := []portfolio.CalculatedDividend{
			{Ticker: "VIVT3.SA", Type: "JCP", Currency: "BRL", PaymentDate: time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC), NetAmount: 10},
			{Ticker: "VIVT3.SA", Type: "JCP", Currency: "BRL", PaymentDate: time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC), NetAmount: 20},
			{Ticker: "VIVT3.SA", Type: "JCP", Currency: "BRL", PaymentDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC), NetAmount: 30},
			{Ticker: "VIVT3.SA", Type: "JCP", Currency: "BRL", PaymentDate: time.Date(2025, 4, 10, 0, 0, 0, 0, time.UTC), NetAmount: 40},
		}

		// Negative page clamped to 0, with ticker filter
		mCtxNeg := new(MockTelebotContext)
		mCtxNeg.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxNeg.On("Respond", mock.Anything).Return(nil).Once()
		mCtxNeg.On("Data").Return("-2:VIVT3")
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtxNeg.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo filtrado: *`VIVT3`*") &&
				strings.Contains(msg, "_Página 1 de 2_") &&
				strings.Contains(msg, "• *2025-04*")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleDividendsByMonth(mCtxNeg))

		// Out-of-bounds page clamped to last page (page 2 of 2)
		mCtxOOB := new(MockTelebotContext)
		mCtxOOB.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOOB.On("Respond", mock.Anything).Return(nil).Once()
		mCtxOOB.On("Data").Return("99:VIVT3")
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtxOOB.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "_Página 2 de 2_") &&
				strings.Contains(msg, "• *2025-01*")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleDividendsByMonth(mCtxOOB))
	})
}

func TestPhase1_HistoryDeleteConfirmationFlow(t *testing.T) {
	t.Run("compact callback encoding and decoding respects 63-byte budget", func(t *testing.T) {
		assert.Equal(t, "A", encodeHistoryFilterChar("ALL"))
		assert.Equal(t, "B", encodeHistoryFilterChar("BUY"))
		assert.Equal(t, "S", encodeHistoryFilterChar("SELL"))
		assert.Equal(t, "ALL", decodeHistoryFilterChar("A"))
		assert.Equal(t, "BUY", decodeHistoryFilterChar("B"))
		assert.Equal(t, "SELL", decodeHistoryFilterChar("S"))

		uuid := "12345678-1234-1234-1234-1234567890ab"
		cbFull := buildTxDeleteCallback("txda_", uuid, 2, "BUY", "PETR4")
		assert.LessOrEqual(t, len(cbFull), 63)
		txID, ret := parseTxDeleteCallback(strings.TrimPrefix(cbFull, "txda_"))
		assert.Equal(t, uuid, txID)
		assert.Equal(t, "2:BUY:PETR4", ret)

		cbNoTicker := buildTxDeleteCallback("txda_", uuid, 1, "SELL", "")
		txID, ret = parseTxDeleteCallback(strings.TrimPrefix(cbNoTicker, "txda_"))
		assert.Equal(t, uuid, txID)
		assert.Equal(t, "1:SELL", ret)

		// Long ticker exceeds 63 bytes -> falls back to base without ticker
		cbLongTicker := buildTxDeleteCallback("txda_", uuid, 1, "ALL", "VERYLONGTICKERNAMETHATEXCEEDS63BYTES")
		assert.LessOrEqual(t, len(cbLongTicker), 63)
		txID, ret = parseTxDeleteCallback(strings.TrimPrefix(cbLongTicker, "txda_"))
		assert.Equal(t, uuid, txID)
		assert.Equal(t, "1:ALL", ret)

		// Excessively long ID -> falls back to prefix+txID
		longID := strings.Repeat("x", 60)
		cbLongID := buildTxDeleteCallback("txda_", longID, 1, "ALL", "")
		assert.Equal(t, "txda_"+longID, cbLongID)

		// Payload without return context
		txID, ret = parseTxDeleteCallback("tx-only")
		assert.Equal(t, "tx-only", txID)
		assert.Equal(t, "", ret)
	})

	t.Run("handleAskDeleteTransaction and handleDeleteTransactionWithReturn via DynamicCallback", func(t *testing.T) {
		h, svc, pSvc, _, _, _ := setupHandlersTest()
		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Minha_Carteira"}}
		txTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

		// 1. userID error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUserErr.On("Respond", mock.Anything).Return(nil).Once()
		mCtxUserErr.store = map[string]interface{}{"user_id": 12345}
		mCtxUserErr.On("Send", mock.MatchedBy(func(s string) bool { return strings.Contains(s, "Sessão inválida") }), mock.Anything).Return(nil).Once()
		assert.Error(t, h.handleAskDeleteTransaction(mCtxUserErr, "tx-1"))

		// 2. 0 portfolios
		mCtxNoPort := new(MockTelebotContext)
		mCtxNoPort.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxNoPort.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtxNoPort.On("Edit", "⚠️ Carteira não encontrada.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.handleAskDeleteTransaction(mCtxNoPort, "tx-1"))

		// 3. Not found with returnData
		mCtxNFHist := new(MockTelebotContext)
		mCtxNFHist.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxNFHist.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{}, nil).Once()
		mCtxNFHist.On("Edit", "⚠️ Operação não encontrada ou já foi removida.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.handleAskDeleteTransaction(mCtxNFHist, "tx-missing|1|B|PETR4"))

		// 4. Not found without returnData
		mCtxNFPlain := new(MockTelebotContext)
		mCtxNFPlain.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxNFPlain.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{}, nil).Once()
		mCtxNFPlain.On("Edit", "⚠️ Operação não encontrada ou já foi removida.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.handleAskDeleteTransaction(mCtxNFPlain, "tx-missing"))

		// 5. Found via HandleDynamicCallback txda_ (from history with returnData, BUY + USD)
		mCtxAskHist := new(MockTelebotContext)
		mCtxAskHist.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAskHist.On("Respond", mock.Anything).Return(nil).Once()
		mCtxAskHist.On("Callback").Return(&telebot.Callback{Data: "\ftxda_tx-1|1|B|AAPL"})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{
			{ID: "tx-1", Ticker: "AAPL", Type: "BUY", Quantity: 5, UnitPrice: 200, Fee: 1, TotalCost: 1001, Currency: "USD", ExecutedAt: txTime},
		}, nil).Once()
		mCtxAskHist.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Confirmar exclusão de operação") &&
				strings.Contains(msg, `Minha\_Carteira`) &&
				strings.Contains(msg, "AAPL") &&
				strings.Contains(msg, "US$ 200,00")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleDynamicCallback(mCtxAskHist))

		// 6. Found via HandleDynamicCallback txda_ (post-creation without returnData, SELL)
		mCtxAskPlain := new(MockTelebotContext)
		mCtxAskPlain.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAskPlain.On("Respond", mock.Anything).Return(nil).Once()
		mCtxAskPlain.On("Callback").Return(&telebot.Callback{Data: "\ftxda_tx-2"})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{
			{ID: "tx-2", Ticker: "PETR4.SA", Type: "SELL", Quantity: 100, UnitPrice: 36, TotalCost: 3600, ExecutedAt: txTime},
		}, nil).Once()
		mCtxAskPlain.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "VENDA") && strings.Contains(msg, "PETR4")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleDynamicCallback(mCtxAskPlain))

		// 7. Confirm deletion via HandleDynamicCallback txdc_ with returnData -> Voltar ao Histórico button
		mCtxConf := new(MockTelebotContext)
		mCtxConf.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxConf.On("Respond", mock.Anything).Return(nil).Once()
		mCtxConf.On("Callback").Return(&telebot.Callback{Data: "\ftxdc_tx-1|1|B|AAPL"})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("DeleteTransaction", mock.Anything, "tx-1", "p1", "00000000-0000-0000-0000-000000000000").Return(nil).Once()
		var confMenu *telebot.ReplyMarkup
		mCtxConf.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Operação desfeita com sucesso")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if menu, ok := opt.(*telebot.ReplyMarkup); ok {
					confMenu = menu
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleDynamicCallback(mCtxConf))
		assert.NotNil(t, confMenu)
		assert.Equal(t, "btn_history", confMenu.InlineKeyboard[0][0].Unique)
		assert.Equal(t, "1:BUY:AAPL", confMenu.InlineKeyboard[0][0].Data)
	})
}
