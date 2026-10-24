package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
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
