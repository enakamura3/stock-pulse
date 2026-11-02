package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/alert"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_Phase4_MultiQuote(t *testing.T) {
	h, svc, _, mSvc, _, _, _ := setupHandlersTestWithWatchlist()

	t.Run("HandleQuote single ticker delegation", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Args").Return([]string{"PETR4"})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()

		quote := &market.Quote{
			Ticker:        "PETR4",
			Price:         34.50,
			Change:        0.50,
			ChangePercent: 1.47,
			Currency:      "BRL",
		}
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(quote, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "PETR4") && strings.Contains(msg, "34,50")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
		mCtx.AssertExpectations(t)
	})

	t.Run("HandleQuote multi ticker panel with diverse quotes and comma separation", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Args").Return([]string{"PETR4,", "VALE3", "ITUB4", "FLAT", "FAIL", "AAPL"})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()

		// 1. Positive change
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{
			Ticker:        "PETR4",
			Price:         35.0,
			ChangePercent: 2.5,
			Currency:      "R$",
		}, nil).Once()

		// 2. Negative change
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{
			Ticker:        "VALE3",
			Price:         60.0,
			ChangePercent: -1.8,
			Currency:      "R$",
		}, nil).Once()

		// 3. Neutral change (0.0)
		mSvc.On("GetQuote", mock.Anything, "ITUB4").Return(&market.Quote{
			Ticker:        "ITUB4",
			Price:         32.0,
			ChangePercent: 0.0,
			Currency:      "R$",
		}, nil).Once()

		// 4. Flat change with empty currency (fallback to default currency)
		mSvc.On("GetQuote", mock.Anything, "FLAT").Return(&market.Quote{
			Ticker:        "FLAT",
			Price:         10.0,
			ChangePercent: 0.0,
			Currency:      "",
		}, nil).Once()

		// 5. Fail ticker (unavailable)
		mSvc.On("GetQuote", mock.Anything, "FAIL").Return(nil, errors.New("not found")).Once()

		// 6. US ticker (AAPL)
		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{
			Ticker:        "AAPL",
			Price:         220.0,
			ChangePercent: 0.8,
			Currency:      "US$",
		}, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Painel de Cotações") &&
				strings.Contains(msg, "PETR4") && strings.Contains(msg, "🟢 +") &&
				strings.Contains(msg, "VALE3") && strings.Contains(msg, "🔴 ") &&
				strings.Contains(msg, "ITUB4") && strings.Contains(msg, "⚪ ") &&
				strings.Contains(msg, "FAIL") && strings.Contains(msg, "Cotação indisponível") &&
				strings.Contains(msg, "AAPL")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
		mCtx.AssertExpectations(t)
	})

	t.Run("HandleQuote multi ticker callback with > 10 tickers capping", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Callback").Return(&telebot.Callback{})
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()

		// Provide 12 tickers to test the [:10] capping
		args := []string{"T1", "T2", "T3", "T4", "T5", "T6", "T7", "T8", "T9", "T10", "T11", "T12"}
		mCtx.On("Args").Return(args)

		for i := 1; i <= 10; i++ {
			ticker := args[i-1]
			mSvc.On("GetQuote", mock.Anything, ticker).Return(&market.Quote{
				Ticker:        ticker,
				Price:         float64(i * 10),
				ChangePercent: float64(i),
				Currency:      "R$",
			}, nil).Once()
		}

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "T10") && !strings.Contains(msg, "T11") && !strings.Contains(msg, "T12")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleQuote(mCtx)
		assert.NoError(t, err)
		mCtx.AssertExpectations(t)
	})
}

