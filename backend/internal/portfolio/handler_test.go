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
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
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

func (m *MockPortfolioService) SetDefaultPortfolio(ctx context.Context, portfolioID, userID string) error {
	return m.Called(ctx, portfolioID, userID).Error(0)
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

func (m *MockPortfolioService) GetAssetMetadata(ctx context.Context, ticker string) (*AssetMetadata, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*AssetMetadata), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) ListAssetsAdmin(ctx context.Context) ([]AssetMetadata, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]AssetMetadata), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) UpdateAssetTypeAdmin(ctx context.Context, assetID, assetType string) error {
	return m.Called(ctx, assetID, assetType).Error(0)
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

	t.Run("Payload Too Large", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"name": "My Long Long Portfolio Name", "base_currency": "USD"}`
		rec := httptest.NewRecorder()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios", bytes.NewBufferString(body)), "u1", nil)
		req.Body = http.MaxBytesReader(rec, req.Body, 10)
		h.CreatePortfolio(rec, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Contains(t, rec.Body.String(), "limite máximo")
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
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/", nil), "u1", map[string]string{"id": ""})
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

	t.Run("Payload Too Large", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		req.Body = http.MaxBytesReader(rec, req.Body, 10)
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Contains(t, rec.Body.String(), "limite máximo")
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.Anything).Return((*Transaction)(nil), errors.New("err"))
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Negative Fee", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "fee": -5}`
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

	t.Run("Invalid Type", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "INVALID", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid Quantity", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 0, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid Asset Type", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "asset_type": "INVALID_TYPE", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Tipo de ativo inválido")
	})

	t.Run("Success with Valid Asset Type", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.MatchedBy(func(tx *Transaction) bool {
			return tx.AssetType == "FII"
		})).Return(&Transaction{ID: "tx-fii"}, nil)

		body := `{"ticker": "HGLG11.SA", "type": "BUY", "asset_type": "FII", "quantity": 10, "unit_price": 160}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("Success Split/Bonus Zeroes Fee", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.MatchedBy(func(tx *Transaction) bool {
			return tx.Fee == 0
		})).Return(&Transaction{ID: "tx-split"}, nil)

		body := `{"ticker": "AAPL", "type": "SPLIT", "quantity": 10, "unit_price": 0, "fee": 15}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("Invalid ExecutedAt", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "executed_at": "invalid-date"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Data de execução inválida")
	})

	t.Run("Future ExecutedAt", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "executed_at": "2099-01-01"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Data de execução não pode ser futura")
	})

	t.Run("RFC3339 ExecutedAt", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.Anything).Return(&Transaction{ID: "tx-rfc"}, nil)
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "executed_at": "2024-01-15T12:00:00Z"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.AddTransaction(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestHandler_GetTransactions(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios//transactions", nil), "u1", map[string]string{"id": ""})
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
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios/", nil), "u1", map[string]string{"id": ""})
		rec := httptest.NewRecorder()
		h.DeletePortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

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
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios//transactions/", nil), "u1", map[string]string{"id": "", "txId": ""})
		rec := httptest.NewRecorder()
		h.DeleteTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeleteTransaction", mock.Anything, "tx1", "p1", "u1").Return(errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("DELETE", "/portfolios/p1/transactions/tx1", nil), "u1", map[string]string{"id": "p1", "txId": "tx1"})
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
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios//performance", nil), "u1", map[string]string{"id": ""})
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
		s.On("GetPortfolioPerformance", mock.Anything, "p1", "u1", "ALL", []string(nil)).Return([]PerformancePoint{}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/performance", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetPerformance(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("With Tickers", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioPerformance", mock.Anything, "p1", "u1", "1M", []string{"AAPL", "GOOG"}).Return([]PerformancePoint{}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/performance?period=1M&tickers=AAPL,GOOG", nil), "u1", map[string]string{"id": "p1"})
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
	rec := httptest.NewRecorder()
	httputils.RespondWithJSON(rec, http.StatusOK, failMarshal{})
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
	args := m.Called(ctx, userID, portfolioID, txID, tx)
	return args.Error(0)
}

func (m *MockPortfolioService) ConvertAsset(ctx context.Context, userID, portfolioID string, req ConversionRequest) (*ConversionResult, error) {
	args := m.Called(ctx, userID, portfolioID, req)
	if args.Get(0) != nil {
		return args.Get(0).(*ConversionResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) BulkAddTransactions(ctx context.Context, userID, portfolioID string, file multipart.File) (*BulkImportResult, error) {
	args := m.Called(ctx, userID, portfolioID, file)
	if args.Get(0) != nil {
		return args.Get(0).(*BulkImportResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) GetFixedIncomeService() fixedincome.Service {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(fixedincome.Service)
	}
	return nil
}

func TestHandler_CtxOrDefault(t *testing.T) {
	ctx := ctxOrDefault(nil)
	assert.NotNil(t, ctx)
}

func TestHandler_UpdateTransaction(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios//transactions/", nil), "u1", map[string]string{"id": "", "txId": ""})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString("{}"))
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString("invalid")), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Payload Too Large", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		req.Body = http.MaxBytesReader(rec, req.Body, 10)
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Contains(t, rec.Body.String(), "limite máximo")
	})

	t.Run("Invalid Type", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "INVALID", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid Quantity", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 0, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateTransaction", mock.Anything, "u1", "p1", "tx1", mock.Anything).Return(errors.New("err"))
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Insufficient Sell Balance", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateTransaction", mock.Anything, "u1", "p1", "tx1", mock.Anything).Return(errors.New("saldo insuficiente do ativo para realizar a venda"))
		body := `{"ticker": "ORIG11.SA", "type": "SELL", "quantity": 100, "unit_price": 10.5}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "saldo insuficiente")
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateTransaction", mock.Anything, "u1", "p1", "tx1", mock.Anything).Return(nil)
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "executed_at": "2024-01-15"}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid ExecutedAt", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "executed_at": "invalid-date"}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Data de execução inválida")
	})

	t.Run("Future ExecutedAt", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "executed_at": "2099-01-01"}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Data de execução não pode ser futura")
	})

	t.Run("Invalid Asset Type", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "asset_type": "INVALID_TYPE", "quantity": 10, "unit_price": 150}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Tipo de ativo inválido")
	})

	t.Run("Success with Valid Asset Type", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateTransaction", mock.Anything, "u1", "p1", "tx1", mock.MatchedBy(func(tx *Transaction) bool {
			return tx.AssetType == "STOCK_BR"
		})).Return(nil)
		body := `{"ticker": "PETR4.SA", "type": "BUY", "asset_type": "STOCK_BR", "quantity": 10, "unit_price": 38}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Negative Fee", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"ticker": "AAPL", "type": "BUY", "quantity": 10, "unit_price": 150, "fee": -5}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Taxa/Corretagem não pode ser negativa")
	})

	t.Run("Success Split/Bonus Zeroes Fee", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateTransaction", mock.Anything, "u1", "p1", "tx1", mock.MatchedBy(func(tx *Transaction) bool {
			return tx.Fee == 0
		})).Return(nil)
		body := `{"ticker": "AAPL", "type": "SPLIT", "quantity": 10, "unit_price": 0, "fee": 20}`
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx1", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1", "txId": "tx1"})
		rec := httptest.NewRecorder()
		h.UpdateTransaction(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_BulkImportTransactions(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios//bulk", nil), "u1", map[string]string{"id": ""})
		rec := httptest.NewRecorder()
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("POST", "/portfolios/p1/bulk", nil)
		rec := httptest.NewRecorder()
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid Multipart", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/bulk", bytes.NewBufferString("invalid body")), "u1", map[string]string{"id": "p1"})
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
		rec := httptest.NewRecorder()
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Wrong File Name", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("wrong_file", "test.csv")
		part.Write([]byte("ticker,type\nAAPL,BUY"))
		writer.Close()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/bulk", body), "u1", map[string]string{"id": "p1"})
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		part.Write([]byte("ticker,type\nAAPL,BUY"))
		writer.Close()

		s.On("BulkAddTransactions", mock.Anything, "u1", "p1", mock.Anything).Return((*BulkImportResult)(nil), errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/bulk", body), "u1", map[string]string{"id": "p1"})
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Payload Too Large", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		_, _ = part.Write([]byte("ticker,type\nAAPL,BUY"))
		_ = writer.Close()

		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/bulk", body), "u1", map[string]string{"id": "p1"})
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		req.Body = http.MaxBytesReader(rec, req.Body, 10)
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Contains(t, rec.Body.String(), "Arquivo excede o limite máximo permitido")
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		part.Write([]byte("ticker,type\nAAPL,BUY"))
		writer.Close()

		s.On("BulkAddTransactions", mock.Anything, "u1", "p1", mock.Anything).Return(&BulkImportResult{Success: 1}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/bulk", body), "u1", map[string]string{"id": "p1"})
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		h.BulkImportTransactions(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_GetDividends(t *testing.T) {
	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios//dividends", nil), "u1", map[string]string{"id": ""})
		rec := httptest.NewRecorder()
		h.GetDividends(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/p1/dividends", nil)
		rec := httptest.NewRecorder()
		h.GetDividends(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioDividends", mock.Anything, "p1", "u1").Return(([]CalculatedDividend)(nil), errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/dividends", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetDividends(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetPortfolioDividends", mock.Anything, "p1", "u1").Return([]CalculatedDividend{}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/portfolios/p1/dividends", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.GetDividends(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_SetDefaultPortfolio(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("PUT", "/portfolios/p1/default", nil)
		rec := httptest.NewRecorder()
		h.SetDefaultPortfolio(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios//default", nil), "u1", map[string]string{"id": ""})
		rec := httptest.NewRecorder()
		h.SetDefaultPortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("SetDefaultPortfolio", mock.Anything, "p1", "u1").Return(errors.New("err"))
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/default", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.SetDefaultPortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("SetDefaultPortfolio", mock.Anything, "p1", "u1").Return(nil)
		req := reqWithUserAndParams(httptest.NewRequest("PUT", "/portfolios/p1/default", nil), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.SetDefaultPortfolio(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_GetAssetMetadata(t *testing.T) {
	t.Run("Missing Ticker", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/assets/", nil), "u1", map[string]string{"ticker": ""})
		rec := httptest.NewRecorder()
		h.GetAssetMetadata(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetAssetMetadata", mock.Anything, "INVALID").Return((*AssetMetadata)(nil), errors.New("not found"))
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/assets/INVALID", nil), "u1", map[string]string{"ticker": "INVALID"})
		rec := httptest.NewRecorder()
		h.GetAssetMetadata(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetAssetMetadata", mock.Anything, "PETR4.SA").Return(&AssetMetadata{
			ID: "a1", Ticker: "PETR4.SA", Name: "Petrobras", AssetType: "STOCK_BR", Currency: "BRL",
		}, nil)
		req := reqWithUserAndParams(httptest.NewRequest("GET", "/assets/PETR4.SA", nil), "u1", map[string]string{"ticker": "PETR4.SA"})
		rec := httptest.NewRecorder()
		h.GetAssetMetadata(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "STOCK_BR")
	})
}

func TestHandler_ListAssetsAdmin(t *testing.T) {
	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("ListAssetsAdmin", mock.Anything).Return(([]AssetMetadata)(nil), errors.New("db error"))
		req := httptest.NewRequest("GET", "/admin/assets", nil)
		rec := httptest.NewRecorder()
		h.ListAssetsAdmin(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("ListAssetsAdmin", mock.Anything).Return([]AssetMetadata{
			{ID: "a1", Ticker: "TAEE11.SA", Name: "Taesa", AssetType: "STOCK_BR", Currency: "BRL"},
		}, nil)
		req := httptest.NewRequest("GET", "/admin/assets", nil)
		rec := httptest.NewRecorder()
		h.ListAssetsAdmin(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "TAEE11.SA")
	})
}

func TestHandler_UpdateAssetTypeAdmin(t *testing.T) {
	t.Run("Missing Asset ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("PATCH", "/admin/assets/", bytes.NewBufferString(`{"asset_type":"STOCK_BR"}`)), "", map[string]string{"id": ""})
		rec := httptest.NewRecorder()
		h.UpdateAssetTypeAdmin(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("PATCH", "/admin/assets/a1", bytes.NewBufferString("invalid")), "", map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()
		h.UpdateAssetTypeAdmin(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Payload Too Large", func(t *testing.T) {
		h, _ := setupHandlerTest()
		body := `{"asset_type": "STOCK_BR_LONG_PAYLOAD"}`
		rec := httptest.NewRecorder()
		req := reqWithUserAndParams(httptest.NewRequest("PATCH", "/admin/assets/a1", bytes.NewBufferString(body)), "", map[string]string{"id": "a1"})
		req.Body = http.MaxBytesReader(rec, req.Body, 5)
		h.UpdateAssetTypeAdmin(rec, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("Invalid Asset Type", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("PATCH", "/admin/assets/a1", bytes.NewBufferString(`{"asset_type":"INVALID_TYPE"}`)), "", map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()
		h.UpdateAssetTypeAdmin(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateAssetTypeAdmin", mock.Anything, "a1", "STOCK_BR").Return(errors.New("db error"))
		req := reqWithUserAndParams(httptest.NewRequest("PATCH", "/admin/assets/a1", bytes.NewBufferString(`{"asset_type":"stock_br"}`)), "", map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()
		h.UpdateAssetTypeAdmin(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("UpdateAssetTypeAdmin", mock.Anything, "a1", "STOCK_BR").Return(nil)
		req := reqWithUserAndParams(httptest.NewRequest("PATCH", "/admin/assets/a1", bytes.NewBufferString(`{"asset_type":"stock_br"}`)), "", map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()
		h.UpdateAssetTypeAdmin(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "STOCK_BR")
	})
}

func TestHandler_ConvertAsset_And_ConversionTxTypes(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("POST", "/portfolios/p1/conversions", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		h.ConvertAsset(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing Portfolio ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios//conversions", nil), "u1", map[string]string{"id": ""})
		rec := httptest.NewRecorder()
		h.ConvertAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid JSON and Payload Too Large", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/conversions", bytes.NewBufferString("invalid")), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.ConvertAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		rec2 := httptest.NewRecorder()
		req2 := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/conversions", bytes.NewBufferString(`{"source_ticker":"ORIG11.SA"}`)), "u1", map[string]string{"id": "p1"})
		req2.Body = http.MaxBytesReader(rec2, req2.Body, 5)
		h.ConvertAsset(rec2, req2)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec2.Code)
	})

	t.Run("Validations", func(t *testing.T) {
		h, _ := setupHandlerTest()

		cases := []struct {
			name string
			body string
		}{
			{"InvalidTargetAssetType", `{"source_ticker":"ORIG11.SA","target_ticker":"DEST11.SA","target_asset_type":"INVALID","source_quantity":10,"target_quantity":8,"target_unit_price":12.50}`},
			{"MissingTickers", `{"source_ticker":"","target_ticker":"DEST11.SA","source_quantity":10,"target_quantity":8,"target_unit_price":12.50}`},
			{"InvalidQuantitiesOrPrice", `{"source_ticker":"ORIG11.SA","target_ticker":"DEST11.SA","source_quantity":0,"target_quantity":8,"target_unit_price":12.50}`},
			{"NegativeSourcePrice", `{"source_ticker":"ORIG11.SA","target_ticker":"DEST11.SA","source_quantity":10,"source_unit_price":-1,"target_quantity":8,"target_unit_price":12.50}`},
			{"InvalidDate", `{"source_ticker":"ORIG11.SA","target_ticker":"DEST11.SA","source_quantity":10,"target_quantity":8,"target_unit_price":12.50,"executed_at":"bad-date"}`},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/conversions", bytes.NewBufferString(tc.body)), "u1", map[string]string{"id": "p1"})
				rec := httptest.NewRecorder()
				h.ConvertAsset(rec, req)
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			})
		}
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("ConvertAsset", mock.Anything, "u1", "p1", mock.Anything).Return((*ConversionResult)(nil), errors.New("saldo insuficiente"))
		body := `{"source_ticker":"ORIG11.SA","target_ticker":"DEST11.SA","source_quantity":200,"target_quantity":160,"target_unit_price":12.50,"executed_at":"2025-06-15"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/conversions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.ConvertAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "saldo insuficiente")
	})

	t.Run("Success 201", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("ConvertAsset", mock.Anything, "u1", "p1", mock.MatchedBy(func(req ConversionRequest) bool {
			return req.SourceTicker == "ORIG11.SA" && req.TargetTicker == "DEST11.SA" && req.TargetAssetType == "FII" && req.TargetQuantity > 159.9 && req.TargetQuantity < 160.1
		})).Return(&ConversionResult{
			ConversionOut: &Transaction{ID: "tx-out", Type: "CONVERSION_OUT"},
			ConversionIn:  &Transaction{ID: "tx-in", Type: "CONVERSION_IN"},
		}, nil)

		body := `{"source_ticker":"orig11.sa","target_ticker":"dest11.sa","target_asset_type":"fii","source_quantity":200,"target_quantity":160,"target_unit_price":12.50,"executed_at":"2025-06-15"}`
		req := reqWithUserAndParams(httptest.NewRequest("POST", "/portfolios/p1/conversions", bytes.NewBufferString(body)), "u1", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()
		h.ConvertAsset(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "CONVERSION_OUT")
		assert.Contains(t, rec.Body.String(), "CONVERSION_IN")
	})

	t.Run("AddTransaction and UpdateTransaction accept CONVERSION_OUT and CONVERSION_IN", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddTransaction", mock.Anything, "u1", mock.MatchedBy(func(tx *Transaction) bool {
			return tx.Type == "CONVERSION_OUT" && tx.Fee < 1e-6 && tx.UnitPrice < 1e-6
		})).Return(&Transaction{ID: "tx-out", Type: "CONVERSION_OUT"}, nil).Once()

		reqOut := reqWithUserAndParams(
			httptest.NewRequest("POST", "/portfolios/p1/transactions", bytes.NewBufferString(`{"ticker":"ORIG11.SA","type":"CONVERSION_OUT","quantity":100,"unit_price":0,"fee":10}`)),
			"u1", map[string]string{"id": "p1"},
		)
		recOut := httptest.NewRecorder()
		h.AddTransaction(recOut, reqOut)
		assert.Equal(t, http.StatusCreated, recOut.Code)

		s.On("UpdateTransaction", mock.Anything, "u1", "p1", "tx-in", mock.MatchedBy(func(tx *Transaction) bool {
			return tx.Type == "CONVERSION_IN" && tx.Fee < 1e-6 && tx.UnitPrice > 12.49 && tx.UnitPrice < 12.51
		})).Return(nil).Once()

		reqIn := reqWithUserAndParams(
			httptest.NewRequest("PUT", "/portfolios/p1/transactions/tx-in", bytes.NewBufferString(`{"ticker":"DEST11.SA","type":"CONVERSION_IN","quantity":160,"unit_price":12.50,"fee":5}`)),
			"u1", map[string]string{"id": "p1", "txId": "tx-in"},
		)
		recIn := httptest.NewRecorder()
		h.UpdateTransaction(recIn, reqIn)
		assert.Equal(t, http.StatusOK, recIn.Code)
	})
}

