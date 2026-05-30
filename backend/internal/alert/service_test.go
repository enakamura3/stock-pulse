package alert

import (
	"context"
	"errors"
	"testing"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type MockAlertRepo struct {
	mock.Mock
}

func (m *MockAlertRepo) GetAssetByTicker(ctx context.Context, ticker string) (string, error) {
	args := m.Called(ctx, ticker)
	return args.String(0), args.Error(1)
}

func (m *MockAlertRepo) CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error) {
	args := m.Called(ctx, ticker, name, assetType, currency)
	return args.String(0), args.Error(1)
}

func (m *MockAlertRepo) CreateAlert(ctx context.Context, a *Alert) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockAlertRepo) GetAlertsByUserID(ctx context.Context, userID string) ([]*Alert, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]*Alert), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAlertRepo) DeleteAlert(ctx context.Context, id string, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockAlertRepo) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]*Alert), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAlertRepo) MarkAlertTriggered(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAlertRepo) ToggleAlertStatus(ctx context.Context, id string, userID string) (string, error) {
	args := m.Called(ctx, id, userID)
	return args.String(0), args.Error(1)
}

type MockMarketService struct {
	mock.Mock
}

func (m *MockMarketService) GetQuote(ctx context.Context, symbol string) (*market.Quote, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) != nil {
		return args.Get(0).(*market.Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMarketService) SearchAssets(ctx context.Context, query string) ([]market.SearchResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) != nil {
		return args.Get(0).([]market.SearchResult), args.Error(1)
	}
	return nil, args.Error(1)
}

// Tests

func setupServiceTest() (*Service, *MockAlertRepo, *MockMarketService) {
	repo := new(MockAlertRepo)
	mp := new(MockMarketService)
	svc := NewService(repo, mp)
	return svc, repo, mp
}

func TestService_CreateAlert(t *testing.T) {
	t.Run("Invalid Params", func(t *testing.T) {
		svc, _, _ := setupServiceTest()

		_, err := svc.CreateAlert(context.Background(), "u1", "", 100.0, "ABOVE")
		assert.ErrorContains(t, err, "vazio")

		_, err = svc.CreateAlert(context.Background(), "u1", "AAPL", 0, "ABOVE")
		assert.ErrorContains(t, err, "zero")

		_, err = svc.CreateAlert(context.Background(), "u1", "AAPL", 100.0, "EQUAL")
		assert.ErrorContains(t, err, "ABOVE")
	})

	t.Run("Asset Exists Locally", func(t *testing.T) {
		svc, repo, _ := setupServiceTest()
		
		repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("a1", nil)
		repo.On("CreateAlert", mock.Anything, mock.MatchedBy(func(a *Alert) bool {
			return a.AssetID == "a1" && a.TargetPrice == 150.0 && a.Condition == "ABOVE"
		})).Return(nil)

		alert, err := svc.CreateAlert(context.Background(), "u1", "AAPL", 150.0, "ABOVE")
		assert.NoError(t, err)
		assert.Equal(t, "AAPL", alert.Ticker)
		repo.AssertExpectations(t)
	})

	t.Run("Asset Not Found Locally - Provider Error", func(t *testing.T) {
		svc, repo, mp := setupServiceTest()

		repo.On("GetAssetByTicker", mock.Anything, "INVALID").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "INVALID").Return((*market.Quote)(nil), errors.New("api error"))

		_, err := svc.CreateAlert(context.Background(), "u1", "INVALID", 150.0, "ABOVE")
		assert.ErrorContains(t, err, "provedor")
		repo.AssertExpectations(t)
		mp.AssertExpectations(t)
	})

	t.Run("Asset Created - Crypto", func(t *testing.T) {
		svc, repo, mp := setupServiceTest()

		repo.On("GetAssetByTicker", mock.Anything, "BTC-USD").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "BTC-USD").Return(&market.Quote{Name: "Bitcoin", Currency: "USD"}, nil)
		repo.On("CreateAsset", mock.Anything, "BTC-USD", "Bitcoin", "CRYPTO", "USD").Return("a1", nil)
		repo.On("CreateAlert", mock.Anything, mock.Anything).Return(nil)

		_, err := svc.CreateAlert(context.Background(), "u1", "BTC-USD", 50000.0, "ABOVE")
		assert.NoError(t, err)
		repo.AssertExpectations(t)
		mp.AssertExpectations(t)
	})

	t.Run("Asset Created - US Equity", func(t *testing.T) {
		svc, repo, mp := setupServiceTest()

		repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Name: "Apple", Currency: "USD"}, nil)
		repo.On("CreateAsset", mock.Anything, "AAPL", "Apple", "EQUITY_US", "USD").Return("a1", nil)
		repo.On("CreateAlert", mock.Anything, mock.Anything).Return(nil)

		_, err := svc.CreateAlert(context.Background(), "u1", "AAPL", 150.0, "ABOVE")
		assert.NoError(t, err)
	})

	t.Run("Asset Created - General Equity", func(t *testing.T) {
		svc, repo, mp := setupServiceTest()

		repo.On("GetAssetByTicker", mock.Anything, "PETR4.SA").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "PETR4.SA").Return(&market.Quote{Name: "Petrobras", Currency: "BRL"}, nil)
		repo.On("CreateAsset", mock.Anything, "PETR4.SA", "Petrobras", "EQUITY", "BRL").Return("a1", nil)
		repo.On("CreateAlert", mock.Anything, mock.Anything).Return(nil)

		_, err := svc.CreateAlert(context.Background(), "u1", "PETR4.SA", 30.0, "ABOVE")
		assert.NoError(t, err)
	})

	t.Run("Asset Creation DB Error", func(t *testing.T) {
		svc, repo, mp := setupServiceTest()

		repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Name: "Apple", Currency: "USD"}, nil)
		repo.On("CreateAsset", mock.Anything, "AAPL", "Apple", "EQUITY_US", "USD").Return("", errors.New("db err"))

		_, err := svc.CreateAlert(context.Background(), "u1", "AAPL", 150.0, "ABOVE")
		assert.ErrorContains(t, err, "banco de dados")
	})

	t.Run("Alert Creation DB Error", func(t *testing.T) {
		svc, repo, _ := setupServiceTest()

		repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("a1", nil)
		repo.On("CreateAlert", mock.Anything, mock.Anything).Return(errors.New("db err"))

		_, err := svc.CreateAlert(context.Background(), "u1", "AAPL", 150.0, "ABOVE")
		assert.ErrorContains(t, err, "banco")
	})
}

