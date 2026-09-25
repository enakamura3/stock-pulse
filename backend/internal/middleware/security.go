package middleware

import "net/http"

// SecurityHeaders adiciona cabeçalhos HTTP defensivos para proteger contra
// ataques como MIME type sniffing (X-Content-Type-Options), clickjacking (X-Frame-Options)
// e vazamento de informações de referência (Referrer-Policy).
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}
