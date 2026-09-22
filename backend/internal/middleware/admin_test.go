package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireAdminKey(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	t.Run("Server key empty or unconfigured", func(t *testing.T) {
		mw := RequireAdminKey("")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "any-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "Chave administrativa não configurada no servidor.", body["error"])
	})

	t.Run("Server key whitespace only", func(t *testing.T) {
		mw := RequireAdminKey("   ")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "any-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "Chave administrativa não configurada no servidor.", body["error"])
	})

	t.Run("Missing X-Admin-Key header", func(t *testing.T) {
		mw := RequireAdminKey("secret-admin-key")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "Chave administrativa ausente.", body["error"])
	})

	t.Run("Empty or whitespace X-Admin-Key header", func(t *testing.T) {
		mw := RequireAdminKey("secret-admin-key")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "   ")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "Chave administrativa ausente.", body["error"])
	})

	t.Run("Invalid X-Admin-Key header", func(t *testing.T) {
		mw := RequireAdminKey("secret-admin-key")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "wrong-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "Chave administrativa inválida.", body["error"])
	})

	t.Run("Valid X-Admin-Key header allows request", func(t *testing.T) {
		mw := RequireAdminKey("secret-admin-key")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "secret-admin-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "ok", body["status"])
	})

	t.Run("Valid X-Admin-Key with surrounding whitespace", func(t *testing.T) {
		mw := RequireAdminKey("secret-admin-key")
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "  secret-admin-key  ")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "ok", body["status"])
	})
}
