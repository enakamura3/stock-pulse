package portfolio

import (
	"context"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
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
