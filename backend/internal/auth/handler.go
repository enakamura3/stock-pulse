package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

type contextKey string

// UserIDKey é a chave usada para armazenar e resgatar o ID do usuário autenticado no contexto HTTP.
const UserIDKey contextKey = "user_id"

// AuthService define os casos de uso de autenticação que o Handler pode consumir.
type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (*User, string, string, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	ValidateRefreshToken(ctx context.Context, token string) (string, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GenerateAccessToken(user *User) (string, error)
	UpdateProfile(ctx context.Context, id, name, email string) (*User, error)
	UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) error
	DeleteUser(ctx context.Context, id string) error
}

// Handler expõe os métodos HTTP da API de Autenticação.
type Handler struct {
	service      AuthService
	cookieSecure bool
}

// NewHandler cria uma nova instância de Handler.
func NewHandler(service AuthService) *Handler {
	// Em modo de desenvolvimento local, cookieSecure pode ser desativado para permitir testes sem HTTPS
	cookieSecure := config.Envs.Env != "development"
	return &Handler{
		service:      service,
		cookieSecure: cookieSecure,
	}
}

type registerPayload struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register lida com o registro de novos usuários.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var payload registerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	user, err := h.service.Register(r.Context(), payload.Name, payload.Email, payload.Password)
	if err != nil {
		slog.Warn("Falha no registro", "email", payload.Email, "error", err.Error())
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, user)
}

// Login lida com a autenticação e injeção de Cookies HttpOnly.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var payload loginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	user, accessToken, refreshToken, err := h.service.Login(r.Context(), payload.Email, payload.Password)
	if err != nil {
		slog.Warn("Falha de autenticação", "email", payload.Email, "error", err.Error())
		httputils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.setTokenCookies(w, accessToken, refreshToken)
	httputils.RespondWithJSON(w, http.StatusOK, user)
}

// Logout limpa os cookies e invalida o refresh token no Redis.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie != nil {
		_ = h.service.RevokeRefreshToken(r.Context(), cookie.Value)
	}

	h.clearTokenCookies(w)
	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Logout efetuado com sucesso"})
}

// Refresh renova o access_token se o refresh_token for válido.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie == nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Sessão não encontrada. Faça login novamente.")
		return
	}

	userID, err := h.service.ValidateRefreshToken(r.Context(), cookie.Value)
	if err != nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Usuário não encontrado.")
		return
	}

	newAccessToken, err := h.service.GenerateAccessToken(user)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao gerar credenciais de acesso.")
		return
	}

	// Atualiza o cookie do access token
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Path:     "/",
		Expires:  time.Now().Add(2 * time.Hour),
		MaxAge:   7200,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Sessão renovada com sucesso"})
}

// Me retorna as informações do usuário autenticado no contexto HTTP.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, user)
}

// setTokenCookies injeta os cookies access_token e refresh_token.
func (h *Handler) setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	// Access Token: 2 horas
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(2 * time.Hour),
		MaxAge:   7200,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Refresh Token: 7 dias
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearTokenCookies define Max-Age=-1 para expirar e remover os cookies no browser.
func (h *Handler) clearTokenCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

type updateProfilePayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type updatePasswordPayload struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdateProfile atualiza as informações do usuário.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	var payload updateProfilePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, payload.Name, payload.Email)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, user)
}

// UpdatePassword altera a senha do usuário logado.
func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	var payload updatePasswordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	err := h.service.UpdatePassword(r.Context(), userID, payload.CurrentPassword, payload.NewPassword)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Senha atualizada com sucesso"})
}

// DeleteUser exclui a conta do usuário e limpa os cookies.
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie != nil {
		_ = h.service.RevokeRefreshToken(r.Context(), cookie.Value)
	}

	err = h.service.DeleteUser(r.Context(), userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.clearTokenCookies(w)
	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Conta excluída com sucesso"})
}
