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
		r.Get("/monthly-yields", h.getMonthlyYields)
		r.Post("/assets", h.createAsset)
		r.Post("/bulk", h.bulkImportTransactions)
		r.Delete("/assets/{assetID}", h.deleteAsset)
		r.Post("/assets/{assetID}/transactions", h.createTransaction)
		r.Put("/transactions/{txID}", h.updateTransaction)
		r.Delete("/transactions/{txID}", h.deleteTransaction)
	})

	r.Route("/portfolios/{portfolioID}/treasury", func(r chi.Router) {
		r.Get("/positions", h.getTreasuryPositions)
		r.Get("/transactions", h.getTreasuryTransactions)
		r.Post("/transactions", h.createTreasuryTransaction)
		r.Put("/transactions/{txID}", h.updateTreasuryTransaction)
		r.Delete("/transactions/{txID}", h.deleteTreasuryTransaction)
		r.Get("/performance", h.getTreasuryPerformance)
	})
}

func (h *Handler) getMonthlyYields(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	yields, err := h.service.CalculateMonthlyYields(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if yields == nil {
		yields = []MonthlyYield{}
	}
	json.NewEncoder(w).Encode(yields)
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

func (h *Handler) bulkImportTransactions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "ID da carteira é obrigatório", http.StatusBadRequest)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Erro ao ler o formulário", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Arquivo CSV é obrigatório", http.StatusBadRequest)
		return
	}
	defer file.Close()

	res, err := h.service.BulkAddTransactions(r.Context(), portfolioID, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(res.Errors) > 0 {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) getTreasuryPositions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	positions, err := h.service.GetTreasuryPositions(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(positions)
}

func (h *Handler) getTreasuryTransactions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	transactions, err := h.service.GetTreasuryTransactions(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

func (h *Handler) createTreasuryTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	var req TreasuryTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateTreasuryTransaction(r.Context(), portfolioID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) getTreasuryPerformance(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		http.Error(w, "portfolioID is required", http.StatusBadRequest)
		return
	}

	performance, err := h.service.GetTreasuryPerformance(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(performance)
}

func (h *Handler) updateTreasuryTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	txID := chi.URLParam(r, "txID")
	if portfolioID == "" || txID == "" {
		http.Error(w, "portfolioID and txID are required", http.StatusBadRequest)
		return
	}

	var req TreasuryTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.service.UpdateTreasuryTransaction(r.Context(), portfolioID, txID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteTreasuryTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	txID := chi.URLParam(r, "txID")
	if portfolioID == "" || txID == "" {
		http.Error(w, "portfolioID and txID are required", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteTreasuryTransaction(r.Context(), portfolioID, txID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
