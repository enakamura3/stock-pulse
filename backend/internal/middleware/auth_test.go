package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/config"
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
			expectedBody:   `{"error":"Sessão ausente. Faça login novamente."}`,
		},
		{
			name:           "Invalid algorithm",
			cookieName:     "access_token",
			cookieValue:    invalidAlgTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Sessão inválida ou expirada. Refaça o login."}`,
		},
		{
			name:           "Invalid signature",
			cookieName:     "access_token",
			cookieValue:    validTokenStr + "invalid",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Sessão inválida ou expirada. Refaça o login."}`,
		},
		{
			name:           "Missing user_id",
			cookieName:     "access_token",
			cookieValue:    missingUserIdTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Erro ao processar as credenciais."}`,
		},
		{
			name:           "Empty user_id",
			cookieName:     "access_token",
			cookieValue:    emptyUserIdTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"ID de usuário inválido nas credenciais."}`,
		},
		{
			name:           "Invalid claims type",
			cookieName:     "access_token",
			cookieValue:    customClaimsTokenStr,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Erro ao processar as credenciais."}`,
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
	config.Envs.FrontendURL = "http://example.com"
	defer func() { config.Envs.FrontendURL = "" }()

	tests := []struct {
		name                string
		method              string
		origin              string
		expectedOrigin      string
		expectedCredentials string
		expectedHeaders     string
		expectedMethods     string
		expectedStatus      int
	}{
		{
			name:                "Allowed frontend URL",
			method:              http.MethodGet,
			origin:              "http://example.com",
			expectedOrigin:      "http://example.com",
			expectedCredentials: "true",
			expectedHeaders:     "Content-Type, Authorization, X-Requested-With, X-Idempotency-Key, X-Admin-Key",
			expectedMethods:     "GET, POST, PUT, DELETE, OPTIONS",
			expectedStatus:      http.StatusOK,
		},
		{
			name:                "Disallowed localhost port",
			method:              http.MethodGet,
			origin:              "http://localhost:8080",
			expectedOrigin:      "",
			expectedCredentials: "",
			expectedHeaders:     "",
			expectedMethods:     "",
			expectedStatus:      http.StatusOK,
		},
		{
			name:                "Disallowed origin",
			method:              http.MethodGet,
			origin:              "http://hacker.com",
			expectedOrigin:      "",
			expectedCredentials: "",
			expectedHeaders:     "",
			expectedMethods:     "",
			expectedStatus:      http.StatusOK,
		},
		{
			name:                "OPTIONS preflight",
			method:              http.MethodOptions,
			origin:              "http://example.com",
			expectedOrigin:      "http://example.com",
			expectedCredentials: "true",
			expectedHeaders:     "Content-Type, Authorization, X-Requested-With, X-Idempotency-Key, X-Admin-Key",
			expectedMethods:     "GET, POST, PUT, DELETE, OPTIONS",
			expectedStatus:      http.StatusNoContent,
		},
		{
			name:                "OPTIONS preflight disallowed",
			method:              http.MethodOptions,
			origin:              "http://hacker.com",
			expectedOrigin:      "",
			expectedCredentials: "",
			expectedHeaders:     "",
			expectedMethods:     "",
			expectedStatus:      http.StatusForbidden,
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
			assert.Equal(t, tc.expectedCredentials, rr.Header().Get("Access-Control-Allow-Credentials"))
			assert.Equal(t, tc.expectedHeaders, rr.Header().Get("Access-Control-Allow-Headers"))
			assert.Equal(t, tc.expectedMethods, rr.Header().Get("Access-Control-Allow-Methods"))
		})
	}
}

func TestCORS_FallbackURL(t *testing.T) {
	config.Envs.FrontendURL = "" // ensure it is empty
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

func TestCORS_CommaSeparatedURLs(t *testing.T) {
	config.Envs.FrontendURL = "http://localhost:3000,http://192.168.1.100:3000"
	defer func() { config.Envs.FrontendURL = "" }()

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://192.168.1.100:3000")

	rr := httptest.NewRecorder()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS()
	middleware(nextHandler).ServeHTTP(rr, req)

	assert.Equal(t, "http://192.168.1.100:3000", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_RemoteIP_SameHost_Dev(t *testing.T) {
	config.Envs.FrontendURL = ""
	config.Envs.Env = "development"
	defer func() {
		config.Envs.FrontendURL = ""
		config.Envs.Env = ""
	}()

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Host = "192.168.1.100:8080"
	req.Header.Set("Origin", "http://192.168.1.100:3000")

	rr := httptest.NewRecorder()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS()
	middleware(nextHandler).ServeHTTP(rr, req)

	assert.Equal(t, "http://192.168.1.100:3000", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestIsOriginAllowed_EdgeCases(t *testing.T) {
	// 1. Empty origin
	assert.False(t, IsOriginAllowed("", "http://example.com", "development", "localhost:8080"))

	// 2. Production with origin not in configured list
	assert.False(t, IsOriginAllowed("http://hacker.com", "http://example.com", "production", "example.com"))

	// 3. Invalid URL syntax in origin returns empty host
	assert.Empty(t, extractHost("://invalid-url"))

	// 4. Dev mode with IP without port in reqHost
	assert.True(t, IsOriginAllowed("http://192.168.1.100:3000", "", "development", "192.168.1.100"))
}

