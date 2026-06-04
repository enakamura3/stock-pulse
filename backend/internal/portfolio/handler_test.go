package portfolio

import (
	"mime/multipart"

	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPortfolioService struct {
	mock.Mock
}

func (m *MockPortfolioService) CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error) {
	args := m.Called(ctx, userID, name, baseCurrency)
	if args.Get(0) != nil {
		return args.Get(0).(*Portfolio), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) GetPortfolios(ctx context.Context, userID string) ([]Portfolio, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]Portfolio), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*Portfolio, []Position, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*Portfolio), args.Get(1).([]Position), args.Error(2)
	}
	return nil, nil, args.Error(2)
}

func (m *MockPortfolioService) AddTransaction(ctx context.Context, userID string, tx *Transaction) (*Transaction, error) {
	args := m.Called(ctx, userID, tx)
	if args.Get(0) != nil {
		return args.Get(0).(*Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error {
	return m.Called(ctx, txID, portfolioID, userID).Error(0)
}

func (m *MockPortfolioService) DeletePortfolio(ctx context.Context, id, userID string) error {
	return m.Called(ctx, id, userID).Error(0)
}

func (m *MockPortfolioService) GetPortfolioPerformance(ctx context.Context, portfolioID, userID, period string, filterTickers []string) ([]PerformancePoint, error) {
	args := m.Called(ctx, portfolioID, userID, period, filterTickers)
	if args.Get(0) != nil {
		return args.Get(0).([]PerformancePoint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) BackfillHistoricalPrices(ctx context.Context, assetID, ticker string) error {
	return m.Called(ctx, assetID, ticker).Error(0)
}

func (m *MockPortfolioService) GetPortfolioTransactions(ctx context.Context, portfolioID, userID string) ([]Transaction, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]CalculatedDividend, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]CalculatedDividend), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupHandlerTest() (*Handler, *MockPortfolioService) {
	s := new(MockPortfolioService)
	return NewHandler(s), s
}

func reqWithUserAndParams(req *http.Request, userID string, params map[string]string) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestHandler_CreatePortfolio(t *testing.T) {
	t.Run("Missing UserID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("POST", "/portfolios", nil)
		rec := httptest.NewRecorder()
		h.CreatePortfolio(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios", bytes.NewBufferString("invalid")), "u1", nil)
		rec := httptest.NewRecorder()
		h.CreatePortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("CreatePortfolio", mock.Anything, "u1", "My Port", "USD").Return(nil, errors.New("err"))
		body := `{"name": "My Port", "base_currency": "USD"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios", bytes.NewBufferString(body)), "u1", nil)
		rec := httptest.NewRecorder()
		h.CreatePortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("CreatePortfolio", mock.Anything, "u1", "My Port", "USD").Return(&Portfolio{ID: "p1"}, nil)
		body := `{"name": "My Port", "base_currency": "USD"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios", bytes.NewBufferString(body)), "u1", nil)
		rec := httptest.NewRecorder()
		h.CreatePortfolio(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestHandler_GetPortfolios(t *testing.T) {
	t.Run("Missing UserID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios", nil)
		rec := httptest.NewRecorder()
		h.GetPortfolios(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolios", mock.Anything, "u1").Return(nil, errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.GetPortfolios(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolios", mock.Anything, "u1").Return([]Portfolio{{ID: "p1"}}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.GetPortfolios(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_GetPortfolioDetails(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.GetPortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioDetails", mock.Anything, "p1", "u1").Return((*Portfolio)(nil), ([]Position)(nil), errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetPortfolio(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioDetails", mock.Anything, "p1", "u1").Return(&Portfolio{ID: "p1"}, []Position{}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetPortfolio(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_AddTransaction(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios//transactions", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString("invalid")), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.Anything).Return(nil, errors.New("err"))
		body := `{"ticker": "AAPL", "type": "BUY"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.Anything).Return(&Transaction{ID: "tx1"}, nil)
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestHandler_GetTransactions(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios//transactions", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.GetTransactions(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioTransactions", mock.Anything, "p1", "u1").Return(nil, errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/transactions", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetTransactions(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioTransactions", mock.Anything, "p1", "u1").Return([]Transaction{{ID: "tx1"}}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/transactions", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetTransactions(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_DeletePortfolio(t *testing.T) {
	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeletePortfolio", mock.Anything, "p1", "u1").Return(errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios/p1", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.DeletePortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeletePortfolio", mock.Anything, "p1", "u1").Return(nil)
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios/p1", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.DeletePortfolio(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_DeleteTransaction(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios//transactions/", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.DeleteTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeleteTransaction", mock.Anything, "tx1", "p1", "u1").Return(errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios/p1/transactions/tx1", nil), "u1", map[string]string{"id": "p1", "tx_id": "tx1"})
		rec := httptest.NewRecorder()
		h.DeleteTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeleteTransaction", mock.Anything, "tx1", "p1", "u1").Return(nil)
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios/p1/transactions/tx1", nil), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.DeleteTransaction(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_GetPortfolioPerformance(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios//performance", nil), "u1", nil)
		rec := httptest.NewRecorder()
		h.GetPerformance(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioPerformance", mock.Anything, "p1", "u1", "1M", mock.Anything).Return(([]PerformancePoint)(nil), errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/performance?period=1M", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetPerformance(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioPerformance", mock.Anything, "p1", "u1", "1M", mock.Anything).Return([]PerformancePoint{}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/performance?period=1M", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetPerformance(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

type failMarshal struct{}

func (f failMarshal) MarshalJSON() ([]byte, error) {
	return nil, errors.New("err")
}
func TestHandler_RespondWithJSON_Error(t *testing.T) {
	h, _ := setupHandlerTest()
	rec := httptest.NewRecorder()
	h.respondWithJSON(rec, http.StatusOK, failMarshal{})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_Unauthorized(t *testing.T) {
	h, _ := setupHandlerTest()
	req := httptest.NewRequest("GET", "/portfolios", nil)

	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"GetPortfolios", h.GetPortfolios},
		{"CreatePortfolio", h.CreatePortfolio},
		{"GetPortfolio", h.GetPortfolio},
		{"DeletePortfolio", h.DeletePortfolio},
		{"GetTransactions", h.GetTransactions},
		{"AddTransaction", h.AddTransaction},
		{"DeleteTransaction", h.DeleteTransaction},
		{"GetPerformance", h.GetPerformance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler(rec, req)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func (m *MockPortfolioService) UpdateTransaction(ctx context.Context, userID, portfolioID, txID string, tx *Transaction) error {
	return nil
}

func (m *MockPortfolioService) BulkAddTransactions(ctx context.Context, userID, portfolioID string, file multipart.File) (*BulkImportResult, error) {
	return &BulkImportResult{Success: 1, Errors: []string{}}, nil
}
