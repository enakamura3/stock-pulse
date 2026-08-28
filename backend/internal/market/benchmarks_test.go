package market

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQuoteProviderForBenchmarks struct {
	mock.Mock
}

func (m *MockQuoteProviderForBenchmarks) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) != nil {
		return args.Get(0).(*Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuoteProviderForBenchmarks) SearchAssets(ctx context.Context, query string) ([]SearchResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) != nil {
		return args.Get(0).([]SearchResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuoteProviderForBenchmarks) GetHistoricalPrices(ctx context.Context, symbol string, rangePeriod string) ([]HistoricalPrice, error) {
	args := m.Called(ctx, symbol, rangePeriod)
	if args.Get(0) != nil {
		return args.Get(0).([]HistoricalPrice), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuoteProviderForBenchmarks) GetHistoricalPricesBetween(ctx context.Context, symbol string, period1, period2 int64) ([]HistoricalPrice, error) {
	args := m.Called(ctx, symbol, period1, period2)
	if args.Get(0) != nil {
		return args.Get(0).([]HistoricalPrice), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestService_GetBenchmarks(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Run("Cache Miss - Fetch from Provider and Cache", func(t *testing.T) {
		mockProv := new(MockQuoteProviderForBenchmarks)
		mockProv.On("GetQuote", mock.Anything, "^BVSP").Return(&Quote{
			Symbol:        "^BVSP",
			Name:          "Ibovespa",
			Price:         130000.0,
			Change:        1300.0,
			ChangePercent: 1.01,
			PreviousClose: 128700.0,
		}, nil)
		mockProv.On("GetQuote", mock.Anything, "^GSPC").Return(&Quote{
			Symbol:        "^GSPC",
			Name:          "S&P 500",
			Price:         5500.0,
			Change:        25.0,
			ChangePercent: 0.45,
			PreviousClose: 5475.0,
		}, nil)
		mockProv.On("GetQuote", mock.Anything, "BRL=X").Return(&Quote{
			Symbol:        "BRL=X",
			Name:          "", // Empty name to test fallback
			Price:         5.45,
			Change:        -0.02,
			ChangePercent: -0.37,
			PreviousClose: 5.47,
		}, nil)
		mockProv.On("GetQuote", mock.Anything, "IFIX.SA").Return(&Quote{
			Symbol:        "IFIX.SA",
			Name:          "IFIX",
			Price:         3350.0,
			Change:        5.0,
			ChangePercent: 0.15,
			PreviousClose: 3345.0,
		}, nil)

		svc := NewService(mockProv, rdb)
		benchmarks, err := svc.GetBenchmarks(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, benchmarks)
		assert.NotNil(t, benchmarks.IBOV)
		assert.Equal(t, "Ibovespa", benchmarks.IBOV.Name)
		assert.Equal(t, 130000.0, benchmarks.IBOV.Value)
		assert.Equal(t, 128700.0, benchmarks.IBOV.PreviousClose)

		assert.NotNil(t, benchmarks.SP500)
		assert.Equal(t, "S&P 500", benchmarks.SP500.Name)

		assert.NotNil(t, benchmarks.USDBRL)
		assert.Equal(t, "Dólar Comercial", benchmarks.USDBRL.Name) // Verified fallback

		assert.NotNil(t, benchmarks.IFIX)
		assert.Equal(t, "IFIX", benchmarks.IFIX.Name)

		// Verify that it is now cached in Redis
		cachedVal, err := rdb.Get(context.Background(), "benchmarks:summary:v1").Result()
		assert.NoError(t, err)
		assert.Contains(t, cachedVal, "Ibovespa")
	})

	t.Run("Cache Hit - Returns Cached Value", func(t *testing.T) {
		cachedBenchmarks := MarketBenchmarks{
			IBOV: &BenchmarkItem{
				Symbol:        "^BVSP",
				Name:          "Cached IBOV",
				Value:         131000.0,
				Change:        2000.0,
				ChangePercent: 1.55,
			},
			UpdatedAt: time.Now().UTC(),
		}
		data, _ := json.Marshal(cachedBenchmarks)
		err := rdb.Set(context.Background(), "benchmarks:summary:v1", string(data), 5*time.Minute).Err()
		assert.NoError(t, err)

		mockProv := new(MockQuoteProviderForBenchmarks)
		// Provider should NOT be called
		svc := NewService(mockProv, rdb)
		res, err := svc.GetBenchmarks(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, res.IBOV)
		assert.Equal(t, "Cached IBOV", res.IBOV.Name)
		assert.Equal(t, 131000.0, res.IBOV.Value)
	})

	t.Run("Partial Provider Failure - Resilient", func(t *testing.T) {
		mr.FlushAll()
		mockProv := new(MockQuoteProviderForBenchmarks)
		mockProv.On("GetQuote", mock.Anything, "^BVSP").Return(nil, errors.New("yahoo timeout"))
		mockProv.On("GetQuote", mock.Anything, "^GSPC").Return(&Quote{
			Symbol:        "^GSPC",
			Name:          "S&P 500",
			Price:         5500.0,
			Change:        25.0,
			ChangePercent: 0.45,
		}, nil)
		mockProv.On("GetQuote", mock.Anything, "BRL=X").Return(nil, errors.New("rate limit"))
		mockProv.On("GetQuote", mock.Anything, "IFIX.SA").Return(&Quote{
			Symbol:        "IFIX.SA",
			Name:          "IFIX",
			Price:         3350.0,
			Change:        5.0,
			ChangePercent: 0.15,
		}, nil)

		svc := NewService(mockProv, rdb)
		benchmarks, err := svc.GetBenchmarks(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, benchmarks)
		assert.Nil(t, benchmarks.IBOV) // failed target is nil
		assert.NotNil(t, benchmarks.SP500)
		assert.Nil(t, benchmarks.USDBRL)
		assert.NotNil(t, benchmarks.IFIX)
	})
}
