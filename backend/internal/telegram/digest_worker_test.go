package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDigestNotifier struct {
	mock.Mock
}

func (m *MockDigestNotifier) SendDailyDigest(
	chatID int64,
	userName, portfolioName, currency string,
	totalValue, dailyChange, returnPercent float64,
	todayDividends, upcomingDividends []portfolio.CalculatedDividend,
) error {
	args := m.Called(chatID, userName, portfolioName, currency, totalValue, dailyChange, returnPercent, todayDividends, upcomingDividends)
	return args.Error(0)
}

func TestDailyDigestWorker_New(t *testing.T) {
	repo := new(MockRepository)
	svc := new(MockService)
	pSvc := new(MockPortfolioService)
	fiSvc := new(MockFixedIncomeSvc)
	notifier := new(MockDigestNotifier)

	worker := NewDailyDigestWorker(repo, svc, pSvc, fiSvc, notifier, nil)
	assert.NotNil(t, worker)
	assert.Equal(t, 8, worker.targetHour)
	assert.Equal(t, 30, worker.targetMinute)
	assert.NotNil(t, worker.nowFunc)
}

func TestDailyDigestWorker_ProcessDailyDigests(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	assert.NoError(t, err)

	targetTime := time.Date(2026, 9, 26, 8, 30, 0, 0, loc)

	t.Run("nil dependencies", func(t *testing.T) {
		w1 := &DailyDigestWorker{notifier: nil}
		w1.ProcessDailyDigests(context.Background())

		w2 := &DailyDigestWorker{notifier: new(MockDigestNotifier), portfolioSvc: nil}
		w2.ProcessDailyDigests(context.Background())

		w3 := &DailyDigestWorker{notifier: new(MockDigestNotifier), portfolioSvc: new(MockPortfolioService), repo: nil}
		w3.ProcessDailyDigests(context.Background())
	})

	t.Run("before target time", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		earlyTime := time.Date(2026, 9, 26, 7, 45, 0, 0, loc)
		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return earlyTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)

		earlyMinuteTime := time.Date(2026, 9, 26, 8, 29, 0, 0, loc)
		w.nowFunc = func() time.Time { return earlyMinuteTime }
		w.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)
	})

	t.Run("repo GetLinkedUsers returns error", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		repo.On("GetLinkedUsers", mock.Anything).Return(nil, errors.New("db connection failure"))

		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("no linked users", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{}, nil)

		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("skip user already sent in redis", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		uID := uuid.New()
		todayStr := targetTime.Format("2006-01-02")
		_ = rdb.Set(context.Background(), "daily_digest_sent:"+uID.String()+":"+todayStr, "1", time.Hour).Err()

		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{
			{UserID: uID, TelegramChatID: 100, UserName: "Maria"},
		}, nil)

		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			rdb:          rdb,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertExpectations(t)
		pSvc.AssertNotCalled(t, "GetPortfolios", mock.Anything, mock.Anything)
	})

	t.Run("user has no portfolios or GetPortfolios errors", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		uID1 := uuid.New()
		uID2 := uuid.New()

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{
			{UserID: uID1, TelegramChatID: 101, UserName: "João"},
			{UserID: uID2, TelegramChatID: 102, UserName: "Ana"},
		}, nil)

		pSvc.On("GetPortfolios", mock.Anything, uID1.String()).Return(nil, errors.New("portfolio svc error"))
		pSvc.On("GetPortfolios", mock.Anything, uID2.String()).Return([]portfolio.Portfolio{}, nil)

		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertExpectations(t)
		pSvc.AssertExpectations(t)
		notifier.AssertNotCalled(t, "SendDailyDigest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("GetPortfolioDetails errors", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		svc := new(MockService)
		notifier := new(MockDigestNotifier)

		uID := uuid.New()
		pID := "port-1"

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{
			{UserID: uID, TelegramChatID: 103, UserName: "Carlos"},
		}, nil)

		svc.On("GetActivePortfolio", mock.Anything, int64(103)).Return("", nil)
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{
			{ID: pID, Name: "Ações", IsDefault: true},
		}, nil)
		pSvc.On("GetPortfolioDetails", mock.Anything, pID, uID.String()).Return(nil, nil, errors.New("details query error"))

		w := &DailyDigestWorker{
			repo:         repo,
			svc:          svc,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		pSvc.AssertExpectations(t)
		notifier.AssertNotCalled(t, "SendDailyDigest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("successful processing with equities, fixed income, today and upcoming dividends", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		repo := new(MockRepository)
		svc := new(MockService)
		pSvc := new(MockPortfolioService)
		fiSvc := new(MockFixedIncomeSvc)
		notifier := new(MockDigestNotifier)

		uID := uuid.New()
		pID := "port-full"

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{
			{UserID: uID, TelegramChatID: 200, UserName: "Eduardo"},
		}, nil)

		svc.On("GetActivePortfolio", mock.Anything, int64(200)).Return(pID, nil)

		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{
			{ID: pID, Name: "Principal", BaseCurrency: "BRL"},
		}, nil)

		portDetails := &portfolio.Portfolio{
			ID:           pID,
			Name:         "Principal",
			BaseCurrency: "BRL",
		}

		positions := []portfolio.Position{
			{
				Ticker:       "PETR4",
				Quantity:     100,
				CurrentPrice: 35.0,
				CurrentValue: 3500.0,
				TotalCost:    3000.0,
				DailyChange:  1.50,
			},
			{
				Ticker:       "TEST_ZERO",
				Quantity:     0,
				CurrentPrice: 0,
				CurrentValue: 0,
				TotalCost:    0,
				DailyChange:  0,
			},
		}

		pSvc.On("GetPortfolioDetails", mock.Anything, pID, uID.String()).Return(portDetails, positions, nil)

		fiPositions := []fixedincome.Position{
			{
				NetValue:      2000.0,
				TotalInvested: 1800.0,
			},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, pID).Return(fiPositions, nil)

		// Dividends:
		// 1. Zero payment date (should be skipped)
		// 2. Today dividend
		// 3. Upcoming within 7 days
		// 4. Past dividend
		// 5. Far future dividend (> 7 days)
		divs := []portfolio.CalculatedDividend{
			{
				Ticker:   "SKIP1",
				Currency: "BRL",
			},
			{
				Ticker:      "PETR4",
				Type:        "DIVIDENDO",
				Currency:    "BRL",
				NetAmount:   150.0,
				PaymentDate: targetTime, // today
			},
			{
				Ticker:      "VALE3",
				Type:        "JCP",
				Currency:    "BRL",
				NetAmount:   200.0,
				PaymentDate: targetTime.AddDate(0, 0, 3), // upcoming in 3 days
			},
			{
				Ticker:      "BBAS3",
				Type:        "DIVIDENDO",
				Currency:    "BRL",
				NetAmount:   50.0,
				PaymentDate: targetTime.AddDate(0, 0, -2), // past
			},
			{
				Ticker:      "ITUB4",
				Type:        "JCP",
				Currency:    "BRL",
				NetAmount:   75.0,
				PaymentDate: targetTime.AddDate(0, 0, 15), // > 7 days
			},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, pID, uID.String()).Return(divs, nil)

		notifier.On("SendDailyDigest",
			int64(200),
			"Eduardo",
			"Principal",
			"BRL",
			5500.0, // 3500 + 2000
			150.0,  // dailyChange: 1.50 * 100 * 1.0 = 150.0
			mock.MatchedBy(func(ret float64) bool {
				// (5500 - 4800) / 4800 * 100 = 700 / 4800 * 100 = 14.5833%
				return ret > 14.0 && ret < 15.0
			}),
			mock.MatchedBy(func(tDivs []portfolio.CalculatedDividend) bool {
				return len(tDivs) == 1 && tDivs[0].Ticker == "PETR4"
			}),
			mock.MatchedBy(func(uDivs []portfolio.CalculatedDividend) bool {
				return len(uDivs) == 1 && uDivs[0].Ticker == "VALE3"
			}),
		).Return(nil)

		w := &DailyDigestWorker{
			repo:         repo,
			svc:          svc,
			portfolioSvc: pSvc,
			fiSvc:        fiSvc,
			notifier:     notifier,
			rdb:          rdb,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())

		repo.AssertExpectations(t)
		pSvc.AssertExpectations(t)
		fiSvc.AssertExpectations(t)
		notifier.AssertExpectations(t)

		// Assert Redis dedup key is set
		dedupVal, err := rdb.Get(context.Background(), "daily_digest_sent:"+uID.String()+":"+targetTime.Format("2006-01-02")).Result()
		assert.NoError(t, err)
		assert.Equal(t, "1", dedupVal)
	})

	t.Run("SendDailyDigest error does not set redis key and fixed income error handled", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		repo := new(MockRepository)
		svc := new(MockService)
		pSvc := new(MockPortfolioService)
		fiSvc := new(MockFixedIncomeSvc)
		notifier := new(MockDigestNotifier)

		uID := uuid.New()
		pID := "port-fail"

		repo.On("GetLinkedUsers", mock.Anything).Return([]TelegramUserLink{
			{UserID: uID, TelegramChatID: 201, UserName: "Lucas"},
		}, nil)

		svc.On("GetActivePortfolio", mock.Anything, int64(201)).Return("", nil)
		pSvc.On("GetPortfolios", mock.Anything, uID.String()).Return([]portfolio.Portfolio{
			{ID: pID, Name: "Ações", BaseCurrency: ""},
		}, nil)

		pSvc.On("GetPortfolioDetails", mock.Anything, pID, uID.String()).Return(
			&portfolio.Portfolio{ID: pID, Name: "Ações", BaseCurrency: ""},
			[]portfolio.Position{},
			nil,
		)

		fiSvc.On("GetPortfolioPositions", mock.Anything, pID).Return(nil, errors.New("fi error"))
		pSvc.On("GetPortfolioDividends", mock.Anything, pID, uID.String()).Return(nil, errors.New("divs error"))

		notifier.On("SendDailyDigest",
			int64(201),
			"Lucas",
			"Ações",
			"BRL",
			0.0,
			0.0,
			0.0,
			[]portfolio.CalculatedDividend(nil),
			[]portfolio.CalculatedDividend(nil),
		).Return(errors.New("send failed"))

		w := &DailyDigestWorker{
			repo:         repo,
			svc:          svc,
			portfolioSvc: pSvc,
			fiSvc:        fiSvc,
			notifier:     notifier,
			rdb:          rdb,
			targetHour:   8,
			targetMinute: 30,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())

		notifier.AssertExpectations(t)
		_, err = rdb.Get(context.Background(), "daily_digest_sent:"+uID.String()+":"+targetTime.Format("2006-01-02")).Result()
		assert.Equal(t, redis.Nil, err)
	})
}
