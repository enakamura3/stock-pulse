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

	t.Run("Stock JCP and Dividendo reconciliation in base and secondary", func(t *testing.T) {
		date1 := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
		pay1 := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		pay2 := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

		// Base (e.g. B3) has duplicate with Dividendo then JCP on date1, and Dividendo on date2
		fundEvents := []DividendEvent{
			{Date: date1, PaymentDate: pay1, Amount: 0.02, Type: "Dividendo"},
			{Date: date1, PaymentDate: pay1, Amount: 0.02, Type: "JCP"},
			{Date: date2, PaymentDate: pay2, Amount: 0.05, Type: "Dividendo"},
		}
		// Secondary (e.g. Fundamentus) has JCP on date2 (with 1-day cum_date offset but same PaymentDate and Amount)
		saEvents := []DividendEvent{
			{Date: date2.AddDate(0, 0, 1), PaymentDate: pay2, Amount: 0.05, Type: "JCP"},
		}

		merged := mergeAndDedupDividends(saEvents, fundEvents, "STOCK_BR")
		assert.Len(t, merged, 2)
		assert.Equal(t, "JCP", merged[0].Type)
		assert.Equal(t, 0.05, merged[0].Amount)
		assert.Equal(t, "JCP", merged[1].Type)
		assert.Equal(t, 0.02, merged[1].Amount)
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
	gateway := NewDividendGateway(primary, secondary, fallback, fallback, rdb, 12*time.Hour)

	now := time.Now()
	currMonthDate := time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.UTC)
	evWithCurrDate := DividendEvent{Type: "Dividendo", Date: currMonthDate}

	t.Run("Primary Success", func(t *testing.T) {
		rmock.ExpectGet("dividends:v2:TICKER1").RedisNil()
		rmock.ExpectSet("dividends:v2:TICKER1", mock.Anything, 12*time.Hour).SetVal("OK")

		primary.On("GetDividends", mock.Anything, "TICKER1", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()
		secondary.On("GetDividends", mock.Anything, "TICKER1", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		res, err := gateway.GetDividends(context.Background(), "TICKER1", "STOCK_BR")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("Primary and Secondary Fail, Fallback Success", func(t *testing.T) {
		rmock.ExpectGet("dividends:v2:FAIL1").RedisNil()
		rmock.ExpectSet("dividends:v2:FAIL1", mock.Anything, 12*time.Hour).SetVal("OK")

		primary.On("GetDividends", mock.Anything, "FAIL1", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		secondary.On("GetDividends", mock.Anything, "FAIL1", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		fallback.On("GetDividends", mock.Anything, "FAIL1", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		res, err := gateway.GetDividends(context.Background(), "FAIL1", "STOCK_BR")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("All Fail", func(t *testing.T) {
		rmock.ExpectGet("dividends:v2:ALL_FAIL").RedisNil()

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

	t.Run("Primary and Secondary nil, Fallback only", func(t *testing.T) {
		rdb, rmock := redismock.NewClientMock()
		rmock.ExpectGet("dividends:v2:FALL_ONLY").RedisNil()
		rmock.ExpectSet("dividends:v2:FALL_ONLY", mock.Anything, 12*time.Hour).SetVal("OK")

		gw := NewDividendGateway(nil, nil, fallback, fallback, rdb, 12*time.Hour)
		fallback.On("GetDividends", mock.Anything, "FALL_ONLY", "CRYPTO").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		res, err := gw.GetDividends(context.Background(), "FALL_ONLY", "CRYPTO")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("Enrich with Fallback error", func(t *testing.T) {
		rdb, rmock := redismock.NewClientMock()
		rmock.ExpectGet("dividends:v2:ENRICH_ERR").RedisNil()
		rmock.ExpectSet("dividends:v2:ENRICH_ERR", mock.Anything, 12*time.Hour).SetVal("OK")

		oldEv := DividendEvent{Type: "Dividendo", Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}

		myPrimary := new(mockDividendSource)
		myFallback := new(mockDividendSource)
		myPrimary.On("Name").Return("primary")
		myFallback.On("Name").Return("fallback")

		myPrimary.On("GetDividends", mock.Anything, "ENRICH_ERR", "STOCK_BR").Return([]DividendEvent{oldEv}, nil).Once()
		myFallback.On("GetDividends", mock.Anything, "ENRICH_ERR", "STOCK_BR").Return([]DividendEvent{}, errors.New("enrich err")).Once()

		gw := NewDividendGateway(myPrimary, nil, myFallback, myFallback, rdb, 12*time.Hour)
		res, err := gw.GetDividends(context.Background(), "ENRICH_ERR", "STOCK_BR")
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Primary Error, Secondary Success", func(t *testing.T) {
		rdb, rmock := redismock.NewClientMock()
		rmock.ExpectGet("dividends:v2:SEC_SUCC").RedisNil()
		rmock.ExpectSet("dividends:v2:SEC_SUCC", mock.Anything, 12*time.Hour).SetVal("OK")

		myPrimary := new(mockDividendSource)
		mySecondary := new(mockDividendSource)
		myPrimary.On("Name").Return("primary")
		mySecondary.On("Name").Return("secondary")

		myPrimary.On("GetDividends", mock.Anything, "SEC_SUCC", "STOCK_BR").Return([]DividendEvent{}, errors.New("err")).Once()
		mySecondary.On("GetDividends", mock.Anything, "SEC_SUCC", "STOCK_BR").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		gw := NewDividendGateway(myPrimary, mySecondary, nil, nil, rdb, 12*time.Hour)
		res, err := gw.GetDividends(context.Background(), "SEC_SUCC", "STOCK_BR")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("FII Primary and Secondary Success", func(t *testing.T) {
		rdb, rmock := redismock.NewClientMock()
		rmock.ExpectGet("dividends:v2:FII_SUCC").RedisNil()
		rmock.ExpectSet("dividends:v2:FII_SUCC", mock.Anything, 12*time.Hour).SetVal("OK")

		myPrimary := new(mockDividendSource)
		mySecondary := new(mockDividendSource)
		myPrimary.On("Name").Return("primary")
		mySecondary.On("Name").Return("secondary")

		myPrimary.On("GetDividends", mock.Anything, "FII_SUCC", "FII").Return([]DividendEvent{evWithCurrDate}, nil).Once()
		mySecondary.On("GetDividends", mock.Anything, "FII_SUCC", "FII").Return([]DividendEvent{evWithCurrDate}, nil).Once()

		gw := NewDividendGateway(mySecondary, myPrimary, nil, nil, rdb, 12*time.Hour)
		res, err := gw.GetDividends(context.Background(), "FII_SUCC", "FII")
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})
}
