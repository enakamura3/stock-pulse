package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stockpulse/backend/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAlertService struct {
	mock.Mock
}

func (m *MockAlertService) CreateAlert(ctx context.Context, userID string, ticker string, targetPrice float64, condition string) (*Alert, error) {
	args := m.Called(ctx, userID, ticker, targetPrice, condition)
	if args.Get(0) != nil {
		return args.Get(0).(*Alert), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAlertService) GetAlerts(ctx context.Context, userID string) ([]*Alert, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]*Alert), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAlertService) DeleteAlert(ctx context.Context, id string, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockAlertService) ToggleAlert(ctx context.Context, id string, userID string) (string, error) {
	args := m.Called(ctx, id, userID)
	return args.String(0), args.Error(1)
}

func setupHandlerTest() (*Handler, *MockAlertService) {
	svc := new(MockAlertService)
	h := NewHandler(svc)
	return h, svc
}

func reqWithUser(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
	return req.WithContext(ctx)
}

func reqWithParams(req *http.Request, params map[string]string) *http.Request {
	routeCtx := chi.NewRouteContext()
	for k, v := range params {
		routeCtx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestHandler_CreateAlert(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("POST", "/alerts", nil)
		rec := httptest.NewRecorder()

		h.CreateAlert(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser(httptest.NewRequest("POST", "/alerts", bytes.NewBufferString("{invalid_json}")), "u1")
		rec := httptest.NewRecorder()

		h.CreateAlert(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, svc := setupHandlerTest()
		payload := CreateReq{Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE"}
		body, _ := json.Marshal(payload)
		req := reqWithUser(httptest.NewRequest("POST", "/alerts", bytes.NewBuffer(body)), "u1")
		rec := httptest.NewRecorder()

		svc.On("CreateAlert", mock.Anything, "u1", "AAPL", 150.0, "ABOVE").Return((*Alert)(nil), errors.New("svc err"))
		h.CreateAlert(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, svc := setupHandlerTest()
		payload := CreateReq{Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE"}
		body, _ := json.Marshal(payload)
		req := reqWithUser(httptest.NewRequest("POST", "/alerts", bytes.NewBuffer(body)), "u1")
		rec := httptest.NewRecorder()

		alert := &Alert{ID: "a1", Ticker: "AAPL"}
		svc.On("CreateAlert", mock.Anything, "u1", "AAPL", 150.0, "ABOVE").Return(alert, nil)
		h.CreateAlert(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestHandler_GetAlerts(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/alerts", nil)
		rec := httptest.NewRecorder()

		h.GetAlerts(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, svc := setupHandlerTest()
		req := reqWithUser(httptest.NewRequest("GET", "/alerts", nil), "u1")
		rec := httptest.NewRecorder()

		svc.On("GetAlerts", mock.Anything, "u1").Return(([]*Alert)(nil), errors.New("svc err"))
		h.GetAlerts(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, svc := setupHandlerTest()
		req := reqWithUser(httptest.NewRequest("GET", "/alerts", nil), "u1")
		rec := httptest.NewRecorder()

		alerts := []*Alert{{ID: "a1"}}
		svc.On("GetAlerts", mock.Anything, "u1").Return(alerts, nil)
		h.GetAlerts(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_DeleteAlert(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("DELETE", "/alerts/a1", nil)
		rec := httptest.NewRecorder()

		h.DeleteAlert(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithParams(reqWithUser(httptest.NewRequest("DELETE", "/alerts/", nil), "u1"), map[string]string{"id": ""})
		rec := httptest.NewRecorder()

		h.DeleteAlert(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, svc := setupHandlerTest()
		req := reqWithParams(reqWithUser(httptest.NewRequest("DELETE", "/alerts/a1", nil), "u1"), map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()

		svc.On("DeleteAlert", mock.Anything, "a1", "u1").Return(errors.New("svc err"))
		h.DeleteAlert(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, svc := setupHandlerTest()
		req := reqWithParams(reqWithUser(httptest.NewRequest("DELETE", "/alerts/a1", nil), "u1"), map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()

		svc.On("DeleteAlert", mock.Anything, "a1", "u1").Return(nil)
		h.DeleteAlert(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestHandler_ToggleAlert(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("PATCH", "/alerts/a1/toggle", nil)
		rec := httptest.NewRecorder()

		h.ToggleAlert(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithParams(reqWithUser(httptest.NewRequest("PATCH", "/alerts//toggle", nil), "u1"), map[string]string{"id": ""})
		rec := httptest.NewRecorder()

		h.ToggleAlert(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, svc := setupHandlerTest()
		req := reqWithParams(reqWithUser(httptest.NewRequest("PATCH", "/alerts/a1/toggle", nil), "u1"), map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()

		svc.On("ToggleAlert", mock.Anything, "a1", "u1").Return("", errors.New("svc err"))
		h.ToggleAlert(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, svc := setupHandlerTest()
		req := reqWithParams(reqWithUser(httptest.NewRequest("PATCH", "/alerts/a1/toggle", nil), "u1"), map[string]string{"id": "a1"})
		rec := httptest.NewRecorder()

		svc.On("ToggleAlert", mock.Anything, "a1", "u1").Return("DISABLED", nil)
		h.ToggleAlert(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