func TestHandlers_Phase4_SafeMessageLength(t *testing.T) {
	h, svc, pSvc, _, _, _, _ := setupHandlersTestWithWatchlist()

	t.Run("HandleAgenda truncates when message exceeds safe character budget", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Principal"},
		}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		// Generate 45 upcoming dividend items
		var divs []portfolio.CalculatedDividend
		now := time.Now()
		for i := 0; i < 45; i++ {
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:         "VERYLONGTICKERNAME4",
				Type:           "DIVIDENDO",
				PaymentDate:    now.AddDate(0, 0, 5),
				Quantity:       1000.0,
				PerShareAmount: 0.8521,
				NetAmount:      852.10,
				Currency:       "BRL",
			})
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "... e mais") &&
				strings.Contains(msg, "Total de eventos no período: 45") &&
				len(msg) <= 4096
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAgenda(mCtx)
		assert.NoError(t, err)
		mCtx.AssertExpectations(t)
	})

	t.Run("HandleDividendsByYear truncates when historical years exceed safe character budget", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Principal"},
		}
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		// Generate 30 years of dividend data
		var divs []portfolio.CalculatedDividend
		for y := 2026; y >= 1997; y-- {
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:      "PETR4",
				Type:        "DIVIDENDO",
				PaymentDate: time.Date(y, 6, 15, 0, 0, 0, 0, time.UTC),
				NetAmount:   1500.0,
				Currency:    "BRL",
			})
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:      "VALE3",
				Type:        "JCP",
				PaymentDate: time.Date(y, 11, 20, 0, 0, 0, 0, time.UTC),
				NetAmount:   2300.0,
				Currency:    "BRL",
			})
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "... e mais") &&
				strings.Contains(msg, "com histórico no Stock Pulse") &&
				len(msg) <= 4096
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividendsByYear(mCtx)
		assert.NoError(t, err)
		mCtx.AssertExpectations(t)
	})
}

func TestDailyDigestWorker_Phase4_CoverageBranches(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	assert.NoError(t, err)

	targetTime := time.Date(2026, 9, 25, 19, 0, 0, 0, loc)

	t.Run("empty portfolio name defaults to Principal and per-portfolio dedup skips", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		repo := new(MockRepository)
		svc := new(MockService)
		pSvc := new(MockPortfolioService)
		fiSvc := new(MockFixedIncomeSvc)
		notifier := new(MockDigestNotifier)

		uID := uuid.New()
		pID1 := "p-already-sent"
		pID2 := "p-unnamed"

		todayStr := targetTime.Format("2006-01-02")
		// Mark pID1 as already sent in Redis
		_ = rdb.Set(context.Background(), "daily_digest_sent:"+uID.String()+":"+pID1+":"+todayStr, "1", 24*time.Hour)

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{
			{UserID: uID, TelegramChatID: 300, UserName: "UnnamedTester"},
		}, nil)

		svc.On("GetActivePortfolio", mock.Anything, int64(300)).Return(pID2, nil)
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{
			{ID: pID1, Name: "Enviada", BaseCurrency: "BRL"},
			{ID: pID2, Name: "", BaseCurrency: "BRL"}, // Empty name should default to "Principal"
		}, nil)

		pSvc.On("GetPortfolioDetails", mock.Anything, pID2, uID.String()).Return(
			&portfolio.Portfolio{ID: pID2, Name: "", BaseCurrency: "BRL"},
			[]portfolio.Position{
				{Ticker: "WEGE3", Quantity: 50, CurrentPrice: 40.0, CurrentValue: 2000.0, TotalCost: 1800.0, DailyChange: 0.5, DailyChangePercent: 1.25},
			},
			nil,
		)

		pSvc.On("GetPortfolioDividends", mock.Anything, pID2, uID.String()).Return([]portfolio.CalculatedDividend{}, nil)

		notifier.On("SendDailyDigest",
			int64(300),
			"UnnamedTester",
			"Principal", // Defaulted
			"BRL",
			2000.0,
			25.0,
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(nil).Once()

		w := &DailyDigestWorker{
			repo:         repo,
			svc:          svc,
			portfolioSvc: pSvc,
			fiSvc:        fiSvc,
			notifier:     notifier,
			rdb:          rdb,
			targetHour:   19,
			targetMinute: 0,
			pacingDelay:  time.Millisecond,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		notifier.AssertExpectations(t)
	})
}

