package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestClientIP(t *testing.T) {
	t.Run("X-Forwarded-For with multiple IPs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", " 203.0.113.195 , 70.41.3.18, 150.172.238.178 ")
		assert.Equal(t, "203.0.113.195", ClientIP(req))
	})

	t.Run("X-Forwarded-For with single IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", "198.51.100.2")
		assert.Equal(t, "198.51.100.2", ClientIP(req))
	})

	t.Run("X-Real-IP when X-Forwarded-For empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", " 198.51.100.1 ")
		assert.Equal(t, "198.51.100.1", ClientIP(req))
	})

	t.Run("RemoteAddr with host:port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		assert.Equal(t, "192.0.2.1", ClientIP(req))
	})

	t.Run("RemoteAddr without port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.5"
		assert.Equal(t, "192.0.2.5", ClientIP(req))
	})

	t.Run("Empty RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ""
		assert.Equal(t, "", ClientIP(req))
	})
}

func TestRateLimit_NilRedis(t *testing.T) {
	mw := RateLimit(nil, RateLimiterConfig{Limit: 5, Window: time.Minute})
	handlerCalled := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimit_Defaults(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	// All defaults: Limit <= 0, Window <= 0, KeyPrefix == "", KeyExtractor == nil
	mw := RateLimit(rdb, RateLimiterConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "59", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_UnderLimitAndOverLimit(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	cfg := RateLimiterConfig{
		Limit:     3,
		Window:    time.Minute,
		KeyPrefix: "rate_limit:test",
	}
	mw := RateLimit(rdb, cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 1st request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:5000"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "2", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

	// 2nd request
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1", w.Header().Get("X-RateLimit-Remaining"))

	// 3rd request (hits limit exactly)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))

	// 4th request (exceeds limit -> 429)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))

	var errResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "Muitas requisições. Tente novamente mais tarde.", errResp["error"])

	// Different IP is unaffected
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:5000"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "2", w2.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_CustomKeyExtractor(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	t.Run("Custom extractor returns value", func(t *testing.T) {
		cfg := RateLimiterConfig{
			Limit:     2,
			Window:    time.Minute,
			KeyPrefix: "rate_limit:user",
			KeyExtractor: func(r *http.Request) string {
				return r.Header.Get("X-User-ID")
			},
		}
		mw := RateLimit(rdb, cfg)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User-ID", "user_123")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1", w.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("Custom extractor returns empty string -> falls back to unknown", func(t *testing.T) {
		cfg := RateLimiterConfig{
			Limit:     2,
			Window:    time.Minute,
			KeyPrefix: "rate_limit:user",
			KeyExtractor: func(r *http.Request) string {
				return ""
			},
		}
		mw := RateLimit(rdb, cfg)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRateLimit_RedisError_FailOpen(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})

	cfg := RateLimiterConfig{
		Limit:     2,
		Window:    time.Minute,
		KeyPrefix: "rate_limit:failopen",
	}
	mw := RateLimit(rdb, cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pass"))
	}))

	// Close miniredis to simulate connection failure
	s.Close()
	rdb.Close()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// In fail-open mode, error is logged and request still succeeds
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pass", w.Body.String())
}

func TestRateLimit_Presets(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	presets := []struct {
		name     string
		mw       func(http.Handler) http.Handler
		expLimit string
	}{
		{"AuthLogin", RateLimitAuthLogin(rdb), "10"},
		{"AuthRegister", RateLimitAuthRegister(rdb), "5"},
		{"AuthRefresh", RateLimitAuthRefresh(rdb), "30"},
		{"TelegramLink", RateLimitTelegramLink(rdb), "10"},
	}

	for _, p := range presets {
		t.Run(p.name, func(t *testing.T) {
			handler := p.mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("POST", "/test", nil)
			req.RemoteAddr = "10.10.10.10:1234"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, p.expLimit, w.Header().Get("X-RateLimit-Limit"))
		})
	}
}

func TestRateLimit_SubSecondWindow(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	mw := RateLimit(rdb, RateLimiterConfig{
		Limit:  5,
		Window: 100 * time.Millisecond,
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimit_InvalidScriptResult(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectEvalSha(rateLimitLuaScript.Hash(), []string{"rate_limit:192.0.2.1"}, 60).SetVal("unexpected string")

	mw := RateLimit(db, RateLimiterConfig{
		Limit:  5,
		Window: time.Minute,
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pass"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pass", w.Body.String())
}

func TestRateLimit_ZeroOrNegativeTTL(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectEvalSha(rateLimitLuaScript.Hash(), []string{"rate_limit:192.0.2.1"}, 60).SetVal([]interface{}{int64(1), int64(-1)})

	mw := RateLimit(db, RateLimiterConfig{
		Limit:  5,
		Window: time.Minute,
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "60", w.Header().Get("X-RateLimit-Reset"))
}


