package watchlist

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

// WatchlistService define as operações de negócio da watchlist que o Handler pode consumir.
type WatchlistService interface {
	CreateWatchlist(ctx context.Context, userID, name string) (*Watchlist, error)
	GetWatchlists(ctx context.Context, userID string) ([]Watchlist, error)
	GetWatchlist(ctx context.Context, id, userID string) (*Watchlist, error)
	DeleteWatchlist(ctx context.Context, id, userID string) error
	AddAssetToWatchlist(ctx context.Context, watchlistID, userID, ticker string) (*Item, error)
	RemoveAssetFromWatchlist(ctx context.Context, watchlistID, userID, ticker string) error
}

// Handler expõe endpoints REST protegidos para gerenciamento de favoritos.
type Handler struct {
	service WatchlistService
}

// NewHandler cria uma nova instância de Handler.
func NewHandler(service WatchlistService) *Handler {
	return &Handler{service: service}
}

type watchlistPayload struct {
	Name string `json:"name"`
}

type itemPayload struct {
	Ticker string `json:"ticker"`
}

// GetWatchlists lista todas as watchlists do usuário ativo (Cria 'Favoritos' se vazia).
func (h *Handler) GetWatchlists(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	lists, err := h.service.GetWatchlists(r.Context(), userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao recuperar favoritos")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, lists)
}

// CreateWatchlist cria uma nova lista customizada.
func (h *Handler) CreateWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	var payload watchlistPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	wList, err := h.service.CreateWatchlist(r.Context(), userID, payload.Name)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, wList)
}

// GetWatchlist exibe uma watchlist e suas cotações intradiárias integradas.
func (h *Handler) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	watchlistID := chi.URLParam(r, "id")
	if watchlistID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da watchlist é obrigatório")
		return
	}

	wList, err := h.service.GetWatchlist(r.Context(), watchlistID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, wList)
}

// DeleteWatchlist apaga uma lista de favoritos.
func (h *Handler) DeleteWatchlist(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	watchlistID := chi.URLParam(r, "id")
	if watchlistID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da watchlist é obrigatório")
		return
	}

	err := h.service.DeleteWatchlist(r.Context(), watchlistID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Lista de favoritos excluída com sucesso"})
}

// AddAsset adiciona um ativo à watchlist correspondente.
func (h *Handler) AddAsset(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	watchlistID := chi.URLParam(r, "id")
	if watchlistID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da watchlist é obrigatório")
		return
	}

	var payload itemPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	item, err := h.service.AddAssetToWatchlist(r.Context(), watchlistID, userID, payload.Ticker)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, item)
}

// RemoveAsset desvincula um ativo da watchlist correspondente.
func (h *Handler) RemoveAsset(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	watchlistID := chi.URLParam(r, "id")
	ticker := chi.URLParam(r, "ticker")
	if watchlistID == "" || ticker == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "Parâmetros de rota inválidos")
		return
	}

	err := h.service.RemoveAssetFromWatchlist(r.Context(), watchlistID, userID, ticker)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Ativo removido dos favoritos com sucesso"})
}
