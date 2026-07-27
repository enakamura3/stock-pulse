package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDividendSource struct {
	mock.Mock
}

func (m *mockDividendSource) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockDividendSource) SupportedAssetTypes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *mockDividendSource) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	args := m.Called(ctx, ticker, assetType)
	return args.Get(0).([]DividendEvent), args.Error(1)
}

func TestMergeAndDedupDividends(t *testing.T) {
	t.Run("FII deduplication and type normalization", func(t *testing.T) {
		date1 := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
		dateOlder := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)

		saEvents := []DividendEvent{
			{Date: date1, PaymentDate: date1.AddDate(0, 0, 7), Amount: 0.101, Type: "Dividendo"},
		}
		fundEvents := []DividendEvent{
			{Date: date2, PaymentDate: date2.AddDate(0, 0, 7), Amount: 0.10, Type: "Rendimento"},
			{Date: dateOlder, PaymentDate: dateOlder.AddDate(0, 0, 7), Amount: 0.10, Type: "Rendimento"},
		}

		merged := mergeAndDedupDividends(saEvents, fundEvents, "FII")
		assert.Len(t, merged, 2)

		assert.Equal(t, 0.101, merged[0].Amount)
		assert.Equal(t, "Rendimento", merged[0].Type)
		assert.Equal(t, date1, merged[0].Date)

		assert.Equal(t, 0.10, merged[1].Amount)
		assert.Equal(t, "Rendimento", merged[1].Type)
		assert.Equal(t, dateOlder, merged[1].Date)
	})

	t.Run("Stock multiple dividends in same month", func(t *testing.T) {
		date1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

		saEvents := []DividendEvent{
			{Date: date1, Amount: 1.5, Type: "Dividendo"},
		}
		fundEvents := []DividendEvent{
			{Date: date1, Amount: 1.5, Type: "Dividendo"},
			{Date: date1, Amount: 2.0, Type: "Dividendo"},
		}

		merged := mergeAndDedupDividends(saEvents, fundEvents, "STOCK_BR")
		assert.Len(t, merged, 2)

		assert.Equal(t, 1.5, merged[0].Amount)
		assert.Equal(t, 2.0, merged[1].Amount)
	})
}

func TestIsMissingCurrentMonth(t *testing.T) {
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	t.Run("vazio retorna true", func(t *testing.T) {
		assert.True(t, isMissingCurrentMonth([]DividendEvent{}))
	})

	t.Run("apenas mês anterior retorna true", func(t *testing.T) {
		events := []DividendEvent{
			{Date: time.Date(lastMonth.Year(), lastMonth.Month(), 15, 0, 0, 0, 0, time.UTC)},
		}
		assert.True(t, isMissingCurrentMonth(events))
	})

	t.Run("mês atual presente retorna false", func(t *testing.T) {
		events := []DividendEvent{
			{Date: time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.UTC)},
		}
		assert.False(t, isMissingCurrentMonth(events))
	})
}

