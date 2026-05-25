package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

type contextKey string

// UserIDKey é a chave usada para armazenar e resgatar o ID do usuário autenticado no contexto HTTP.
const UserIDKey contextKey = "user_id"

// Handler expõe os métodos HTTP da API de Autenticação.
type Handler struct {
	service      *Service
	cookieSecure bool
}

// NewHandler cria uma nova instância de Handler.
func NewHandler(service *Service) *Handler {
	// Em modo de desenvolvimento local, cookieSecure pode ser desativado para permitir testes sem HTTPS
	cookieSecure := os.Getenv("ENV") != "development"
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
		h.respondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	user, err := h.service.Register(r.Context(), payload.Name, payload.Email, payload.Password)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusCreated, user)
}

// Login lida com a autenticação e injeção de Cookies HttpOnly.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var payload loginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	user, accessToken, refreshToken, err := h.service.Login(r.Context(), payload.Email, payload.Password)
	if err != nil {
		h.respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.setTokenCookies(w, accessToken, refreshToken)
	h.respondWithJSON(w, http.StatusOK, user)
}

// Logout limpa os cookies e invalida o refresh token no Redis.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie != nil {
		_ = h.service.RevokeRefreshToken(r.Context(), cookie.Value)
	}

	h.clearTokenCookies(w)
	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Logout efetuado com sucesso"})
}

// Refresh renova o access_token se o refresh_token for válido.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie == nil {
		h.respondWithError(w, http.StatusUnauthorized, "Sessão não encontrada. Faça login novamente.")
		return
	}

	userID, err := h.service.ValidateRefreshToken(r.Context(), cookie.Value)
	if err != nil {
		h.respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		h.respondWithError(w, http.StatusUnauthorized, "Usuário não encontrado.")
		return
	}

	newAccessToken, err := h.service.GenerateAccessToken(user)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao gerar credenciais de acesso.")
		return
	}

	// Atualiza o cookie do access token
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Path:     "/",
		Expires:  time.Now().Add(15 * time.Minute),
		MaxAge:   900,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Sessão renovada com sucesso"})
}

// Me retorna as informações do usuário autenticado no contexto HTTP.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

// setTokenCookies injeta os cookies access_token e refresh_token.
func (h *Handler) setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	// Access Token: 15 minutos
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(15 * time.Minute),
		MaxAge:   900,
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

func (h *Handler) respondWithError(w http.ResponseWriter, status int, msg string) {
	h.respondWithJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Erro de serialização JSON interno"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(response)
}
