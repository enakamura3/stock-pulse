package alert

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
)

// AlertService define a interface para operações de alertas (usada para mocking nos testes).
type AlertService interface {
	CreateAlert(ctx context.Context, userID string, ticker string, targetPrice float64, condition string) (*Alert, error)
	GetAlerts(ctx context.Context, userID string) ([]*Alert, error)
	DeleteAlert(ctx context.Context, id string, userID string) error
	ToggleAlert(ctx context.Context, id string, userID string) (string, error)
}

// Handler expõe os endpoints HTTP de Alertas de Preço.
type Handler struct {
	svc AlertService
}

// NewHandler inicializa o Alert Handler.
func NewHandler(svc AlertService) *Handler {
	return &Handler{
		svc: svc,
	}
}

// CreateReq representa o payload de requisição para criar um alerta.
type CreateReq struct {
	Ticker      string  `json:"ticker"`
	TargetPrice float64 `json:"target_price"`
	Condition   string  `json:"condition"` // "ABOVE" ou "BELOW"
}

// CreateAlert cria um novo alerta de preço associado ao usuário logado.
func (h *Handler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Sessão não autorizada ou expirada")
		return
	}

	var req CreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	alert, err := h.svc.CreateAlert(r.Context(), userID, req.Ticker, req.TargetPrice, req.Condition)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(alert)
}

// GetAlerts lista todos os alertas do usuário autenticado.
func (h *Handler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Sessão não autorizada ou expirada")
		return
	}

	alerts, err := h.svc.GetAlerts(r.Context(), userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Falha ao listar alertas")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(alerts)
}

// ToggleAlert altera o status de ativação de um alerta específico do usuário (Anti-IDOR).
func (h *Handler) ToggleAlert(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Sessão não autorizada ou expirada")
		return
	}

	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID do alerta inválido")
		return
	}

	nextStatus, err := h.svc.ToggleAlert(r.Context(), alertID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":     alertID,
		"status": nextStatus,
	})
}

// DeleteAlert exclui permanentemente o alerta do usuário autenticado (Anti-IDOR).
func (h *Handler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Sessão não autorizada ou expirada")
		return
	}

	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID do alerta inválido")
		return
	}

	err := h.svc.DeleteAlert(r.Context(), alertID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respondWithError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