func TestEnrichWithFallback(t *testing.T) {
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	t.Run("FII: fallback preenche mês corrente ausente no base", func(t *testing.T) {
		baseEvents := []DividendEvent{
			{Date: time.Date(lastMonth.Year(), lastMonth.Month(), 18, 0, 0, 0, 0, time.UTC), Amount: 0.92, Type: "Rendimento"},
		}
		fallbackEvents := []DividendEvent{
			{Date: time.Date(now.Year(), now.Month(), 17, 0, 0, 0, 0, time.UTC), Amount: 0.92, Type: "Rendimento"},
			{Date: time.Date(lastMonth.Year(), lastMonth.Month(), 18, 0, 0, 0, 0, time.UTC), Amount: 0.92, Type: "Rendimento"},
		}

		result := enrichWithFallback(baseEvents, fallbackEvents, "FII")

		assert.Len(t, result, 2, "deve ter mês atual + mês anterior")
		assert.False(t, isMissingCurrentMonth(result), "mês atual deve estar presente após enriquecimento")
		assert.Equal(t, now.Month(), result[0].Date.Month())
	})

	t.Run("FII: fallback NÃO sobrescreve mês já coberto pelo base", func(t *testing.T) {
		baseCurrDate := time.Date(now.Year(), now.Month(), 18, 0, 0, 0, 0, time.UTC)
		baseEvents := []DividendEvent{
			{Date: baseCurrDate, Amount: 0.95, Type: "Rendimento"},
		}
		fallbackEvents := []DividendEvent{
			{Date: time.Date(now.Year(), now.Month(), 20, 0, 0, 0, 0, time.UTC), Amount: 0.92, Type: "Rendimento"},
		}

		result := enrichWithFallback(baseEvents, fallbackEvents, "FII")

		assert.Len(t, result, 1, "fallback não deve duplicar mês já coberto")
		assert.Equal(t, 0.95, result[0].Amount, "valor do base deve ser preservado")
	})

	t.Run("FII: type Dividendo do fallback é normalizado para Rendimento", func(t *testing.T) {
		baseEvents := []DividendEvent{
			{Date: time.Date(lastMonth.Year(), lastMonth.Month(), 18, 0, 0, 0, 0, time.UTC), Amount: 0.92, Type: "Rendimento"},
		}
		fallbackEvents := []DividendEvent{
			{Date: time.Date(now.Year(), now.Month(), 17, 0, 0, 0, 0, time.UTC), Amount: 0.92, Type: "Dividendo"},
		}

		result := enrichWithFallback(baseEvents, fallbackEvents, "FII")

		assert.Len(t, result, 2)
		assert.Equal(t, "Rendimento", result[0].Type, "tipo deve ser normalizado para Rendimento")
	})
}

func TestDividendGateway_GetDividends(t *testing.T) {
	primary := new(mockDividendSource)
	secondary := new(mockDividendSource)
	fallback := new(mockDividendSource)

	primary.On("Name").Return("primary")
	secondary.On("Name").Return("secondary")
	fallback.On("Name").Return("fallback")

	rdb, rmock := redismock.NewClientMock()
	gateway := NewDividendGateway(primary, secondary, nil, fallback, rdb, 12*time.Hour)

	now := time.Now()
	currMonthDate := time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.UTC)
	evWithCurrDate := DividendEvent{Type: "Dividendo", Date: currMonthDate}

	t.Run("Primary Success", func(t *testing.T) {
		rmock.ExpectGet("dividends:TICKER1").RedisNil()
		rmock.ExpectSet("dividends:TICKER1", mock.Anything, 12*time.Hour).SetVal("OK")

		primary.On("GetDividends", mock.Anything, "TICKER1", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()
		secondary.On("GetDividends", mock.Anything, "TICKER1", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		res, err := gateway.GetDividends(context.Background(), "TICKER1", "STOCK_BR")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("Primary and Secondary Fail, Fallback Success", func(t *testing.T) {
		rmock.ExpectGet("dividends:FAIL1").RedisNil()
		rmock.ExpectSet("dividends:FAIL1", mock.Anything, 12*time.Hour).SetVal("OK")

		primary.On("GetDividends", mock.Anything, "FAIL1", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		secondary.On("GetDividends", mock.Anything, "FAIL1", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		fallback.On("GetDividends", mock.Anything, "FAIL1", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		res, err := gateway.GetDividends(context.Background(), "FAIL1", "STOCK_BR")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("All Fail", func(t *testing.T) {
		rmock.ExpectGet("dividends:ALL_FAIL").RedisNil()

		primary.On("GetDividends", mock.Anything, "ALL_FAIL", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		secondary.On("GetDividends", mock.Anything, "ALL_FAIL", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		fallback.On("GetDividends", mock.Anything, "ALL_FAIL", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()

		res, err := gateway.GetDividends(context.Background(), "ALL_FAIL", "STOCK_BR")
		assert.Error(t, err)
		assert.Empty(t, res)
	})

	t.Run("Unknown Asset Type", func(t *testing.T) {
		res, err := gateway.GetDividends(context.Background(), "UNKNOWN", "WEIRD")
		assert.Error(t, err)
		assert.Empty(t, res)
	})
}
