package portfolio

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDividendWorker_SyncAllDividends_FuzzyMatch(t *testing.T) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)

	worker := NewDividendWorker(repo, ms)

	ctx := context.Background()

	assets := []AssetCompact{
		{ID: "asset-1", Ticker: "PETR4.SA", AssetType: "STOCK_BR"},
	}

	repo.On("GetAllAssets", mock.Anything).Return(assets, nil)

	// Scraper returns a dividend of 1.54
	exDate := time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)
	payDate := time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC)

	scrapedEvents := []market.DividendEvent{
		{
			Date:        exDate,
			Type:        "Dividendo",
			Amount:      1.54,
			PaymentDate: payDate,
		},
	}

	ms.On("GetDividends", mock.Anything, "PETR4.SA", "STOCK_BR").Return(scrapedEvents, nil)

	// DB returns an existing dividend of 1.50
	existingEvents := []AssetEvent{
		{
			ID:          "evt-1",
			AssetID:     "asset-1",
			Type:        "Dividendo",
			GrossAmount: 1.50,
			CumDate:     exDate,
			PaymentDate: payDate,
		},
	}

	repo.On("GetAssetEventsByDate", mock.Anything, "asset-1", exDate).Return(existingEvents, nil)

	// Should update since 1.54 - 1.50 = 0.04 <= 0.05
	repo.On("UpdateAssetEventValueByID", mock.Anything, "evt-1", 1.54, 1.54, payDate).Return(nil)

	worker.SyncAllDividends(ctx)

	repo.AssertExpectations(t)
	ms.AssertExpectations(t)
}

func TestDividendWorker_SyncAllDividends_NoMatch(t *testing.T) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)

	worker := NewDividendWorker(repo, ms)
	ctx := context.Background()

	assets := []AssetCompact{
		{ID: "asset-1", Ticker: "PETR4.SA", AssetType: "STOCK_BR"},
	}

	repo.On("GetAllAssets", mock.Anything).Return(assets, nil)

	exDate := time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)
	payDate := time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC)

	scrapedEvents := []market.DividendEvent{
		{
			Date:        exDate,
			Type:        "Dividendo",
			Amount:      2.00,
			PaymentDate: payDate,
		},
	}

	ms.On("GetDividends", mock.Anything, "PETR4.SA", "STOCK_BR").Return(scrapedEvents, nil)

	// DB returns an existing dividend of 1.50
	existingEvents := []AssetEvent{
		{
			ID:          "evt-1",
			AssetID:     "asset-1",
			Type:        "Dividendo",
			GrossAmount: 1.50,
			CumDate:     exDate,
			PaymentDate: payDate,
		},
	}

	repo.On("GetAssetEventsByDate", mock.Anything, "asset-1", exDate).Return(existingEvents, nil)

	// Difference is 0.50 (> 0.05), so it should INSERT
	repo.On("UpsertAssetEvent", mock.Anything, mock.AnythingOfType("AssetEvent")).Return(nil)

	worker.SyncAllDividends(ctx)

	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "UpdateAssetEventValueByID")
	ms.AssertExpectations(t)
}

func TestDividendWorker_SyncAllDividends_ExactMatchSkip(t *testing.T) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)

	worker := NewDividendWorker(repo, ms)
	ctx := context.Background()

	assets := []AssetCompact{
		{ID: "asset-1", Ticker: "PETR4.SA", AssetType: "STOCK_BR"},
	}

	repo.On("GetAllAssets", mock.Anything).Return(assets, nil)

	exDate := time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)
	payDate := time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC)

	scrapedEvents := []market.DividendEvent{
		{
			Date:        exDate,
			Type:        "Dividendo",
			Amount:      1.50,
			PaymentDate: payDate,
		},
	}

	ms.On("GetDividends", mock.Anything, "PETR4.SA", "STOCK_BR").Return(scrapedEvents, nil)

	existingEvents := []AssetEvent{
		{
			ID:          "evt-1",
			AssetID:     "asset-1",
			Type:        "Dividendo",
			GrossAmount: 1.50,
			CumDate:     exDate,
			PaymentDate: payDate,
		},
	}

	repo.On("GetAssetEventsByDate", mock.Anything, "asset-1", exDate).Return(existingEvents, nil)

	// Should not update or upsert
	worker.SyncAllDividends(ctx)

	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "UpdateAssetEventValueByID")
	repo.AssertNotCalled(t, "UpsertAssetEvent")
	ms.AssertExpectations(t)
}