func TestService_GetAlerts(t *testing.T) {
	t.Run("Invalid User", func(t *testing.T) {
		svc, _, _ := setupServiceTest()
		_, err := svc.GetAlerts(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		svc, repo, _ := setupServiceTest()
		alerts := []*Alert{{ID: "a1"}}
		repo.On("GetAlertsByUserID", mock.Anything, "u1").Return(alerts, nil)

		res, err := svc.GetAlerts(context.Background(), "u1")
		assert.NoError(t, err)
		assert.Equal(t, alerts, res)
	})
}

func TestService_DeleteAlert(t *testing.T) {
	t.Run("Invalid Params", func(t *testing.T) {
		svc, _, _ := setupServiceTest()
		err := svc.DeleteAlert(context.Background(), "", "u1")
		assert.Error(t, err)
		err = svc.DeleteAlert(context.Background(), "a1", "")
		assert.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		svc, repo, _ := setupServiceTest()
		repo.On("DeleteAlert", mock.Anything, "a1", "u1").Return(nil)

		err := svc.DeleteAlert(context.Background(), "a1", "u1")
		assert.NoError(t, err)
	})
}

func TestService_ToggleAlert(t *testing.T) {
	t.Run("Invalid Params", func(t *testing.T) {
		svc, _, _ := setupServiceTest()
		_, err := svc.ToggleAlert(context.Background(), "", "u1")
		assert.Error(t, err)
		_, err = svc.ToggleAlert(context.Background(), "a1", "")
		assert.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		svc, repo, _ := setupServiceTest()
		repo.On("ToggleAlertStatus", mock.Anything, "a1", "u1").Return("DISABLED", nil)

		status, err := svc.ToggleAlert(context.Background(), "a1", "u1")
		assert.NoError(t, err)
		assert.Equal(t, "DISABLED", status)
	})
}

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string) ([]market.DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]market.DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}
