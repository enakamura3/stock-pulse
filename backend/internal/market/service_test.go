package market

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQuoteProvider struct {
	mock.Mock
}

func (m *MockQuoteProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) != nil {
		return args.Get(0).(*Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuoteProvider) SearchAssets(ctx context.Context, query string) ([]SearchResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) != nil {
		return args.Get(0).([]SearchResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuoteProvider) GetDividends(ctx context.Context, ticker string) ([]DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupServiceTest() (*Service, *MockQuoteProvider, *redis.Client, redismock.ClientMock) {
	mp := new(MockQuoteProvider)
	rdb, rmock := redismock.NewClientMock()
	s := NewService(mp, rdb)
	return s, mp, rdb, rmock
}

func TestService_GetQuote(t *testing.T) {
	t.Run("Invalid Symbol", func(t *testing.T) {
		s, _, _, _ := setupServiceTest()
		_, err := s.GetQuote(context.Background(), "   ")
		assert.ErrorContains(t, err, "inválido")
	})

	t.Run("Cache Hit", func(t *testing.T) {
		s, _, _, rmock := setupServiceTest()
		q := Quote{Symbol: "AAPL", Price: 150.0}
		b, _ := json.Marshal(q)

		rmock.ExpectGet("quote:AAPL").SetVal(string(b))

		quote, err := s.GetQuote(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Equal(t, 150.0, quote.Price)
		assert.NoError(t, rmock.ExpectationsWereMet())
	})

	t.Run("Cache Miss - Provider Error", func(t *testing.T) {
		s, mp, _, rmock := setupServiceTest()
		rmock.ExpectGet("quote:AAPL").RedisNil()

		mp.On("GetQuote", mock.Anything, "AAPL").Return(nil, errors.New("provider err"))

		_, err := s.GetQuote(context.Background(), "AAPL")
		assert.ErrorContains(t, err, "provider err")
		assert.NoError(t, rmock.ExpectationsWereMet())
	})

	t.Run("Cache Miss - Success and Set Cache", func(t *testing.T) {
		s, mp, _, rmock := setupServiceTest()
		rmock.ExpectGet("quote:AAPL").RedisNil()

		q := &Quote{Symbol: "AAPL", Price: 150.0}
		mp.On("GetQuote", mock.Anything, "AAPL").Return(q, nil)

		b, _ := json.Marshal(q)
		rmock.ExpectSet("quote:AAPL", b, 60*time.Second).SetVal("OK")

		quote, err := s.GetQuote(context.Background(), "aapl")
		assert.NoError(t, err)
		assert.Equal(t, 150.0, quote.Price)
		assert.NoError(t, rmock.ExpectationsWereMet())
	})
	
	t.Run("Cache Miss - Set Cache Error (Logs)", func(t *testing.T) {
		s, mp, _, rmock := setupServiceTest()
		rmock.ExpectGet("quote:AAPL").RedisNil()

		q := &Quote{Symbol: "AAPL", Price: 150.0}
		mp.On("GetQuote", mock.Anything, "AAPL").Return(q, nil)

		b, _ := json.Marshal(q)
		rmock.ExpectSet("quote:AAPL", b, 60*time.Second).SetErr(errors.New("redis err"))

		quote, err := s.GetQuote(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Equal(t, 150.0, quote.Price)
		assert.NoError(t, rmock.ExpectationsWereMet())
	})

	t.Run("Invalid Cache JSON", func(t *testing.T) {
		s, mp, _, rmock := setupServiceTest()
		rmock.ExpectGet("quote:AAPL").SetVal("invalid json")

		q := &Quote{Symbol: "AAPL", Price: 150.0}
		mp.On("GetQuote", mock.Anything, "AAPL").Return(q, nil)

		b, _ := json.Marshal(q)
		rmock.ExpectSet("quote:AAPL", b, 60*time.Second).SetVal("OK")

		quote, err := s.GetQuote(context.Background(), "AAPL")
		assert.NoError(t, err)
		assert.Equal(t, 150.0, quote.Price)
		assert.NoError(t, rmock.ExpectationsWereMet())
	})
}

func TestService_SearchAssets(t *testing.T) {
	t.Run("Empty Query", func(t *testing.T) {
		s, _, _, _ := setupServiceTest()
		res, err := s.SearchAssets(context.Background(), "   ")
		assert.NoError(t, err)
		assert.Len(t, res, 0)
	})

	t.Run("Success", func(t *testing.T) {
		s, mp, _, _ := setupServiceTest()
		mp.On("SearchAssets", mock.Anything, "AAPL").Return([]SearchResult{{Symbol: "AAPL"}}, nil)

		res, err := s.SearchAssets(context.Background(), "AAPL ")
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "AAPL", res[0].Symbol)
	})
}
