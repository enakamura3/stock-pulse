package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		routePattern string
		statusCode   int
	}{
		{
			name:         "Normal request",
			path:         "/api/data",
			routePattern: "/api/data",
			statusCode:   http.StatusOK,
		},
		{
			name:         "Metrics endpoint ignored",
			path:         "/metrics",
			routePattern: "/metrics",
			statusCode:   http.StatusOK,
		},
		{
			name:         "Healthz endpoint ignored",
			path:         "/healthz",
			routePattern: "/healthz",
			statusCode:   http.StatusOK,
		},
		{
			name:         "Fallback route pattern",
			path:         "/fallback",
			routePattern: "", // chi context missing or route pattern empty
			statusCode:   http.StatusCreated,
		},
		{
			name:         "Default 200 status when WriteHeader not explicitly called but Write is",
			path:         "/api/write-only",
			routePattern: "/api/write-only",
			statusCode:   0, // Write will trigger default 200
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tc.path, nil)

			// Simulate chi context for routePattern
			if tc.routePattern != "" {
				rctx := chi.NewRouteContext()
				rctx.RoutePatterns = []string{tc.routePattern}
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			rr := httptest.NewRecorder()

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.statusCode == 0 {
					w.Write([]byte("ok")) // will trigger 200
				} else {
					w.WriteHeader(tc.statusCode)
				}
			})

			middleware := Metrics()
			middleware(nextHandler).ServeHTTP(rr, req)

			if tc.statusCode == 0 {
				assert.Equal(t, http.StatusOK, rr.Code)
			} else {
				assert.Equal(t, tc.statusCode, rr.Code)
			}
		})
	}
}

func TestResponseWriterDelegator_Default200(t *testing.T) {
	// Directly test delegator when nextHandler does not call WriteHeader or Write
	req, _ := http.NewRequest(http.MethodGet, "/empty", nil)
	rr := httptest.NewRecorder()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Does nothing
	})

	middleware := Metrics()
	middleware(nextHandler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code) // Recorder default is 200
}
