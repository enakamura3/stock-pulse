package portfolio

import (
	"context"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
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
