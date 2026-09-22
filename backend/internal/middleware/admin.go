package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

// RequireAdminKey valida o cabeçalho X-Admin-Key contra a chave administrativa configurada.
// Utiliza subtle.ConstantTimeCompare para mitigar ataques de temporização (timing attacks).
// Se adminKey estiver vazio ou não configurado, rejeita qualquer requisição por segurança (fail-closed).
func RequireAdminKey(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trimmedAdminKey := strings.TrimSpace(adminKey)
			if trimmedAdminKey == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Chave administrativa não configurada no servidor.")
				return
			}

			providedKey := strings.TrimSpace(r.Header.Get("X-Admin-Key"))
			if providedKey == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Chave administrativa ausente.")
				return
			}

			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(trimmedAdminKey)) != 1 {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Chave administrativa inválida.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
