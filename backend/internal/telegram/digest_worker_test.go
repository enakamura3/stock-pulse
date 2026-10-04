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
	totalValue, dailyChange, dailyChangePercent, returnPercent float64,
	topGainers, topLosers []DailyMover,
	todayDividends, upcomingDividends []portfolio.CalculatedDividend,
) error {
	args := m.Called(chatID, userName, portfolioName, currency, totalValue, dailyChange, dailyChangePercent, returnPercent, topGainers, topLosers, todayDividends, upcomingDividends)
	return args.Error(0)
}

func TestDailyDigestWorker_New(t *testing.T) {
	repo := new(MockRepository)
	svc := new(MockService)
	pSvc := new(MockPortfolioService)
	fiSvc := new(MockFixedIncomeSvc)
	notifier := new(MockDigestNotifier)

	// Default values: 19:00
	worker := NewDailyDigestWorker(repo, svc, pSvc, fiSvc, notifier, nil)
	assert.NotNil(t, worker)
	assert.Equal(t, 19, worker.targetHour)
	assert.Equal(t, 0, worker.targetMinute)
	assert.NotNil(t, worker.nowFunc)

	// Custom environment variables
	t.Setenv("TELEGRAM_DIGEST_HOUR", "20")
	t.Setenv("TELEGRAM_DIGEST_MINUTE", "15")
	workerEnv := NewDailyDigestWorker(repo, svc, pSvc, fiSvc, notifier, nil)
	assert.Equal(t, 20, workerEnv.targetHour)
	assert.Equal(t, 15, workerEnv.targetMinute)

	// Invalid environment variables fall back to defaults
	t.Setenv("TELEGRAM_DIGEST_HOUR", "99")
	t.Setenv("TELEGRAM_DIGEST_MINUTE", "invalid")
	workerInvalid := NewDailyDigestWorker(repo, svc, pSvc, fiSvc, notifier, nil)
	assert.Equal(t, 19, workerInvalid.targetHour)
	assert.Equal(t, 0, workerInvalid.targetMinute)
}