func TestDividendWorker_SyncAllDividends_ErrorScenariosAndBranches(t *testing.T) {
	t.Run("GetAllAssets Error", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		worker := NewDividendWorker(repo, ms)

		repo.On("GetAllAssets", mock.Anything).Return(([]AssetCompact)(nil), assert.AnError)
		worker.SyncAllDividends(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Empty Ticker and Market Error", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		worker := NewDividendWorker(repo, ms)

		assets := []AssetCompact{
			{ID: "empty-1", Ticker: ""},
			{ID: "err-1", Ticker: "VALE3.SA", AssetType: "STOCK_BR"},
		}
		repo.On("GetAllAssets", mock.Anything).Return(assets, nil)
		ms.On("GetDividends", mock.Anything, "VALE3.SA", "STOCK_BR").Return(([]market.DividendEvent)(nil), assert.AnError)

		worker.SyncAllDividends(context.Background())
		repo.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("GetAssetEventsByDate Error and Different Types and DB Write Errors", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		worker := NewDividendWorker(repo, ms)

		assets := []AssetCompact{
			{ID: "asset-1", Ticker: "ITUB4.SA", AssetType: "STOCK_BR"},
		}
		repo.On("GetAllAssets", mock.Anything).Return(assets, nil)

		exDate1 := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		exDate2 := time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC)
		exDate3 := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
		payDate := time.Date(2024, 4, 10, 0, 0, 0, 0, time.UTC)

		scrapedEvents := []market.DividendEvent{
			{Date: exDate1, Type: "Dividendo", Amount: 1.0, PaymentDate: payDate},
			{Date: exDate2, Type: "JCP", Amount: 2.0, PaymentDate: payDate},
			{Date: exDate3, Type: "Rendimento", Amount: 3.0, PaymentDate: payDate},
		}
		ms.On("GetDividends", mock.Anything, "ITUB4.SA", "STOCK_BR").Return(scrapedEvents, nil)

		// 1. GetAssetEventsByDate error for exDate1
		repo.On("GetAssetEventsByDate", mock.Anything, "asset-1", exDate1).Return(([]AssetEvent)(nil), assert.AnError)

		// 2. Existing event has different type ("Dividendo" != "JCP"), falls through to Insert, which fails
		repo.On("GetAssetEventsByDate", mock.Anything, "asset-1", exDate2).Return([]AssetEvent{
			{ID: "diff-type", Type: "Dividendo", GrossAmount: 2.0, CumDate: exDate2},
		}, nil)
		repo.On("UpsertAssetEvent", mock.Anything, mock.AnythingOfType("AssetEvent")).Return(assert.AnError)

		// 3. Update existing with matching type but different amount (diff <= 0.05), and Update fails
		repo.On("GetAssetEventsByDate", mock.Anything, "asset-1", exDate3).Return([]AssetEvent{
			{ID: "update-err", Type: "Rendimento", GrossAmount: 2.98, CumDate: exDate3, PaymentDate: payDate},
		}, nil)
		repo.On("UpdateAssetEventValueByID", mock.Anything, "update-err", 3.0, 3.0, payDate).Return(assert.AnError)

		worker.SyncAllDividends(context.Background())
		repo.AssertExpectations(t)
		ms.AssertExpectations(t)
	})
}

type MockDividendNotifier struct {
	mock.Mock
}

