package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

// AuthRequired é o middleware real de validação de token JWT.
func AuthRequired(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Busca pelo cookie access_token
			cookie, err := r.Cookie("access_token")
			if err != nil || cookie == nil {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão ausente. Faça login novamente.")
				return
			}

			// Realiza o parse e a validação do JWT
			token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
				// Valida se o algoritmo de assinatura é o correto (HMAC SHA-256)
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("algoritmo de assinatura inesperado: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão inválida ou expirada. Refaça o login.")
				return
			}

			// Extrai as claims e injeta o user_id no contexto da requisição
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || claims["user_id"] == nil {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Erro ao processar as credenciais.")
				return
			}

			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, "ID de usuário inválido nas credenciais.")
				return
			}

			// Injeta o UserID no contexto da requisição
			ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS configura as permissões de compartilhamento de recursos entre origens de forma extremamente segura.
func CORS() func(http.Handler) http.Handler {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000" // Fallback seguro de desenvolvimento
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			// Apenas autoriza a origem se coincidir com o Frontend cadastrado
			if origin == frontendURL {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Idempotency-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

			// Responde imediatamente a requisições de preflight do browser (OPTIONS)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
