package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onigiri/stockpulse/backend/internal/market"
	"github.com/stretchr/testify/mock"
)

func TestDailyWorker_StartAndStop(t *testing.T) {
	repo := new(MockPortfolioRepo)
	mp := new(MockMarketService)

	worker := NewDailyWorker(repo, mp)

	// mock no assets to exit fast
	repo.On("GetAllAssets", mock.Anything).Return([]AssetCompact{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	
	// Start in a goroutine
	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	// Give it some time to start and run the immediate trigger
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop
	cancel()

	// Wait for start to exit
	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("Worker did not stop on context cancel")
	}
}

func TestDailyWorker_run(t *testing.T) {
	t.Run("GetAllAssets Error", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		mp := new(MockMarketService)
		worker := NewDailyWorker(repo, mp)

		repo.On("GetAllAssets", mock.Anything).Return(([]AssetCompact)(nil), errors.New("db err"))
		
		worker.run(context.Background())
		// should log and return, no panic
	})

	t.Run("Empty Assets", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		mp := new(MockMarketService)
		worker := NewDailyWorker(repo, mp)

		repo.On("GetAllAssets", mock.Anything).Return([]AssetCompact{}, nil)
		
		worker.run(context.Background())
	})

	t.Run("Success", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		mp := new(MockMarketService)
		worker := NewDailyWorker(repo, mp)

		assets := []AssetCompact{
			{ID: "a1", Ticker: "AAPL"},
		}
		repo.On("GetAllAssets", mock.Anything).Return(assets, nil)
		mp.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 150.0}, nil)
		repo.On("SaveDailyPrices", mock.Anything, "a1", mock.Anything).Return(nil)

		worker.run(context.Background())
		
		repo.AssertExpectations(t)
		mp.AssertExpectations(t)
	})

	t.Run("Market Provider Error", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		mp := new(MockMarketService)
		worker := NewDailyWorker(repo, mp)

		assets := []AssetCompact{
			{ID: "a1", Ticker: "AAPL"},
		}
		repo.On("GetAllAssets", mock.Anything).Return(assets, nil)
		mp.On("GetQuote", mock.Anything, "AAPL").Return((*market.Quote)(nil), errors.New("api err"))

		worker.run(context.Background())
		// shouldn't call SaveDailyPrices
		repo.AssertNotCalled(t, "SaveDailyPrices")
	})

	t.Run("Save Error", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		mp := new(MockMarketService)
		worker := NewDailyWorker(repo, mp)

		assets := []AssetCompact{
			{ID: "a1", Ticker: "AAPL"},
		}
		repo.On("GetAllAssets", mock.Anything).Return(assets, nil)
		mp.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 150.0}, nil)
		repo.On("SaveDailyPrices", mock.Anything, "a1", mock.Anything).Return(errors.New("db err"))

		worker.run(context.Background())
		// Should log and continue
	})
	
	t.Run("Context Cancelled during wait", func(t *testing.T) {
		repo := new(MockPortfolioRepo)
		mp := new(MockMarketService)
		worker := NewDailyWorker(repo, mp)

		assets := []AssetCompact{
			{ID: "a1", Ticker: "AAPL"},
		}
		repo.On("GetAllAssets", mock.Anything).Return(assets, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel
		
		worker.run(ctx)
		// Should return before calling GetQuote because of the 350ms wait and ctx.Done
		mp.AssertNotCalled(t, "GetQuote")
	})
}