func (m *MockDividendNotifier) SendDividendPaymentAlert(chatID int64, userName, portfolioName string, items []DividendNotificationItem) error {
	args := m.Called(chatID, userName, portfolioName, items)
	return args.Error(0)
}

type MockDividendCalculator struct {
	mock.Mock
}

func (m *MockDividendCalculator) GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]CalculatedDividend, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]CalculatedDividend), args.Error(1)
}

type MockDividendDeduplicator struct {
	mock.Mock
}

func (m *MockDividendDeduplicator) IsDividendNotified(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockDividendDeduplicator) MarkDividendNotified(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

func TestDividendWorker_NotifyPaidDividends(t *testing.T) {
	ctx := context.Background()

	t.Run("Notifier or Calculator is nil", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		worker := NewDividendWorker(repo, ms)

		// Both nil
		worker.NotifyPaidDividends(ctx)

		// Calc nil, notifier set
		worker.notifier = new(MockDividendNotifier)
		worker.NotifyPaidDividends(ctx)

		// Notifier nil, calc set
		worker.notifier = nil
		worker.calc = new(MockDividendCalculator)
		worker.NotifyPaidDividends(ctx)

		repo.AssertNotCalled(t, "GetTelegramLinkedPortfolios")
	})

	t.Run("GetTelegramLinkedPortfolios error", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		calc := new(MockDividendCalculator)
		notifier := new(MockDividendNotifier)
		worker := NewDividendWorker(repo, ms).WithNotifier(calc, notifier)

		repo.On("GetTelegramLinkedPortfolios", mock.Anything).Return(([]LinkedPortfolio)(nil), assert.AnError).Once()

		worker.NotifyPaidDividends(ctx)
		repo.AssertExpectations(t)
		calc.AssertNotCalled(t, "GetPortfolioDividends")
	})

	t.Run("No linked portfolios", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		calc := new(MockDividendCalculator)
		notifier := new(MockDividendNotifier)
		worker := NewDividendWorker(repo, ms).WithNotifier(calc, notifier)

		repo.On("GetTelegramLinkedPortfolios", mock.Anything).Return([]LinkedPortfolio{}, nil).Once()

		worker.NotifyPaidDividends(ctx)
		repo.AssertExpectations(t)
		calc.AssertNotCalled(t, "GetPortfolioDividends")
	})

	t.Run("GetPortfolioDividends error and loadLocation fallback", func(t *testing.T) {
		origLoad := loadLocation
		loadLocation = func(name string) (*time.Location, error) {
			return nil, assert.AnError
		}
		defer func() { loadLocation = origLoad }()

		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		calc := new(MockDividendCalculator)
		notifier := new(MockDividendNotifier)
		worker := NewDividendWorker(repo, ms).WithNotifier(calc, notifier)

		linked := []LinkedPortfolio{
			{PortfolioID: "p1", PortfolioName: "Carteira 1", UserID: "u1", UserName: "Eduardo", TelegramChatID: 123},
		}
		repo.On("GetTelegramLinkedPortfolios", mock.Anything).Return(linked, nil).Once()
		calc.On("GetPortfolioDividends", mock.Anything, "p1", "u1").Return(([]CalculatedDividend)(nil), assert.AnError).Once()

		worker.NotifyPaidDividends(ctx)
		repo.AssertExpectations(t)
		calc.AssertExpectations(t)
		notifier.AssertNotCalled(t, "SendDividendPaymentAlert")
	})

	t.Run("Filter items (zero PaymentDate, not today, zero amount/quantity, already notified)", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		calc := new(MockDividendCalculator)
		notifier := new(MockDividendNotifier)
		dedup := new(MockDividendDeduplicator)
		worker := NewDividendWorker(repo, ms).WithNotifier(calc, notifier, dedup)

		linked := []LinkedPortfolio{
			{PortfolioID: "p1", PortfolioName: "Carteira 1", UserID: "u1", UserName: "Eduardo", TelegramChatID: 123},
		}
		repo.On("GetTelegramLinkedPortfolios", mock.Anything).Return(linked, nil).Once()

		loc, _ := time.LoadLocation("America/Sao_Paulo")
		now := time.Now().In(loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
		yesterday := today.AddDate(0, 0, -1)

		divs := []CalculatedDividend{
			// 1. Zero PaymentDate
			{Ticker: "ZERO", PaymentDate: time.Time{}, Quantity: 10, NetAmount: 10},
			// 2. Not today
			{Ticker: "YESTERDAY", PaymentDate: yesterday, Quantity: 10, NetAmount: 10},
			// 3. Zero Quantity (< 1e-6)
			{Ticker: "NO_QTY", PaymentDate: today, Quantity: 0.0000001, NetAmount: 10},
			// 4. Zero NetAmount (< 1e-6)
			{Ticker: "NO_AMOUNT", PaymentDate: today, Quantity: 10, NetAmount: 0.0000001},
			// 5. Already notified in deduplicator
			{AssetID: "a-dup", Ticker: "DUP", Type: "DIVIDENDO", PaymentDate: today, Quantity: 10, NetAmount: 10},
		}
		calc.On("GetPortfolioDividends", mock.Anything, "p1", "u1").Return(divs, nil).Once()
		dedup.On("IsDividendNotified", mock.Anything, mock.MatchedBy(func(k string) bool {
			return assert.ObjectsAreEqual(k, "notified_dividend:u1:p1:a-dup:DIVIDENDO:"+today.Format("2006-01-02"))
		})).Return(true, nil).Once()

		worker.NotifyPaidDividends(ctx)
		repo.AssertExpectations(t)
		calc.AssertExpectations(t)
		dedup.AssertExpectations(t)
		notifier.AssertNotCalled(t, "SendDividendPaymentAlert")
	})

	t.Run("Successful notification and mark notified", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		calc := new(MockDividendCalculator)
		notifier := new(MockDividendNotifier)
		dedup := new(MockDividendDeduplicator)
		worker := NewDividendWorker(repo, ms).WithNotifier(calc, notifier, dedup)

		linked := []LinkedPortfolio{
			{PortfolioID: "p1", PortfolioName: "Carteira 1", UserID: "u1", UserName: "Eduardo", TelegramChatID: 123},
		}
		repo.On("GetTelegramLinkedPortfolios", mock.Anything).Return(linked, nil).Once()

		loc, _ := time.LoadLocation("America/Sao_Paulo")
		now := time.Now().In(loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, loc)

		divs := []CalculatedDividend{
			{
				AssetID:        "a-petr4",
				Ticker:         "PETR4",
				AssetName:      "Petrobras",
				Type:           "DIVIDENDO",
				Quantity:       100.0,
				PerShareAmount: 1.50,
				GrossAmount:    150.0,
				NetAmount:      150.0,
				Currency:       "BRL",
				PaymentDate:    today,
			},
		}
		calc.On("GetPortfolioDividends", mock.Anything, "p1", "u1").Return(divs, nil).Once()

		dedupKey := "notified_dividend:u1:p1:a-petr4:DIVIDENDO:" + today.Format("2006-01-02")
		dedup.On("IsDividendNotified", mock.Anything, dedupKey).Return(false, nil).Once()

		notifier.On("SendDividendPaymentAlert", int64(123), "Eduardo", "Carteira 1", mock.MatchedBy(func(items []DividendNotificationItem) bool {
			return len(items) == 1 && items[0].Ticker == "PETR4" && items[0].NetAmount == 150.0
		})).Return(nil).Once()

		dedup.On("MarkDividendNotified", mock.Anything, dedupKey, 48*time.Hour).Return(nil).Once()

		worker.NotifyPaidDividends(ctx)
		repo.AssertExpectations(t)
		calc.AssertExpectations(t)
		notifier.AssertExpectations(t)
		dedup.AssertExpectations(t)
	})

	t.Run("Send error skips marking deduplicator", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		ms := new(MockMarketService)
		calc := new(MockDividendCalculator)
		notifier := new(MockDividendNotifier)
		dedup := new(MockDividendDeduplicator)
		worker := NewDividendWorker(repo, ms).WithNotifier(calc, notifier, dedup)

		linked := []LinkedPortfolio{
			{PortfolioID: "p1", PortfolioName: "Carteira 1", UserID: "u1", UserName: "Eduardo", TelegramChatID: 123},
		}
		repo.On("GetTelegramLinkedPortfolios", mock.Anything).Return(linked, nil).Once()

		loc, _ := time.LoadLocation("America/Sao_Paulo")
		now := time.Now().In(loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, loc)

		divs := []CalculatedDividend{
			{
				AssetID:        "a-vale3",
				Ticker:         "VALE3",
				Type:           "JCP",
				Quantity:       50.0,
				PerShareAmount: 2.0,
				NetAmount:      100.0,
				PaymentDate:    today,
			},
		}
		calc.On("GetPortfolioDividends", mock.Anything, "p1", "u1").Return(divs, nil).Once()

		dedupKey := "notified_dividend:u1:p1:a-vale3:JCP:" + today.Format("2006-01-02")
		dedup.On("IsDividendNotified", mock.Anything, dedupKey).Return(false, nil).Once()

		notifier.On("SendDividendPaymentAlert", int64(123), "Eduardo", "Carteira 1", mock.Anything).Return(assert.AnError).Once()

		worker.NotifyPaidDividends(ctx)
		repo.AssertExpectations(t)
		calc.AssertExpectations(t)
		notifier.AssertExpectations(t)
		dedup.AssertNotCalled(t, "MarkDividendNotified")
	})
}

