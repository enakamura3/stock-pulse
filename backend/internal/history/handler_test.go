package history

import (
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

type MockService struct {
	mock.Mock
}

func (m *MockService) GetPortfolioHistory(ctx context.Context, portfolioID, userID string) ([]UnifiedTransaction, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]UnifiedTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(nil)
	assert.NotNil(t, h)
}

func TestHandler_RegisterRoutes(t *testing.T) {
	h := NewHandler(nil)
	r := chi.NewRouter()
	h.RegisterRoutes(r)
}

func TestHandler_getHistory(t *testing.T) {
	t.Run("unauthorized no user id", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/portfolios/port1/history", nil)
		rr := httptest.NewRecorder()

		h := NewHandler(nil)
		h.getHistory(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.JSONEq(t, `{"error":"unauthorized"}`, rr.Body.String())
	})

	t.Run("service error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/portfolios/port1/history", nil)
		ctx := context.WithValue(req.Context(), auth.UserIDKey, "user1")
		req = req.WithContext(ctx)

		// Set chi URL params
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("portfolioID", "port1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		mockSvc := new(MockService)
		mockSvc.On("GetPortfolioHistory", mock.Anything, "port1", "user1").Return(nil, errors.New("svc error"))

		rr := httptest.NewRecorder()
		h := NewHandler(mockSvc)
		h.getHistory(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.JSONEq(t, `{"error":"failed to get history"}`, rr.Body.String())
	})

	t.Run("success nil history", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/portfolios/port1/history", nil)
		ctx := context.WithValue(req.Context(), auth.UserIDKey, "user1")
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("portfolioID", "port1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		mockSvc := new(MockService)
		mockSvc.On("GetPortfolioHistory", mock.Anything, "port1", "user1").Return(nil, nil)

		rr := httptest.NewRecorder()
		h := NewHandler(mockSvc)
		h.getHistory(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, `[]`, rr.Body.String())
	})

	t.Run("success with history", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/portfolios/port1/history", nil)
		ctx := context.WithValue(req.Context(), auth.UserIDKey, "user1")
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("portfolioID", "port1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		mockSvc := new(MockService)
		mockSvc.On("GetPortfolioHistory", mock.Anything, "port1", "user1").Return([]UnifiedTransaction{
			{ID: "tx1", Type: "BUY"},
		}, nil)

		rr := httptest.NewRecorder()
		h := NewHandler(mockSvc)
		h.getHistory(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"tx1"`)
	})
}
