package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
)

type HTTPHandler struct {
	svc         Service
	botUsername string
}

func NewHTTPHandler(svc Service, botUsername string) *HTTPHandler {
	return &HTTPHandler{svc: svc, botUsername: botUsername}
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

	response := map[string]string{
		"token":        token,
		"bot_username": h.botUsername,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) GetTelegramStatus(w http.ResponseWriter, r *http.Request) {
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

	chatID, err := h.svc.GetChatIDByUserID(r.Context(), userID)
	linked := err == nil && chatID != 0

	response := map[string]any{
		"linked":       linked,
		"chat_id":      chatID,
		"bot_username": h.botUsername,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) UnlinkTelegram(w http.ResponseWriter, r *http.Request) {
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

	err = h.svc.UnlinkAccount(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to unlink telegram account: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "Telegram desvinculado com sucesso",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