func TestHandlers_Phase5_UXAndFunctionalImprovements(t *testing.T) {
	t.Run("HandleAssetList sort by RET, AZ, and unknown sort fallback", func(t *testing.T) {
		h, svc, pSvc, _, _, _ := setupHandlersTest()
		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}
		positions := []portfolio.Position{
			{Ticker: "VALE3.SA", Quantity: 10, CurrentValue: 600, TotalCost: 500},
			{Ticker: "BBAS3.SA", Quantity: 20, CurrentValue: 600, TotalCost: 400},
		}

		for _, data := range []string{"0:RET", "0:AZ", "0:UNKNOWN"} {
			mCtx := new(MockTelebotContext)
			mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
			mCtx.On("Data").Return(data)
			pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return(portfolios, nil).Once()
			svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
			pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolios[0], positions, nil).Once()
			mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil).Once()

			err := h.HandleAssetList(mCtx)
			assert.NoError(t, err)
		}
	})

	t.Run("HandleFixedIncome with MaturityDate on fixedincome and treasury positions", func(t *testing.T) {
		h, svc, pSvc, _, fiSvc, _ := setupHandlersTest()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0")

		matDate := time.Date(2028, 5, 15, 0, 0, 0, 0, time.UTC)
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "RF"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{
			{GrossValue: 1100, NetValue: 1050, TotalInvested: 1000, DaysToMaturity: 90, Asset: fixedincome.Asset{Institution: "Itaú", Type: "CDB", DebtType: "PRE", Rate: 12, MaturityDate: matDate}},
		}, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{
			{TreasuryType: "IPCA+", Quantity: 2, GrossValue: 2200, NetValue: 2100, TotalInvested: 2000, DaysToMaturity: 120, MaturityDate: matDate},
		}, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "15/05/2028") && strings.Contains(msg, "Total Aplicado")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory with ticker filter via Args and Data and USD currency", func(t *testing.T) {
		h, svc, pSvc, _, _, _ := setupHandlersTest()
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Args").Return([]string{"AAPL"})
		mCtx.On("Data").Return("0:ALL:AAPL")

		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "Global"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", mock.Anything).Return([]portfolio.Transaction{
			{ID: "tx1", Ticker: "AAPL", Currency: "USD", Type: "BUY", Quantity: 5, UnitPrice: 180, TotalCost: 900, ExecutedAt: time.Now()},
			{ID: "tx2", Ticker: "PETR4.SA", Currency: "BRL", Type: "BUY", Quantity: 10, UnitPrice: 35, TotalCost: 350, ExecutedAt: time.Now()},
		}, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo filtrado: *`AAPL`*") && strings.Contains(msg, "US$") && !strings.Contains(msg, "PETR4")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("Custody helpers and quote/analysis custody enrichment", func(t *testing.T) {
		assert.Equal(t, "", formatCustodyQuoteSummary(nil, 30, "BRL"))
		assert.Equal(t, "\n💼 *Na sua carteira:* 10 cotas\n", formatCustodyQuoteSummary(&portfolio.Position{Quantity: 10}, 0, "BRL"))
		assert.Contains(t, formatCustodyQuoteSummary(&portfolio.Position{Quantity: 10, TotalCost: 250}, 30, "BRL"), "PM: R$ 25,00 (+20,00%)")

		assert.Equal(t, "", formatCustodyAnalysisSummary(nil, nil, 30, "BRL"))
		assert.Equal(t, "", formatCustodyAnalysisSummary(&portfolio.Position{Quantity: 10}, nil, 30, "BRL"))
		assert.Contains(t, formatCustodyAnalysisSummary(&portfolio.Position{Quantity: 10, TotalCost: 250}, &market.Fundamentals{DividendYield: 10}, 30, "BRL"), "Yield on Cost (YoC): *12,00%*")

		hNil := &Handlers{}
		assert.Nil(t, hNil.findActiveCustodyPosition(context.Background(), nil, "PETR4"))

		h, svc, pSvc, mSvc, _, _ := setupHandlersTest()
		mCtxInvalidUser := new(MockTelebotContext)
		mCtxInvalidUser.Set("user_id", "")
		assert.Nil(t, h.findActiveCustodyPosition(context.Background(), mCtxInvalidUser, "PETR4"))

		// renderQuote and renderAnalysis with active custody position
		mCtxQuote := new(MockTelebotContext)
		mCtxQuote.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxQuote.On("Callback").Return((*telebot.Callback)(nil))
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Price: 36, Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 100, AveragePrice: 30},
		}, nil).Once()
		mCtxQuote.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Na sua carteira:") && strings.Contains(msg, "+20,00%")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.renderQuote(mCtxQuote, "PETR4"))

		mCtxAnalise := new(MockTelebotContext)
		mCtxAnalise.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAnalise.On("Callback").Return((*telebot.Callback)(nil))
		mSvc.On("GetFundamentals", mock.Anything, "PETR4").Return(&market.Fundamentals{Ticker: "PETR4.SA", EPS: 5, BookValue: 30, DividendYield: 10}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Price: 36, Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 100, AveragePrice: 30},
		}, nil).Once()
		mCtxAnalise.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Yield on Cost (YoC):")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.renderAnalysis(mCtxAnalise, "PETR4"))
	})

	t.Run("handleAlertCondition with quote & PM buttons and handleSelectedAlertPrice", func(t *testing.T) {
		h, svc, pSvc, mSvc, _, alertSvc := setupHandlersTest()

		// 1. BELOW condition with quote and custody PM
		mCtxBelow := new(MockTelebotContext)
		mCtxBelow.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_COND", Ticker: "PETR4.SA"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4.SA").Return(&market.Quote{Ticker: "PETR4.SA", Price: 40.0, Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 100, AveragePrice: 32.0},
		}, nil).Once()
		mCtxBelow.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Preço Atual:") && strings.Contains(msg, "Seu PM:")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleAlertConditionBelow(mCtxBelow))

		// 2. ABOVE condition with quote
		mCtxAbove := new(MockTelebotContext)
		mCtxAbove.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_COND", Ticker: "PETR4.SA"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4.SA").Return(&market.Quote{Ticker: "PETR4.SA", Price: 40.0, Currency: "BRL"}, nil).Once()
		mCtxAbove.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "ACIMA DE") && strings.Contains(msg, "Preço Atual:")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleAlertConditionAbove(mCtxAbove))

		// 3. handleSelectedAlertPrice: invalid state
		mCtxNoState := new(MockTelebotContext)
		mCtxNoState.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), errors.New("no state")).Once()
		mCtxNoState.On("Edit", "⚠️ Nenhuma criação de alerta em andamento.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.handleSelectedAlertPrice(mCtxNoState, "35.00"))

		// 4. handleSelectedAlertPrice: invalid price
		mCtxBadPrice := new(MockTelebotContext)
		mCtxBadPrice.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4.SA", Type: "BELOW"}, nil).Once()
		mCtxBadPrice.On("Edit", "⚠️ Preço inválido.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.handleSelectedAlertPrice(mCtxBadPrice, "invalid"))

		// 5. handleSelectedAlertPrice: userID error
		mCtxUserErr := new(MockTelebotContext)
		mCtxUserErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUserErr.Set("user_id", nil)
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4.SA", Type: "BELOW"}, nil).Once()
		mCtxUserErr.On("Edit", mock.Anything, mock.Anything).Return(nil).Once()
		assert.Error(t, h.handleSelectedAlertPrice(mCtxUserErr, "35.00"))

		// 6. handleSelectedAlertPrice: CreateAlert error
		mCtxCreateErr := new(MockTelebotContext)
		mCtxCreateErr.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4.SA", Type: "BELOW"}, nil).Once()
		alertSvc.On("CreateAlert", mock.Anything, mock.Anything, "PETR4.SA", 35.0, "BELOW").Return((*alert.Alert)(nil), errors.New("db err")).Once()
		mCtxCreateErr.On("Edit", "❌ Ocorreu um erro ao salvar o alerta. Tente novamente mais tarde.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.handleSelectedAlertPrice(mCtxCreateErr, "35.00"))

		// 7. handleSelectedAlertPrice: success via DynamicCallback (BELOW with empty currency)
		mCtxCbOk := new(MockTelebotContext)
		mCtxCbOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxCbOk.On("Callback").Return(&telebot.Callback{Data: "\fbtn_alert_price_35.50"})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "ALERT_EXPECT_PRICE", Ticker: "PETR4.SA", Type: "BELOW"}, nil).Once()
		alertSvc.On("CreateAlert", mock.Anything, mock.Anything, "PETR4.SA", 35.5, "BELOW").Return(&alert.Alert{
			Ticker:      "PETR4.SA",
			TargetPrice: 35.5,
			Condition:   "BELOW",
			Currency:    "",
		}, nil).Once()
		mCtxCbOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Alerta Criado com Sucesso!") && strings.Contains(msg, "abaixo de")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleDynamicCallback(mCtxCbOk))
	})

	t.Run("HandleLaunchOperation > 12 assets, finalizeTransaction custody PM, and QUOTE_EXPECT_TICKER multi-quote", func(t *testing.T) {
		h, svc, pSvc, mSvc, _, _ := setupHandlersTest()

		// 1. HandleLaunchOperation with 15 positions
		mCtxLaunch := new(MockTelebotContext)
		mCtxLaunch.On("Chat").Return(&telebot.Chat{ID: 123})
		var pos15 []portfolio.Position
		for i := 1; i <= 15; i++ {
			pos15 = append(pos15, portfolio.Position{Ticker: "T" + string(rune('A'+i)), CurrentValue: float64(i * 100)})
		}
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolio.Portfolio{ID: "p1"}, pos15, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		mCtxLaunch.On("Edit", mock.Anything, mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxLaunch))

		// 2. finalizeTransaction with updated custody position (TotalCost/Quantity fallback)
		mCtxFin := new(MockTelebotContext)
		mCtxFin.On("Chat").Return(&telebot.Chat{ID: 123})
		pSvc.On("AddTransaction", mock.Anything, mock.Anything, mock.Anything).Return(&portfolio.Transaction{ID: "tx-new", Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 150, TotalCost: 4500},
		}, nil).Once()
		mCtxFin.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Posição Atualizada:") && strings.Contains(msg, "150 cotas (PM: R$ 30,00)")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.finalizeTransaction(mCtxFin, &ConversationState{
			PortfolioID: "p1",
			Ticker:      "PETR4.SA",
			Type:        "BUY",
			Quantity:    50,
			UnitPrice:   30,
		}, 0, true))

		// 3. HandleText QUOTE_EXPECT_TICKER with multiple tickers
		mCtxMulti := new(MockTelebotContext)
		mCtxMulti.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxMulti.On("Text").Return("PETR4, VALE3")
		mCtxMulti.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "QUOTE_EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Price: 35, Currency: "R$"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "VALE3").Return(&market.Quote{Ticker: "VALE3.SA", Price: 60, Currency: "R$"}, nil).Once()
		mCtxMulti.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Painel de Cotações")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleText(mCtxMulti))

		// 4. HandleText QUOTE_EXPECT_TICKER single ticker with custody position
		mCtxSingle := new(MockTelebotContext)
		mCtxSingle.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxSingle.On("Text").Return("PETR4")
		mCtxSingle.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "QUOTE_EXPECT_TICKER"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Price: 36, Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 100, AveragePrice: 30},
		}, nil).Once()
		mCtxSingle.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Na sua carteira:")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleText(mCtxSingle))
	})
}
