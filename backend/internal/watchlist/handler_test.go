package watchlist

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

type MockWatchlistService struct {
	mock.Mock
}

func (m *MockWatchlistService) CreateWatchlist(ctx context.Context, userID, name string) (*Watchlist, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*Watchlist), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWatchlistService) GetWatchlists(ctx context.Context, userID string) ([]Watchlist, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]Watchlist), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWatchlistService) GetWatchlist(ctx context.Context, id, userID string) (*Watchlist, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*Watchlist), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWatchlistService) DeleteWatchlist(ctx context.Context, id, userID string) error {
	return m.Called(ctx, id, userID).Error(0)
}

func (m *MockWatchlistService) AddAssetToWatchlist(ctx context.Context, watchlistID, userID, ticker string) (*Item, error) {
	args := m.Called(ctx, watchlistID, userID, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*Item), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWatchlistService) RemoveAssetFromWatchlist(ctx context.Context, watchlistID, userID, ticker string) error {
	return m.Called(ctx, watchlistID, userID, ticker).Error(0)
}

func setupHandlerTest() (*Handler, *MockWatchlistService) {
	s := new(MockWatchlistService)
	return NewHandler(s), s
}

func reqWithUser(method, url string, body interface{}, userID string) *http.Request {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, url, bytes.NewBuffer(b))
	if userID != "" {
		ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
		req = req.WithContext(ctx)
	}
	return req
}

func reqWithParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestHandler_GetWatchlists(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("GET", "/", nil, "")
		rec := httptest.NewRecorder()
		h.GetWatchlists(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetWatchlists", mock.Anything, "u1").Return(nil, errors.New("err"))
		req := reqWithUser("GET", "/", nil, "u1")
		rec := httptest.NewRecorder()
		h.GetWatchlists(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetWatchlists", mock.Anything, "u1").Return([]Watchlist{{ID: "w1"}}, nil)
		req := reqWithUser("GET", "/", nil, "u1")
		rec := httptest.NewRecorder()
		h.GetWatchlists(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_CreateWatchlist(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("POST", "/", nil, "")
		rec := httptest.NewRecorder()
		h.CreateWatchlist(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString("invalid json"))
		ctx := context.WithValue(req.Context(), auth.UserIDKey, "u1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.CreateWatchlist(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("CreateWatchlist", mock.Anything, "u1", "My List").Return(nil, errors.New("err"))
		req := reqWithUser("POST", "/", map[string]string{"name": "My List"}, "u1")
		rec := httptest.NewRecorder()
		h.CreateWatchlist(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("CreateWatchlist", mock.Anything, "u1", "My List").Return(&Watchlist{ID: "w1"}, nil)
		req := reqWithUser("POST", "/", map[string]string{"name": "My List"}, "u1")
		rec := httptest.NewRecorder()
		h.CreateWatchlist(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestHandler_GetWatchlist(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("GET", "/w1", nil, "")
		rec := httptest.NewRecorder()
		h.GetWatchlist(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("GET", "/", nil, "u1")
		rec := httptest.NewRecorder()
		h.GetWatchlist(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetWatchlist", mock.Anything, "w1", "u1").Return(nil, errors.New("err"))
		req := reqWithParams(reqWithUser("GET", "/w1", nil, "u1"), map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.GetWatchlist(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("GetWatchlist", mock.Anything, "w1", "u1").Return(&Watchlist{ID: "w1"}, nil)
		req := reqWithParams(reqWithUser("GET", "/w1", nil, "u1"), map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.GetWatchlist(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_DeleteWatchlist(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("DELETE", "/w1", nil, "")
		rec := httptest.NewRecorder()
		h.DeleteWatchlist(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("DELETE", "/", nil, "u1")
		rec := httptest.NewRecorder()
		h.DeleteWatchlist(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeleteWatchlist", mock.Anything, "w1", "u1").Return(errors.New("err"))
		req := reqWithParams(reqWithUser("DELETE", "/w1", nil, "u1"), map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.DeleteWatchlist(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("DeleteWatchlist", mock.Anything, "w1", "u1").Return(nil)
		req := reqWithParams(reqWithUser("DELETE", "/w1", nil, "u1"), map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.DeleteWatchlist(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandler_AddAsset(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("POST", "/w1/assets", nil, "")
		rec := httptest.NewRecorder()
		h.AddAsset(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing ID", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("POST", "/assets", nil, "u1")
		rec := httptest.NewRecorder()
		h.AddAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString("invalid json"))
		ctx := context.WithValue(req.Context(), auth.UserIDKey, "u1")
		req = req.WithContext(ctx)
		req = reqWithParams(req, map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.AddAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddAssetToWatchlist", mock.Anything, "w1", "u1", "AAPL").Return(nil, errors.New("err"))
		req := reqWithParams(reqWithUser("POST", "/w1/assets", map[string]string{"ticker": "AAPL"}, "u1"), map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.AddAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("AddAssetToWatchlist", mock.Anything, "w1", "u1", "AAPL").Return(&Item{ID: "i1"}, nil)
		req := reqWithParams(reqWithUser("POST", "/w1/assets", map[string]string{"ticker": "AAPL"}, "u1"), map[string]string{"id": "w1"})
		rec := httptest.NewRecorder()
		h.AddAsset(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestHandler_RemoveAsset(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("DELETE", "/w1/assets/AAPL", nil, "")
		rec := httptest.NewRecorder()
		h.RemoveAsset(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing Params", func(t *testing.T) {
		h, _ := setupHandlerTest()
		req := reqWithUser("DELETE", "/", nil, "u1")
		rec := httptest.NewRecorder()
		h.RemoveAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("RemoveAssetFromWatchlist", mock.Anything, "w1", "u1", "AAPL").Return(errors.New("err"))
		req := reqWithParams(reqWithUser("DELETE", "/w1/assets/AAPL", nil, "u1"), map[string]string{"id": "w1", "ticker": "AAPL"})
		rec := httptest.NewRecorder()
		h.RemoveAsset(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		h, s := setupHandlerTest()
		s.On("RemoveAssetFromWatchlist", mock.Anything, "w1", "u1", "AAPL").Return(nil)
		req := reqWithParams(reqWithUser("DELETE", "/w1/assets/AAPL", nil, "u1"), map[string]string{"id": "w1", "ticker": "AAPL"})
		rec := httptest.NewRecorder()
		h.RemoveAsset(rec, req)
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
