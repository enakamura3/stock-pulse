package telegram

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
)

type HTTPHandler struct {
	svc Service
}

func NewHTTPHandler(svc Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) GenerateLinkToken(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	token, err := h.svc.GenerateLinkToken(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"token": token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
