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
	"github.com/onigiri/stock-pulse/backend/internal/auth"
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

func (m *mockService) BulkAddTreasuryTransactions(ctx context.Context, portfolioID string, file multipart.File) (*BulkImportResult, error) {
	args := m.Called(ctx, portfolioID, file)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BulkImportResult), args.Error(1)
}

func (m *mockService) ExportTreasuryTransactions(ctx context.Context, portfolioID string) ([]byte, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
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

func authReq(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestHandler_FixedIncomeRoutes(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// 1. GET positions
	svc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]Position{}, nil).Once()
	req := authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/positions", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. GET performance
	svc.On("GetPortfolioPerformance", mock.Anything, "p1", "1M").Return([]PerformancePoint{}, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/performance?period=1M", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. GET monthly-yields
	svc.On("CalculateMonthlyYields", mock.Anything, "p1").Return([]MonthlyYield{}, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/monthly-yields", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. POST assets
	assetReq := Asset{Institution: "Itaú", Type: "CDB", DebtType: "PRE", Rate: 12.0}
	body, _ := json.Marshal(assetReq)
	svc.On("CreateAsset", mock.Anything, mock.Anything).Return(&Asset{ID: "a1"}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 5. DELETE asset
	repo.On("GetAssetByID", mock.Anything, "a1").Return(&Asset{ID: "a1", PortfolioID: "p1"}, nil).Once()
	repo.On("DeleteAsset", mock.Anything, "a1").Return(nil).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// 6. POST asset transaction
	txReq := Transaction{Type: "SUBSCRIPTION", Amount: 1000}
	body, _ = json.Marshal(txReq)
	repo.On("GetAssetByID", mock.Anything, "a1").Return(&Asset{ID: "a1", PortfolioID: "p1"}, nil).Once()
	svc.On("CreateTransaction", mock.Anything, mock.Anything).Return(&Transaction{ID: "t1"}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a1/transactions", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 7. PUT transaction
	svc.On("UpdateTransaction", mock.Anything, "p1", "t1", mock.Anything, mock.Anything).Return(nil).Once()
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/fixed-income/transactions/t1", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 8. DELETE transaction
	svc.On("DeleteTransaction", mock.Anything, "p1", "t1").Return(nil).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/transactions/t1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_TreasuryRoutes(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// 1. GET treasury positions
	svc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]TreasuryPosition{}, nil).Once()
	req := authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/positions", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. GET treasury transactions
	svc.On("GetTreasuryTransactions", mock.Anything, "p1").Return([]TreasuryTxRequest{}, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/transactions", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. POST treasury transaction
	txReq := TreasuryTxRequest{Ticker: "LFT", Type: "SUBSCRIPTION", Quantity: 5.0, UnitPrice: 100.0, TransactionDate: "2026-01-01"}
	body, _ := json.Marshal(txReq)
	svc.On("CreateTreasuryTransaction", mock.Anything, "p1", mock.Anything).Return(map[string]string{"id": "sub1"}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/transactions", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 4. PUT treasury transaction
	svc.On("UpdateTreasuryTransaction", mock.Anything, "p1", "t1", mock.Anything).Return(nil).Once()
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/treasury/transactions/t1", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. DELETE treasury transaction
	svc.On("DeleteTreasuryTransaction", mock.Anything, "p1", "t1").Return(nil).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/treasury/transactions/t1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// 6. GET treasury performance
	svc.On("GetTreasuryPerformance", mock.Anything, "p1").Return([]TreasuryPerfPoint{}, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/performance", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 7. GET treasury monthly-yields
	svc.On("GetTreasuryMonthlyYields", mock.Anything, "p1").Return([]MonthlyYield{}, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/monthly-yields", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ErrorBranches(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// GET positions error
	svc.On("GetPortfolioPositions", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req := authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/positions", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// POST asset invalid JSON
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets", bytes.NewBufferString("invalid json")), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// DELETE asset error
	repo.On("GetAssetByID", mock.Anything, "a1").Return(&Asset{ID: "a1", PortfolioID: "p1"}, nil).Once()
	repo.On("DeleteAsset", mock.Anything, "a1").Return(errors.New("err")).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// POST transaction invalid JSON
	repo.On("GetAssetByID", mock.Anything, "a1").Return(&Asset{ID: "a1", PortfolioID: "p1"}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a1/transactions", bytes.NewBufferString("invalid json")), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// POST treasury transaction error
	txReq := TreasuryTxRequest{Ticker: "LFT", Type: "SUBSCRIPTION"}
	body, _ := json.Marshal(txReq)
	svc.On("CreateTreasuryTransaction", mock.Anything, "p1", mock.Anything).Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/transactions", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_AuthMiddleware_Unauthorized(t *testing.T) {
	_, _, _, r := setupHandlerTest()

	// Requisição sem auth.UserIDKey no contexto deve retornar 401
	req := httptest.NewRequest("GET", "/portfolios/p1/fixed-income/positions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Não autorizado", res["error"])
}

func TestHandler_AuthMiddleware_ForbiddenOrNotFound(t *testing.T) {
	_, _, repo, r := setupHandlerTest()

	// Carteira p_other não pertence a u1
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p_other", "u1").Return(errors.New("carteira não encontrada ou permissão negada")).Once()

	req := authReq(httptest.NewRequest("GET", "/portfolios/p_other/fixed-income/positions", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Carteira não encontrada ou permissão negada", res["error"])
}

func TestHandler_DeleteAsset_CrossPortfolioBlocked(t *testing.T) {
	_, _, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// Ativo a2 pertence a p2, mas a rota chamada é sob p1
	repo.On("GetAssetByID", mock.Anything, "a2").Return(&Asset{ID: "a2", PortfolioID: "p2"}, nil).Once()

	req := authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a2", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Ativo não encontrado na carteira informada", res["error"])
}

func TestHandler_CreateTransaction_CrossPortfolioBlocked(t *testing.T) {
	_, _, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// Ativo a2 pertence a p2, tentativa de criar transação sob p1
	repo.On("GetAssetByID", mock.Anything, "a2").Return(&Asset{ID: "a2", PortfolioID: "p2"}, nil).Once()

	txReq := Transaction{Type: "SUBSCRIPTION", Amount: 500}
	body, _ := json.Marshal(txReq)
	req := authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a2/transactions", bytes.NewBuffer(body)), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, "Ativo não encontrado na carteira informada", res["error"])
}

func TestHandler_BulkImport_WithAuth(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// Criar multipart/form-data com CSV
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.csv")
	part.Write([]byte("date;name;type;amount;indexer;rate;maturity\n"))
	writer.Close()

	svc.On("BulkAddTransactions", mock.Anything, "p1", mock.Anything).Return(&BulkImportResult{Success: 1}, nil).Once()

	req := authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_AuthMiddleware_EmptyPortfolioID(t *testing.T) {
	h, _, _, _ := setupHandlerTest()
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	req := authReq(httptest.NewRequest("GET", "/some/path", nil), "u1")
	w := httptest.NewRecorder()
	h.portfolioAuthMiddleware(next).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, nextCalled)
}

func TestHandler_EmptyPortfolioID_Branches(t *testing.T) {
	h, _, _, _ := setupHandlerTest()
	req := httptest.NewRequest("GET", "/", nil)

	// getMonthlyYields
	w := httptest.NewRecorder()
	h.getMonthlyYields(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// getPositions
	w = httptest.NewRecorder()
	h.getPositions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// getPerformance
	w = httptest.NewRecorder()
	h.getPerformance(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// bulkImportTransactions
	w = httptest.NewRecorder()
	h.bulkImportTransactions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// getTreasuryPositions
	w = httptest.NewRecorder()
	h.getTreasuryPositions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// getTreasuryTransactions
	w = httptest.NewRecorder()
	h.getTreasuryTransactions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// createTreasuryTransaction
	w = httptest.NewRecorder()
	h.createTreasuryTransaction(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// getTreasuryPerformance
	w = httptest.NewRecorder()
	h.getTreasuryPerformance(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// updateTreasuryTransaction
	w = httptest.NewRecorder()
	h.updateTreasuryTransaction(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// deleteTreasuryTransaction
	w = httptest.NewRecorder()
	h.deleteTreasuryTransaction(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// getTreasuryMonthlyYields
	w = httptest.NewRecorder()
	h.getTreasuryMonthlyYields(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// bulkImportTreasuryTransactions
	w = httptest.NewRecorder()
	h.bulkImportTreasuryTransactions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// exportTreasuryTransactions
	w = httptest.NewRecorder()
	h.exportTreasuryTransactions(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NilSliceFallbacks(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// 1. getMonthlyYields returns nil
	svc.On("CalculateMonthlyYields", mock.Anything, "p1").Return(nil, nil).Once()
	req := authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/monthly-yields", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())

	// 2. getPositions returns nil
	svc.On("GetPortfolioPositions", mock.Anything, "p1").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/positions", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())

	// 3. getPerformance returns nil (and without period query param, defaults to "ALL")
	svc.On("GetPortfolioPerformance", mock.Anything, "p1", "ALL").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/performance", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())

	// 4. getTreasuryPositions returns nil
	svc.On("GetTreasuryPositions", mock.Anything, "p1").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/positions", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())

	// 5. getTreasuryTransactions returns nil
	svc.On("GetTreasuryTransactions", mock.Anything, "p1").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/transactions", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())

	// 6. getTreasuryPerformance returns nil
	svc.On("GetTreasuryPerformance", mock.Anything, "p1").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/performance", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())

	// 7. getTreasuryMonthlyYields returns nil
	svc.On("GetTreasuryMonthlyYields", mock.Anything, "p1").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/monthly-yields", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}

func TestHandler_AdditionalErrorBranches(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// 1. getMonthlyYields error
	svc.On("CalculateMonthlyYields", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req := authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/monthly-yields", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 2. getPerformance error
	svc.On("GetPortfolioPerformance", mock.Anything, "p1", "ALL").Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/fixed-income/performance", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 3. createAsset service error
	assetReq := Asset{Institution: "Itaú"}
	body, _ := json.Marshal(assetReq)
	svc.On("CreateAsset", mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets", bytes.NewBuffer(body)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 4. deleteAsset GetAssetByID error (404)
	repo.On("GetAssetByID", mock.Anything, "a_err").Return(nil, errors.New("db error")).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a_err", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 5. deleteAsset GetAssetByID returns nil (404)
	repo.On("GetAssetByID", mock.Anything, "a_nil").Return(nil, nil).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/assets/a_nil", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 6. createTransaction GetAssetByID error (404)
	repo.On("GetAssetByID", mock.Anything, "a_err").Return(nil, errors.New("db error")).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a_err/transactions", bytes.NewBufferString("{}")), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 7. createTransaction service error (500)
	repo.On("GetAssetByID", mock.Anything, "a1").Return(&Asset{ID: "a1", PortfolioID: "p1"}, nil).Once()
	svc.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/assets/a1/transactions", bytes.NewBufferString(`{"amount":100}`)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 8. updateTransaction invalid JSON (400)
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/fixed-income/transactions/t1", bytes.NewBufferString("bad json")), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 9. updateTransaction unauthorized error (403)
	svc.On("UpdateTransaction", mock.Anything, "p1", "t1", mock.Anything, mock.Anything).Return(errors.New("unauthorized action")).Once()
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/fixed-income/transactions/t1", bytes.NewBufferString(`{"amount":100}`)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 10. updateTransaction general error (500)
	svc.On("UpdateTransaction", mock.Anything, "p1", "t1", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/fixed-income/transactions/t1", bytes.NewBufferString(`{"amount":100}`)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 11. deleteTransaction unauthorized error (403)
	svc.On("DeleteTransaction", mock.Anything, "p1", "t1").Return(errors.New("unauthorized access")).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/transactions/t1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 12. deleteTransaction general error (500)
	svc.On("DeleteTransaction", mock.Anything, "p1", "t1").Return(errors.New("db error")).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/fixed-income/transactions/t1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 13. bulkImportTransactions: invalid multipart form (400)
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/bulk", bytes.NewBufferString("not multipart")), "u1")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid_boundary")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 14. bulkImportTransactions: missing 'file' field (400)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormField("other_field")
	part.Write([]byte("some data"))
	writer.Close()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 15. bulkImportTransactions: service error (500)
	buf.Reset()
	writer = multipart.NewWriter(&buf)
	partFile, _ := writer.CreateFormFile("file", "test.csv")
	partFile.Write([]byte("header\nval"))
	writer.Close()
	svc.On("BulkAddTransactions", mock.Anything, "p1", mock.Anything).Return(nil, errors.New("import error")).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 16. bulkImportTransactions: partial errors (206)
	buf.Reset()
	writer = multipart.NewWriter(&buf)
	partFile, _ = writer.CreateFormFile("file", "test.csv")
	partFile.Write([]byte("header\nval"))
	writer.Close()
	svc.On("BulkAddTransactions", mock.Anything, "p1", mock.Anything).Return(&BulkImportResult{
		Success: 1,
		Errors:  []string{"row 2 invalid"},
	}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/fixed-income/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPartialContent, w.Code)

	// 17. getTreasuryPositions error (500)
	svc.On("GetTreasuryPositions", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/positions", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 18. getTreasuryTransactions error (500)
	svc.On("GetTreasuryTransactions", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/transactions", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 19. createTreasuryTransaction invalid JSON (400)
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/transactions", bytes.NewBufferString("invalid json")), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 20. getTreasuryPerformance error (500)
	svc.On("GetTreasuryPerformance", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/performance", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 21. updateTreasuryTransaction invalid JSON (400)
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/treasury/transactions/t1", bytes.NewBufferString("bad json")), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 22. updateTreasuryTransaction unauthorized error (403)
	svc.On("UpdateTreasuryTransaction", mock.Anything, "p1", "t1", mock.Anything).Return(errors.New("unauthorized action")).Once()
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/treasury/transactions/t1", bytes.NewBufferString(`{"ticker":"LFT"}`)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 23. updateTreasuryTransaction general error (500)
	svc.On("UpdateTreasuryTransaction", mock.Anything, "p1", "t1", mock.Anything).Return(errors.New("db error")).Once()
	req = authReq(httptest.NewRequest("PUT", "/portfolios/p1/treasury/transactions/t1", bytes.NewBufferString(`{"ticker":"LFT"}`)), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 24. deleteTreasuryTransaction unauthorized error (403)
	svc.On("DeleteTreasuryTransaction", mock.Anything, "p1", "t1").Return(errors.New("unauthorized deletion")).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/treasury/transactions/t1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 25. deleteTreasuryTransaction general error (500)
	svc.On("DeleteTreasuryTransaction", mock.Anything, "p1", "t1").Return(errors.New("db error")).Once()
	req = authReq(httptest.NewRequest("DELETE", "/portfolios/p1/treasury/transactions/t1", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 26. getTreasuryMonthlyYields error (500)
	svc.On("GetTreasuryMonthlyYields", mock.Anything, "p1").Return(nil, errors.New("err")).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/monthly-yields", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_BulkImportTreasuryTransactions(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// 1. Invalid multipart
	req := authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/bulk", bytes.NewBufferString("not multipart")), "u1")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=bad_boundary")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 2. Missing file field
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("other", "val")
	writer.Close()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 3. Service error (500)
	buf.Reset()
	writer = multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "treasury.csv")
	part.Write([]byte("date;ticker;type;quantity;price\n"))
	writer.Close()
	svc.On("BulkAddTreasuryTransactions", mock.Anything, "p1", mock.Anything).Return(nil, errors.New("import error")).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 4. Success (200)
	buf.Reset()
	writer = multipart.NewWriter(&buf)
	part, _ = writer.CreateFormFile("file", "treasury.csv")
	part.Write([]byte("date;ticker;type;quantity;price\n"))
	writer.Close()
	svc.On("BulkAddTreasuryTransactions", mock.Anything, "p1", mock.Anything).Return(&BulkImportResult{Success: 2}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. Partial content (206)
	buf.Reset()
	writer = multipart.NewWriter(&buf)
	part, _ = writer.CreateFormFile("file", "treasury.csv")
	part.Write([]byte("date;ticker;type;quantity;price\n"))
	writer.Close()
	svc.On("BulkAddTreasuryTransactions", mock.Anything, "p1", mock.Anything).Return(&BulkImportResult{
		Success: 1,
		Errors:  []string{"row 2 invalid"},
	}, nil).Once()
	req = authReq(httptest.NewRequest("POST", "/portfolios/p1/treasury/bulk", &buf), "u1")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPartialContent, w.Code)
}

func TestHandler_ExportTreasuryTransactions(t *testing.T) {
	_, svc, repo, r := setupHandlerTest()
	repo.On("ValidatePortfolioOwnership", mock.Anything, "p1", "u1").Return(nil)

	// 1. Service error (500)
	svc.On("ExportTreasuryTransactions", mock.Anything, "p1").Return(nil, errors.New("export error")).Once()
	req := authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/export", nil), "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 2. Success (200)
	csvData := []byte("Date;Ticker;Type;Quantity;UnitPrice\n2026-01-15;Tesouro Selic 2029;SUBSCRIPTION;1;14000\n")
	svc.On("ExportTreasuryTransactions", mock.Anything, "p1").Return(csvData, nil).Once()
	req = authReq(httptest.NewRequest("GET", "/portfolios/p1/treasury/export", nil), "u1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment; filename=\"tesouro-direto-p1.csv\"")
	assert.Equal(t, csvData, w.Body.Bytes())
}

