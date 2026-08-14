package fixedincome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/history"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) GetUnifiedTransactions(ctx context.Context, portfolioID, userID string) ([]history.UnifiedTransaction, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]history.UnifiedTransaction), args.Error(1)
}

func (m *mockService) CreateAsset(ctx context.Context, asset *Asset) (*Asset, error) {
	args := m.Called(ctx, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Asset), args.Error(1)
}

func (m *mockService) GetPortfolioPositions(ctx context.Context, portfolioID string) ([]Position, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Position), args.Error(1)
}

func (m *mockService) GetPortfolioPerformance(ctx context.Context, portfolioID string, period string) ([]PerformancePoint, error) {
	args := m.Called(ctx, portfolioID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]PerformancePoint), args.Error(1)
}

func (m *mockService) GetAssetPosition(ctx context.Context, assetID string) (*Position, error) {
	args := m.Called(ctx, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Position), args.Error(1)
}

func (m *mockService) CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	args := m.Called(ctx, tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Transaction), args.Error(1)
}

func (m *mockService) UpdateTransaction(ctx context.Context, portfolioID, txID string, tx *Transaction, maturityDate *time.Time) error {
	args := m.Called(ctx, portfolioID, txID, tx, maturityDate)
	return args.Error(0)
}

func (m *mockService) DeleteTransaction(ctx context.Context, portfolioID, txID string) error {
	args := m.Called(ctx, portfolioID, txID)
	return args.Error(0)
}

func (m *mockService) TriggerBackfill(ctx context.Context, indexer string, startDate time.Time) {
	m.Called(ctx, indexer, startDate)
}

func (m *mockService) CalculateMonthlyYields(ctx context.Context, portfolioID string) ([]MonthlyYield, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]MonthlyYield), args.Error(1)
}

func (m *mockService) BulkAddTransactions(ctx context.Context, portfolioID string, file multipart.File) (*BulkImportResult, error) {
	args := m.Called(ctx, portfolioID, file)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BulkImportResult), args.Error(1)
}

func (m *mockService) GetRawTransactions(ctx context.Context, portfolioID string) ([]Transaction, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Transaction), args.Error(1)
}

func (m *mockService) GetAssetsByPortfolio(ctx context.Context, portfolioID string) ([]Asset, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Asset), args.Error(1)
}

func (m *mockService) GetTreasuryPositions(ctx context.Context, portfolioID string) ([]TreasuryPosition, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryPosition), args.Error(1)
}

func (m *mockService) GetTreasuryTransactions(ctx context.Context, portfolioID string) ([]TreasuryTxRequest, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryTxRequest), args.Error(1)
}

func (m *mockService) CreateTreasuryTransaction(ctx context.Context, portfolioID string, req *TreasuryTxRequest) (interface{}, error) {
	args := m.Called(ctx, portfolioID, req)
	return args.Get(0), args.Error(1)
}

func (m *mockService) UpdateTreasuryTransaction(ctx context.Context, portfolioID, txID string, req *TreasuryTxRequest) error {
	args := m.Called(ctx, portfolioID, txID, req)
	return args.Error(0)
}

func (m *mockService) DeleteTreasuryTransaction(ctx context.Context, portfolioID, txID string) error {
	args := m.Called(ctx, portfolioID, txID)
	return args.Error(0)
}

func (m *mockService) GetTreasuryPerformance(ctx context.Context, portfolioID string) ([]TreasuryPerfPoint, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryPerfPoint), args.Error(1)
}

func (m *mockService) GetIndexRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	args := m.Called(ctx, indexer, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]IndexRate), args.Error(1)
}

func (m *mockService) GetTreasuryMonthlyYields(ctx context.Context, portfolioID string) ([]MonthlyYield, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]MonthlyYield), args.Error(1)
}

func setupHandlerTest() (*Handler, *mockService, *MockFullRepo, *chi.Mux) {
	svc := &mockService{}
	repo := &MockFullRepo{}
	h := NewHandler(svc, repo)
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return h, svc, repo, r
}

