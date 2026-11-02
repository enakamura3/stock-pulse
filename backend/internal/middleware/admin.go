package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
	"github.com/redis/go-redis/v9"
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

// RequireAdmin valida se a requisição possui permissão administrativa:
// 1. Ou através do cabeçalho X-Admin-Key válido (autenticação técnica / M2M);
// 2. Ou através de sessão JWT autenticada com papel ADMIN (RoleAdmin).
func RequireAdmin(adminKey string, jwtSecret []byte, rdb ...*redis.Client) func(http.Handler) http.Handler {
	var redisClient *redis.Client
	if len(rdb) > 0 {
		redisClient = rdb[0]
	}
	trimmedAdminKey := strings.TrimSpace(adminKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedKey := strings.TrimSpace(r.Header.Get("X-Admin-Key"))

			// 1. Tenta autenticação via chave de API administrativa
			if providedKey != "" {
				if trimmedAdminKey == "" {
					httputils.RespondWithError(w, http.StatusUnauthorized, "Chave administrativa não configurada no servidor.")
					return
				}

				if subtle.ConstantTimeCompare([]byte(providedKey), []byte(trimmedAdminKey)) != 1 {
					httputils.RespondWithError(w, http.StatusUnauthorized, "Chave administrativa inválida.")
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			// 2. Tenta autenticação via sessão JWT (cookie access_token)
			cookie, err := r.Cookie("access_token")
			if err != nil || cookie == nil || strings.TrimSpace(cookie.Value) == "" {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Credenciais ausentes. Forneça X-Admin-Key ou faça login como administrador.")
				return
			}

			if len(jwtSecret) == 0 {
				httputils.RespondWithError(w, http.StatusInternalServerError, "Configuração de autenticação inválida no servidor.")
				return
			}

			token, err := jwt.Parse(
				cookie.Value,
				func(token *jwt.Token) (interface{}, error) {
					return jwtSecret, nil
				},
				jwt.WithValidMethods([]string{"HS256"}),
				jwt.WithExpirationRequired(),
				jwt.WithIssuer(auth.JWTIssuer),
				jwt.WithAudience(auth.JWTAudience),
			)

			if err != nil || !token.Valid {
				httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão inválida ou expirada.")
				return
			}

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

			// Verifica revogação e troca de senha no Redis
			if redisClient != nil {
				exists, err := redisClient.Exists(r.Context(), fmt.Sprintf("revoked_user:%s", userID)).Result()
				if err == nil && exists > 0 {
					httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão revogada. Faça login novamente.")
					return
				}

				pwdChangedStr, err := redisClient.Get(r.Context(), fmt.Sprintf("pwd_changed_at:%s", userID)).Result()
				if err == nil && pwdChangedStr != "" {
					pwdChangedAt, parseErr := strconv.ParseInt(pwdChangedStr, 10, 64)
					iatFloat, hasIat := claims["iat"].(float64)
					if parseErr == nil && (!hasIat || int64(iatFloat) < pwdChangedAt) {
						httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão revogada após alteração de senha. Faça login novamente.")
						return
					}
				}
			}

			role, _ := claims["role"].(string)
			if strings.ToUpper(strings.TrimSpace(role)) != auth.RoleAdmin {
				httputils.RespondWithError(w, http.StatusForbidden, "Acesso restrito a administradores.")
				return
			}

			ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
			ctx = context.WithValue(ctx, auth.UserRoleKey, auth.RoleAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