func TestDailyDigestWorker_ProcessDailyDigests(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	assert.NoError(t, err)

	// 2026-09-25 is a Friday (weekday)
	targetTime := time.Date(2026, 9, 25, 19, 0, 0, 0, loc)

	t.Run("nil dependencies", func(t *testing.T) {
		w1 := &DailyDigestWorker{notifier: nil}
		w1.ProcessDailyDigests(context.Background())

		w2 := &DailyDigestWorker{notifier: new(MockDigestNotifier), portfolioSvc: nil}
		w2.ProcessDailyDigests(context.Background())

		w3 := &DailyDigestWorker{notifier: new(MockDigestNotifier), portfolioSvc: new(MockPortfolioService), repo: nil}
		w3.ProcessDailyDigests(context.Background())
	})

	t.Run("suppressed on weekends (Saturday and Sunday)", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		// Saturday
		saturdayTime := time.Date(2026, 9, 26, 19, 0, 0, 0, loc)
		wSat := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return saturdayTime },
		}
		wSat.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)

		// Sunday
		sundayTime := time.Date(2026, 9, 27, 19, 0, 0, 0, loc)
		wSun := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return sundayTime },
		}
		wSun.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)
	})

	t.Run("before target time", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		earlyHourTime := time.Date(2026, 9, 25, 18, 0, 0, 0, loc)
		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return earlyHourTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)

		earlyMinuteTime := time.Date(2026, 9, 25, 18, 59, 0, 0, loc)
		w.nowFunc = func() time.Time { return earlyMinuteTime }
		w.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)
	})

	t.Run("fallback when location fails to load and nil locFunc", func(t *testing.T) {
		repo := new(MockRepository)
		pSvc := new(MockPortfolioService)
		notifier := new(MockDigestNotifier)

		// 1. locFunc returns error -> triggers fallback to FixedZone("BRT", -3h)
		// 2026-09-25 15:00 UTC = 12:00 BRT (< 19:00, so returns early)
		w := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return time.Date(2026, 9, 25, 15, 0, 0, 0, time.UTC) },
			locFunc: func(name string) (*time.Location, error) {
				return nil, errors.New("tzdata load failure")
			},
		}
		w.ProcessDailyDigests(context.Background())
		repo.AssertNotCalled(t, "GetLinkedUsers", mock.Anything)

		// 2. locFunc is nil -> defaults to time.LoadLocation
		wNil := &DailyDigestWorker{
			repo:         repo,
			portfolioSvc: pSvc,
			notifier:     notifier,
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return time.Date(2026, 9, 25, 15, 0, 0, 0, time.UTC) },
			locFunc:      nil,
		}
		wNil.ProcessDailyDigests(context.Background())
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
			targetHour:   19,
			targetMinute: 0,
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
			targetHour:   19,
			targetMinute: 0,
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
			targetHour:   19,
			targetMinute: 0,
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
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		repo.AssertExpectations(t)
		pSvc.AssertExpectations(t)
		notifier.AssertNotCalled(t, "SendDailyDigest")
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
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())
		pSvc.AssertExpectations(t)
		notifier.AssertNotCalled(t, "SendDailyDigest")
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
				Ticker:             "PETR4",
				Quantity:           100,
				CurrentPrice:       35.0,
				CurrentValue:       3500.0,
				TotalCost:          3000.0,
				DailyChange:        1.50,
				DailyChangePercent: 4.5,
			},
			{
				Ticker:             "VALE3",
				Quantity:           50,
				CurrentPrice:       60.0,
				CurrentValue:       3000.0,
				TotalCost:          3200.0,
				DailyChange:        -2.0,
				DailyChangePercent: -3.2,
			},
			{
				Ticker:             "ITUB4",
				Quantity:           30,
				CurrentPrice:       30.0,
				CurrentValue:       900.0,
				TotalCost:          850.0,
				DailyChange:        0.90,
				DailyChangePercent: 3.1,
			},
			{
				Ticker:             "BBDC4",
				Quantity:           40,
				CurrentPrice:       15.0,
				CurrentValue:       600.0,
				TotalCost:          550.0,
				DailyChange:        0.30,
				DailyChangePercent: 2.0,
			},
			{
				Ticker:             "WEGE3",
				Quantity:           20,
				CurrentPrice:       50.0,
				CurrentValue:       1000.0,
				TotalCost:          900.0,
				DailyChange:        0.50,
				DailyChangePercent: 1.0,
			},
			{
				Ticker:             "MGLU3",
				Quantity:           100,
				CurrentPrice:       2.0,
				CurrentValue:       200.0,
				TotalCost:          250.0,
				DailyChange:        -0.20,
				DailyChangePercent: -9.1,
			},
			{
				Ticker:             "BBAS3",
				Quantity:           10,
				CurrentPrice:       25.0,
				CurrentValue:       250.0,
				TotalCost:          260.0,
				DailyChange:        -1.0,
				DailyChangePercent: -3.8,
			},
			{
				Ticker:             "B3SA3",
				Quantity:           10,
				CurrentPrice:       12.0,
				CurrentValue:       120.0,
				TotalCost:          130.0,
				DailyChange:        -0.10,
				DailyChangePercent: -0.8,
			},
			{
				Ticker:             "TEST_ZERO",
				Quantity:           0,
				CurrentPrice:       0,
				CurrentValue:       0,
				TotalCost:          0,
				DailyChange:        0,
				DailyChangePercent: 15.0,
			},
			{
				Ticker:             "FLAT",
				Quantity:           10,
				CurrentPrice:       10.0,
				CurrentValue:       100.0,
				TotalCost:          100.0,
				DailyChange:        0,
				DailyChangePercent: 0.0,
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
			mock.AnythingOfType("float64"), // totalValue
			mock.AnythingOfType("float64"), // dailyChange
			mock.AnythingOfType("float64"), // dailyChangePercent
			mock.AnythingOfType("float64"), // returnPercent
			mock.MatchedBy(func(g []DailyMover) bool {
				return len(g) == 3 && g[0].Ticker == "PETR4" && g[1].Ticker == "ITUB4" && g[2].Ticker == "BBDC4"
			}),
			mock.MatchedBy(func(l []DailyMover) bool {
				return len(l) == 3 && l[0].Ticker == "MGLU3" && l[1].Ticker == "BBAS3" && l[2].Ticker == "VALE3"
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
			targetHour:   19,
			targetMinute: 0,
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
			0.0,
			[]DailyMover(nil),
			[]DailyMover(nil),
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
			targetHour:   19,
			targetMinute: 0,
			nowFunc:      func() time.Time { return targetTime },
		}

		w.ProcessDailyDigests(context.Background())

		notifier.AssertExpectations(t)
		_, err = rdb.Get(context.Background(), "daily_digest_sent:"+uID.String()+":"+targetTime.Format("2006-01-02")).Result()
		assert.Equal(t, redis.Nil, err)
	})
}
