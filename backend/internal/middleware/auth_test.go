package middleware

import (

	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/stretchr/testify/assert"
)

type customClaims struct {
	jwt.RegisteredClaims
}

func TestAuthRequired(t *testing.T) {
	jwtSecret := []byte("secret")

	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user123",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	validTokenStr, _ := validToken.SignedString(jwtSecret)

	invalidAlgToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "user123",
	})
	invalidAlgTokenStr, _ := invalidAlgToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	missingUserIdToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"some_claim": "value",
	})
	missingUserIdTokenStr, _ := missingUserIdToken.SignedString(jwtSecret)

	emptyUserIdToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "",
	})
	emptyUserIdTokenStr, _ := emptyUserIdToken.SignedString(jwtSecret)

	customClaimsToken := jwt.NewWithClaims(jwt.SigningMethodHS256, &customClaims{})
	customClaimsTokenStr, _ := customClaimsToken.SignedString(jwtSecret)

	tests := []struct {
		name           string
		cookieName     string
		cookieValue    string
		expectedStatus int
		expectedBody   string
		expectContext  bool
	}{
		{
			name:           "Missing cookie",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Sessão ausente. Faça login novamente."}` + "\n",
		},
		{
			name:           "Invalid algorithm",
			cookieName:     "access_token",
			cookieValue:    invalidAlgTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Sessão inválida ou expirada. Refaça o login."}` + "\n",
		},
		{
			name:           "Invalid signature",
			cookieName:     "access_token",
			cookieValue:    validTokenStr + "invalid",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Sessão inválida ou expirada. Refaça o login."}` + "\n",
		},
		{
			name:           "Missing user_id",
			cookieName:     "access_token",
			cookieValue:    missingUserIdTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Erro ao processar as credenciais."}` + "\n",
		},
		{
			name:           "Empty user_id",
			cookieName:     "access_token",
			cookieValue:    emptyUserIdTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"ID de usuário inválido nas credenciais."}` + "\n",
		},
		{
			name:           "Invalid claims type",
			cookieName:     "access_token",
			cookieValue:    customClaimsTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Erro ao processar as credenciais."}` + "\n",
		},
		{
			name:           "Valid token",
			cookieName:     "access_token",
			cookieValue:    validTokenStr,
			expectedStatus: http.StatusOK,
			expectContext:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			if tc.cookieName != "" {
				req.AddCookie(&http.Cookie{Name: tc.cookieName, Value: tc.cookieValue})
			}

			rr := httptest.NewRecorder()

			var contextUserID string
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if val, ok := r.Context().Value(auth.UserIDKey).(string); ok {
					contextUserID = val
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := AuthRequired(jwtSecret)
			middleware(nextHandler).ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				assert.Equal(t, tc.expectedBody, rr.Body.String())
			}
			if tc.expectContext {
				assert.Equal(t, "user123", contextUserID)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	os.Setenv("FRONTEND_URL", "http://example.com")
	defer os.Unsetenv("FRONTEND_URL")

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedOrigin string
		expectedStatus int
	}{
		{
			name:           "Allowed frontend URL",
			method:         http.MethodGet,
			origin:         "http://example.com",
			expectedOrigin: "http://example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Allowed localhost prefix",
			method:         http.MethodGet,
			origin:         "http://localhost:8080",
			expectedOrigin: "http://localhost:8080",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Disallowed origin",
			method:         http.MethodGet,
			origin:         "http://hacker.com",
			expectedOrigin: "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OPTIONS preflight",
			method:         http.MethodOptions,
			origin:         "http://example.com",
			expectedOrigin: "http://example.com",
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, "/", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			rr := httptest.NewRecorder()
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := CORS()
			middleware(nextHandler).ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Equal(t, tc.expectedOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
		})
	}
}

func TestCORS_FallbackURL(t *testing.T) {
	os.Setenv("FRONTEND_URL", "") // ensure it is empty
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS()
	middleware(nextHandler).ServeHTTP(rr, req)

	assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
}
