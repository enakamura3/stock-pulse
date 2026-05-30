package market

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMarketService struct {
	mock.Mock
}

func (m *MockMarketService) GetQuote(ctx context.Context, ticker string) (*Quote, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMarketService) GetQuoteWithCacheStatus(ctx context.Context, ticker string) (*Quote, bool, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*Quote), args.Bool(1), args.Error(2)
	}
	return nil, false, args.Error(2)
}

func (m *MockMarketService) SearchAssets(ctx context.Context, query string) ([]SearchResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) != nil {
		return args.Get(0).([]SearchResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupHandlerTest() (*Handler, *MockMarketService) {
	s := new(MockMarketService)
	return NewHandler(s), s
}

func reqWithParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestHandler_GetQuote(t *testing.T) {
	t.Run("Missing Ticker", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/quote", nil)
		rec := httptest.NewRecorder()
		h.GetQuote(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetQuoteWithCacheStatus", mock.Anything, "INVALID").Return((*Quote)(nil), false, errors.New("not found"))
		req := reqWithParams(httptest.NewRequest("GET", "/quote/INVALID", nil), map[string]string{"ticker": "INVALID"})
		rec := httptest.NewRecorder()
		h.GetQuote(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetQuoteWithCacheStatus", mock.Anything, "AAPL").Return(&Quote{Symbol: "AAPL", Price: 150.0}, true, nil)
		req := reqWithParams(httptest.NewRequest("GET", "/quote/AAPL", nil), map[string]string{"ticker": "AAPL"})
		rec := httptest.NewRecorder()
		h.GetQuote(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "AAPL")
	})
}

func TestHandler_Search(t *testing.T) {
	t.Run("Empty Query", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("GET", "/search", nil)
		rec := httptest.NewRecorder()
		h.Search(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "[]")
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("SearchAssets", mock.Anything, "AAPL").Return(nil, errors.New("err"))
		req := httptest.NewRequest("GET", "/search?q=AAPL", nil)
		rec := httptest.NewRecorder()
		h.Search(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("SearchAssets", mock.Anything, "AAPL").Return([]SearchResult{{Symbol: "AAPL", Name: "Apple Inc."}}, nil)
		req := httptest.NewRequest("GET", "/search?q=AAPL", nil)
		rec := httptest.NewRecorder()
		h.Search(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Apple Inc.")
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

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}
