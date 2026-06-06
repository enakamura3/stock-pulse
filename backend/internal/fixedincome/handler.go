package fixedincome

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service Service
	repo    Repository
}

func NewHandler(service Service, repo Repository) *Handler {
	return &Handler{
		service: service,
		repo:    repo,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/portfolios/{portfolioID}/fixed-income", func(r chi.Router) {
		r.Get("/positions", h.getPositions)
		r.Get("/performance", h.getPerformance)
		r.Post("/assets", h.createAsset)
		r.Delete("/assets/{assetID}", h.deleteAsset)
		r.Post("/assets/{assetID}/transactions", h.createTransaction)
		r.Put("/transactions/{txID}", h.updateTransaction)
		r.Delete("/transactions/{txID}", h.deleteTransaction)
	})
}

func (h *Handler) getPositions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	positions, err := h.service.GetPortfolioPositions(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if positions == nil {
		positions = []Position{}
	}
	json.NewEncoder(w).Encode(positions)
}

func (h *Handler) getPerformance(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "ALL"
	}

	performance, err := h.service.GetPortfolioPerformance(r.Context(), portfolioID, period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if performance == nil {
		performance = []PerformancePoint{}
	}
	json.NewEncoder(w).Encode(performance)
}

func (h *Handler) createAsset(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	
	var asset Asset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	asset.PortfolioID = portfolioID

	created, err := h.service.CreateAsset(r.Context(), &asset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) deleteAsset(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetID")
	
	err := h.repo.DeleteAsset(r.Context(), assetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetID")
	
	var tx Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tx.AssetID = assetID

	created, err := h.service.CreateTransaction(r.Context(), &tx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) updateTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	txID := chi.URLParam(r, "txID")
	
	var payload struct {
		Type         string     `json:"type"`
		Amount       float64    `json:"amount"`
		Date         time.Time  `json:"date"`
		MaturityDate *time.Time `json:"maturity_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx := Transaction{
		Type:   payload.Type,
		Amount: payload.Amount,
		Date:   payload.Date,
	}

	err := h.service.UpdateTransaction(r.Context(), portfolioID, txID, &tx, payload.MaturityDate)
	if err != nil {
		if err.Error() == "unauthorized: transaction does not belong to the portfolio" {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	txID := chi.URLParam(r, "txID")
	
	err := h.service.DeleteTransaction(r.Context(), portfolioID, txID)
	if err != nil {
		if err.Error() == "unauthorized: transaction does not belong to the portfolio" {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
