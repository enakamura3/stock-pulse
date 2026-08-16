package portfolio

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockFIService struct {
	fixedincome.Service
	mock.Mock
}

func (m *mockFIService) GetRawTransactions(ctx context.Context, portfolioID string) ([]fixedincome.Transaction, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) != nil {
		return args.Get(0).([]fixedincome.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockFIService) GetAssetsByPortfolio(ctx context.Context, portfolioID string) ([]fixedincome.Asset, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) != nil {
		return args.Get(0).([]fixedincome.Asset), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestExportPortfolio(t *testing.T) {
	t.Run("Unauthorized - Missing UserID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/1/export", nil)
		rec := httptest.NewRecorder()
		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing Portfolio ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios//export", nil)
		ctx := context.WithValue(req.Context(), auth.UserIDKey, "user-123")
		rctx := chi.NewRouteContext()
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Portfolio Details Error / Forbidden", func(t *testing.T) {
		h, mockSvc := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/p1/export", nil)
		req = reqWithUserAndParams(req, "user-123", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()

		mockSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-123").Return((*Portfolio)(nil), ([]Position)(nil), errors.New("not found"))

		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("RV Transactions Error", func(t *testing.T) {
		h, mockSvc := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/p1/export", nil)
		req = reqWithUserAndParams(req, "user-123", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()

		mockSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-123").Return(&Portfolio{ID: "p1", Name: "MyPortfolio"}, ([]Position)(nil), nil)
		mockSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "user-123").Return(([]Transaction)(nil), errors.New("err"))

		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("FI Service Raw Transactions Error", func(t *testing.T) {
		h, mockSvc := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/p1/export", nil)
		req = reqWithUserAndParams(req, "user-123", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()

		fiSvc := new(mockFIService)

		mockSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-123").Return(&Portfolio{ID: "p1", Name: "MyPortfolio"}, ([]Position)(nil), nil)
		mockSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "user-123").Return([]Transaction{}, nil)
		mockSvc.On("GetFixedIncomeService").Return(fiSvc)
		fiSvc.On("GetRawTransactions", mock.Anything, "p1").Return(([]fixedincome.Transaction)(nil), errors.New("fi err"))

		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("FI Service Assets Error", func(t *testing.T) {
		h, mockSvc := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/p1/export", nil)
		req = reqWithUserAndParams(req, "user-123", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()

		fiSvc := new(mockFIService)

		mockSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-123").Return(&Portfolio{ID: "p1", Name: "MyPortfolio"}, ([]Position)(nil), nil)
		mockSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "user-123").Return([]Transaction{}, nil)
		mockSvc.On("GetFixedIncomeService").Return(fiSvc)
		fiSvc.On("GetRawTransactions", mock.Anything, "p1").Return([]fixedincome.Transaction{}, nil)
		fiSvc.On("GetAssetsByPortfolio", mock.Anything, "p1").Return(([]fixedincome.Asset)(nil), errors.New("fi asset err"))

		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success Export", func(t *testing.T) {
		h, mockSvc := setupHandlerTest()
		req := httptest.NewRequest("GET", "/portfolios/p1/export", nil)
		req = reqWithUserAndParams(req, "user-123", map[string]string{"id": "p1"})
		rec := httptest.NewRecorder()

		fiSvc := new(mockFIService)

		rvTxs := []Transaction{
			{
				ID:           "tx1",
				Ticker:       "PETR4",
				Type:         "BUY",
				Quantity:     100,
				UnitPrice:    30.50,
				ExchangeRate: 1.0,
				Fee:          0.0,
				ExecutedAt:   time.Now(),
			},
		}

		fiAssets := []fixedincome.Asset{
			{
				ID:           "asset1",
				Institution:  "Banco X",
				Type:         "CDB",
				DebtType:     "POS",
				Rate:         110.0,
				Indexer:      "CDI",
				MaturityDate: time.Now().Add(365 * 24 * time.Hour),
			},
		}

		fiTxs := []fixedincome.Transaction{
			{
				ID:      "fitx1",
				AssetID: "asset1",
				Type:    "APORTES",
				Amount:  1000.0,
				Date:    time.Now(),
			},
		}

		mockSvc.On("GetPortfolioDetails", mock.Anything, "p1", "user-123").Return(&Portfolio{ID: "p1", Name: "MyPortfolio"}, ([]Position)(nil), nil)
		mockSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "user-123").Return(rvTxs, nil)
		mockSvc.On("GetFixedIncomeService").Return(fiSvc)
		fiSvc.On("GetRawTransactions", mock.Anything, "p1").Return(fiTxs, nil)
		fiSvc.On("GetAssetsByPortfolio", mock.Anything, "p1").Return(fiAssets, nil)

		h.ExportPortfolio(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/zip", rec.Header().Get("Content-Type"))

		// Read ZIP content
		zipReader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
		assert.NoError(t, err)
		assert.Len(t, zipReader.File, 2)
	})
}