func TestRedisDividendDeduplicator(t *testing.T) {
	ctx := context.Background()

	t.Run("Nil receiver or client", func(t *testing.T) {
		var dNil *RedisDividendDeduplicator
		isNotif, err := dNil.IsDividendNotified(ctx, "k1")
		assert.NoError(t, err)
		assert.False(t, isNotif)

		err = dNil.MarkDividendNotified(ctx, "k1", time.Hour)
		assert.NoError(t, err)

		dNoClient := &RedisDividendDeduplicator{rdb: nil}
		isNotif, err = dNoClient.IsDividendNotified(ctx, "k1")
		assert.NoError(t, err)
		assert.False(t, isNotif)

		err = dNoClient.MarkDividendNotified(ctx, "k1", time.Hour)
		assert.NoError(t, err)
	})

	t.Run("Redis client operations", func(t *testing.T) {
		db, mockRedis := redismock.NewClientMock()
		dedup := NewRedisDividendDeduplicator(db)

		// 1. IsDividendNotified - redis.Nil
		mockRedis.ExpectGet("k_nil").SetErr(redis.Nil)
		isNotif, err := dedup.IsDividendNotified(ctx, "k_nil")
		assert.NoError(t, err)
		assert.False(t, isNotif)

		// 2. IsDividendNotified - other error
		mockRedis.ExpectGet("k_err").SetErr(assert.AnError)
		isNotif, err = dedup.IsDividendNotified(ctx, "k_err")
		assert.Error(t, err)
		assert.False(t, isNotif)

		// 3. IsDividendNotified - returns "1"
		mockRedis.ExpectGet("k_one").SetVal("1")
		isNotif, err = dedup.IsDividendNotified(ctx, "k_one")
		assert.NoError(t, err)
		assert.True(t, isNotif)

		// 4. IsDividendNotified - returns "0"
		mockRedis.ExpectGet("k_zero").SetVal("0")
		isNotif, err = dedup.IsDividendNotified(ctx, "k_zero")
		assert.NoError(t, err)
		assert.False(t, isNotif)

		// 5. MarkDividendNotified - success
		mockRedis.ExpectSet("k_set", "1", 24*time.Hour).SetVal("OK")
		err = dedup.MarkDividendNotified(ctx, "k_set", 24*time.Hour)
		assert.NoError(t, err)

		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}
