package history

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

type OwnershipValidator interface {
	ValidatePortfolioOwnership(ctx context.Context, portfolioID, userID string) error
}

type Handler struct {
	service   Service
	validator OwnershipValidator
}

func NewHandler(service Service, validator OwnershipValidator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/portfolios/{portfolioID}/history", func(r chi.Router) {
		r.Get("/", h.getHistory)
	})
}

func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")

	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.validator.ValidatePortfolioOwnership(r.Context(), portfolioID, userID); err != nil {
		httputils.RespondWithError(w, http.StatusNotFound, "Carteira não encontrada ou permissão negada")
		return
	}

	history, err := h.service.GetPortfolioHistory(r.Context(), portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "failed to get history")
		return
	}

	if history == nil {
		history = []UnifiedTransaction{}
	}

	httputils.RespondWithJSON(w, http.StatusOK, history)
}
