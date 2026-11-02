package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/redis/go-redis/v9"
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

func createTestAdminToken(secret []byte, userID, role string, expOffset time.Duration, iatOffset time.Duration) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"iss":     auth.JWTIssuer,
		"aud":     auth.JWTAudience,
		"exp":     time.Now().Add(expOffset).Unix(),
		"iat":     time.Now().Add(iatOffset).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(secret)
	return tokenStr
}

func TestRequireAdmin(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	jwtSecret := []byte("super-secret-key-with-at-least-32-bytes!!")
	adminKey := "valid-admin-api-key"

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	t.Run("Valid X-Admin-Key header allows request immediately", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", adminKey)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("X-Admin-Key provided but server key is unconfigured", func(t *testing.T) {
		mw := RequireAdmin("", jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "any-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "não configurada no servidor")
	})

	t.Run("Invalid X-Admin-Key header fails", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.Header.Set("X-Admin-Key", "wrong-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Chave administrativa inválida")
	})

	t.Run("Missing credentials (no header and no cookie)", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Credenciais ausentes")
	})

	t.Run("Empty secret returns 500", func(t *testing.T) {
		mw := RequireAdmin(adminKey, []byte(""), rdb)
		handler := mw(dummyHandler)

		token := createTestAdminToken(jwtSecret, "user-1", auth.RoleAdmin, 15*time.Minute, 0)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Invalid/expired JWT token returns 401", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid.jwt.token"})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Sessão inválida ou expirada")
	})

	t.Run("Token claims missing user_id returns 401", func(t *testing.T) {
		claims := jwt.MapClaims{
			"iss": auth.JWTIssuer,
			"aud": auth.JWTAudience,
			"exp": time.Now().Add(15 * time.Minute).Unix(),
			"iat": time.Now().Unix(),
		}
		rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := rawToken.SignedString(jwtSecret)

		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: tokenStr})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Token claims non-string user_id returns 401", func(t *testing.T) {
		claims := jwt.MapClaims{
			"user_id": 12345, // invalid type
			"iss":     auth.JWTIssuer,
			"aud":     auth.JWTAudience,
			"exp":     time.Now().Add(15 * time.Minute).Unix(),
			"iat":     time.Now().Unix(),
		}
		rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := rawToken.SignedString(jwtSecret)

		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: tokenStr})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Revoked user session returns 401", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		mr.Set("revoked_user:revoked-adm", "1")
		token := createTestAdminToken(jwtSecret, "revoked-adm", auth.RoleAdmin, 15*time.Minute, 0)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Sessão revogada")
	})

	t.Run("Password changed after token issued returns 401", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		now := time.Now().Unix()

		// Set token with iat < pwd_changed_at
		claims := jwt.MapClaims{
			"user_id": "pwd-adm",
			"role":    auth.RoleAdmin,
			"iss":     auth.JWTIssuer,
			"aud":     auth.JWTAudience,
			"exp":     now + 1000,
			"iat":     now - 50,
		}
		mr.Set("pwd_changed_at:pwd-adm", time.Unix(now+10, 0).Format("2006-01-02")) // unparseable or later
		mr.Set("pwd_changed_at:pwd-adm", string([]byte("2000000000")))

		rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := rawToken.SignedString(jwtSecret)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: tokenStr})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Sessão revogada após alteração de senha")
	})

	t.Run("Valid JWT session with role USER returns 403 Forbidden", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		token := createTestAdminToken(jwtSecret, "regular-user", auth.RoleUser, 15*time.Minute, 0)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "Acesso restrito a administradores")
	})

	t.Run("Valid JWT session with role ADMIN returns 200 OK", func(t *testing.T) {
		mw := RequireAdmin(adminKey, jwtSecret, rdb)
		handler := mw(dummyHandler)

		token := createTestAdminToken(jwtSecret, "admin-user", auth.RoleAdmin, 15*time.Minute, 0)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workers", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