func TestHandler_FixedIncomeRoutes(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()

	// 1. GET positions
	svc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]Position{}, nil).Once()
	req := httptest.NewRequest("GET", "/portfolios/p1/fixed-income/positions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. GET performance
	svc.On("GetPortfolioPerformance", mock.Anything, "p1", "1M").Return([]PerformancePoint{}, nil).Once()
	req = httptest.NewRequest("GET", "/portfolios/p1/fixed-income/performance?period=1M", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. GET monthly-yields
	svc.On("CalculateMonthlyYields", mock.Anything, "p1").Return([]MonthlyYield{}, nil).Once()
	req = httptest.NewRequest("GET", "/portfolios/p1/fixed-income/monthly-yields", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. POST assets
	assetReq := Asset{Institution: "Itaú", Type: "CDB", DebtType: "PRE", Rate: 12.0}
	body, _ := json.Marshal(assetReq)
	svc.On("CreateAsset", mock.Anything, mock.Anything).Return(&Asset{ID: "a1"}, nil).Once()
	req = httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 5. DELETE asset
	repo.On("DeleteAsset", mock.Anything, "a1").Return(nil).Once()
	req = httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// 6. POST asset transaction
	txReq := Transaction{Type: "SUBSCRIPTION", Amount: 1000}
	body, _ = json.Marshal(txReq)
	svc.On("CreateTransaction", mock.Anything, mock.Anything).Return(&Transaction{ID: "t1"}, nil).Once()
	req = httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a1/transactions", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 7. PUT transaction
	svc.On("UpdateTransaction", mock.Anything, "p1", "t1", mock.Anything, mock.Anything).Return(nil).Once()
	req = httptest.NewRequest("PUT", "/portfolios/p1/fixed-income/transactions/t1", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 8. DELETE transaction
	svc.On("DeleteTransaction", mock.Anything, "p1", "t1").Return(nil).Once()
	req = httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/transactions/t1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_TreasuryRoutes(t *testing.T) {
	_, svc, _, r := setupHandlerTest()

	// 1. GET treasury positions
	svc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]TreasuryPosition{}, nil).Once()
	req := httptest.NewRequest("GET", "/portfolios/p1/treasury/positions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. GET treasury transactions
	svc.On("GetTreasuryTransactions", mock.Anything, "p1").Return([]TreasuryTxRequest{}, nil).Once()
	req = httptest.NewRequest("GET", "/portfolios/p1/treasury/transactions", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. POST treasury transaction
	txReq := TreasuryTxRequest{Ticker: "LFT", Type: "SUBSCRIPTION", Quantity: 5.0, UnitPrice: 100.0, TransactionDate: "2026-01-01"}
	body, _ := json.Marshal(txReq)
	svc.On("CreateTreasuryTransaction", mock.Anything, "p1", mock.Anything).Return(map[string]string{"id": "sub1"}, nil).Once()
	req = httptest.NewRequest("POST", "/portfolios/p1/treasury/transactions", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 4. PUT treasury transaction
	svc.On("UpdateTreasuryTransaction", mock.Anything, "p1", "t1", mock.Anything).Return(nil).Once()
	req = httptest.NewRequest("PUT", "/portfolios/p1/treasury/transactions/t1", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. DELETE treasury transaction
	svc.On("DeleteTreasuryTransaction", mock.Anything, "p1", "t1").Return(nil).Once()
	req = httptest.NewRequest("DELETE", "/portfolios/p1/treasury/transactions/t1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// 6. GET treasury performance
	svc.On("GetTreasuryPerformance", mock.Anything, "p1").Return([]TreasuryPerfPoint{}, nil).Once()
	req = httptest.NewRequest("GET", "/portfolios/p1/treasury/performance", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 7. GET treasury monthly-yields
	svc.On("GetTreasuryMonthlyYields", mock.Anything, "p1").Return([]MonthlyYield{}, nil).Once()
	req = httptest.NewRequest("GET", "/portfolios/p1/treasury/monthly-yields", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ErrorBranches(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()

	// GET positions error
	svc.On("GetPortfolioPositions", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req := httptest.NewRequest("GET", "/portfolios/p1/fixed-income/positions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// POST asset invalid JSON
	req = httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets", bytes.NewBufferString("invalid json"))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// DELETE asset error
	repo.On("DeleteAsset", mock.Anything, "a1").Return(errors.New("err")).Once()
	req = httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// POST transaction invalid JSON
	req = httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a1/transactions", bytes.NewBufferString("invalid json"))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// POST treasury transaction error
	txReq := TreasuryTxRequest{Ticker: "LFT", Type: "SUBSCRIPTION"}
	body, _ := json.Marshal(txReq)
	svc.On("CreateTreasuryTransaction", mock.Anything, "p1", mock.Anything).Return(nil, errors.New("err")).Once()
	req = httptest.NewRequest("POST", "/portfolios/p1/treasury/transactions", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
