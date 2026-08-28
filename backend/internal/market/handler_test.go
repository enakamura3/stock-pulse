package market

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
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

func (m *MockMarketService) GetBenchmarks(ctx context.Context) (*MarketBenchmarks, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*MarketBenchmarks), args.Error(1)
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

	t.Run("Success Hit", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetQuoteWithCacheStatus", mock.Anything, "AAPL").Return(&Quote{Symbol: "AAPL", Price: 150.0}, true, nil)
		req := reqWithParams(httptest.NewRequest("GET", "/quote/AAPL", nil), map[string]string{"ticker": "AAPL"})
		rec := httptest.NewRecorder()
		h.GetQuote(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "AAPL")
		assert.Equal(t, "HIT", rec.Header().Get("X-Cache"))
	})

	t.Run("Success Miss", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetQuoteWithCacheStatus", mock.Anything, "MSFT").Return(&Quote{Symbol: "MSFT", Price: 300.0}, false, nil)
		req := reqWithParams(httptest.NewRequest("GET", "/quote/MSFT", nil), map[string]string{"ticker": "MSFT"})
		rec := httptest.NewRecorder()
		h.GetQuote(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "MSFT")
		assert.Equal(t, "MISS", rec.Header().Get("X-Cache"))
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
	rec := httptest.NewRecorder()
	httputils.RespondWithJSON(rec, http.StatusOK, failMarshal{})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_GetBenchmarks(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		mockBenchmarks := &MarketBenchmarks{
			IBOV: &BenchmarkItem{
				Symbol:        "^BVSP",
				Name:          "Ibovespa",
				Value:         130000.0,
				Change:        1300.0,
				ChangePercent: 1.01,
				PreviousClose: 128700.0,
			},
			SP500: &BenchmarkItem{
				Symbol:        "^GSPC",
				Name:          "S&P 500",
				Value:         5500.0,
				Change:        25.0,
				ChangePercent: 0.45,
				PreviousClose: 5475.0,
			},
		}
		s.On("GetBenchmarks", mock.Anything).Return(mockBenchmarks, nil)

		req := httptest.NewRequest("GET", "/market/benchmarks", nil)
		rec := httptest.NewRecorder()
		h.GetBenchmarks(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ibovespa")
		assert.Contains(t, rec.Body.String(), "^GSPC")
	})

	t.Run("Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetBenchmarks", mock.Anything).Return(nil, errors.New("provider failure"))

		req := httptest.NewRequest("GET", "/market/benchmarks", nil)
		rec := httptest.NewRecorder()
		h.GetBenchmarks(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Erro ao obter benchmarks de mercado")
	})
}

func (m *MockMarketService) InvalidateQuoteCache(ctx context.Context, symbols []string) (int64, error) {
	args := m.Called(ctx, symbols)
	return args.Get(0).(int64), args.Error(1)
}

func TestHandler_InvalidateCache(t *testing.T) {
	t.Run("Success with Specific Symbols", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("InvalidateQuoteCache", mock.Anything, []string{"PETR4.SA", "VALE3.SA"}).Return(int64(2), nil)

		body := `{"symbols":["PETR4.SA", "VALE3.SA"]}`
		req := httptest.NewRequest("POST", "/market/quotes/invalidate", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.InvalidateCache(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Cache de cotações invalidado com sucesso")
		assert.Contains(t, rec.Body.String(), `"removed":2`)
	})

	t.Run("Success with Empty Body", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("InvalidateQuoteCache", mock.Anything, []string(nil)).Return(int64(15), nil)

		req := httptest.NewRequest("POST", "/market/quotes/invalidate", nil)
		rec := httptest.NewRecorder()
		h.InvalidateCache(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"removed":15`)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("InvalidateQuoteCache", mock.Anything, []string(nil)).Return(int64(0), errors.New("redis down"))

		req := httptest.NewRequest("POST", "/market/quotes/invalidate", nil)
		rec := httptest.NewRecorder()
		h.InvalidateCache(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Erro ao invalidar cache de cotações")
	})
}

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error) {
	args := m.Called(ctx, ticker, assetType)
	if args.Get(0) != nil {
		return args.Get(0).([]DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

