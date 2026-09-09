package fixedincome

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
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

func (h *Handler) portfolioAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(auth.UserIDKey).(string)
		if !ok || userID == "" {
			httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
			return
		}
		portfolioID := chi.URLParam(r, "portfolioID")
		if portfolioID == "" {
			httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
			return
		}
		if err := h.repo.ValidatePortfolioOwnership(r.Context(), portfolioID, userID); err != nil {
			httputils.RespondWithError(w, http.StatusNotFound, "Carteira não encontrada ou permissão negada")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/portfolios/{portfolioID}/fixed-income", func(r chi.Router) {
		r.Use(h.portfolioAuthMiddleware)
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
		r.Use(h.portfolioAuthMiddleware)
		r.Get("/positions", h.getTreasuryPositions)
		r.Get("/transactions", h.getTreasuryTransactions)
		r.Post("/transactions", h.createTreasuryTransaction)
		r.Put("/transactions/{txID}", h.updateTreasuryTransaction)
		r.Delete("/transactions/{txID}", h.deleteTreasuryTransaction)
		r.Get("/performance", h.getTreasuryPerformance)
		r.Get("/monthly-yields", h.getTreasuryMonthlyYields)
	})
}

func (h *Handler) getMonthlyYields(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	yields, err := h.service.CalculateMonthlyYields(r.Context(), portfolioID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if yields == nil {
		yields = []MonthlyYield{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, yields)
}

func (h *Handler) getPositions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	positions, err := h.service.GetPortfolioPositions(r.Context(), portfolioID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if positions == nil {
		positions = []Position{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, positions)
}

func (h *Handler) getPerformance(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "ALL"
	}

	performance, err := h.service.GetPortfolioPerformance(r.Context(), portfolioID, period)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if performance == nil {
		performance = []PerformancePoint{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, performance)
}

func (h *Handler) createAsset(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")

	var asset Asset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	asset.PortfolioID = portfolioID

	created, err := h.service.CreateAsset(r.Context(), &asset)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, created)
}

func (h *Handler) deleteAsset(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	assetID := chi.URLParam(r, "assetID")

	asset, err := h.repo.GetAssetByID(r.Context(), assetID)
	if err != nil || asset == nil || asset.PortfolioID != portfolioID {
		httputils.RespondWithError(w, http.StatusNotFound, "Ativo não encontrado na carteira informada")
		return
	}

	err = h.repo.DeleteAsset(r.Context(), assetID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	assetID := chi.URLParam(r, "assetID")

	asset, err := h.repo.GetAssetByID(r.Context(), assetID)
	if err != nil || asset == nil || asset.PortfolioID != portfolioID {
		httputils.RespondWithError(w, http.StatusNotFound, "Ativo não encontrado na carteira informada")
		return
	}

	var tx Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	tx.AssetID = assetID

	created, err := h.service.CreateTransaction(r.Context(), &tx)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, created)
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
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx := Transaction{
		Type:   payload.Type,
		Amount: payload.Amount,
		Date:   payload.Date,
	}

	err := h.service.UpdateTransaction(r.Context(), portfolioID, txID, &tx, payload.MaturityDate)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			httputils.RespondWithError(w, http.StatusForbidden, err.Error())
		} else {
			httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
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
		if strings.Contains(err.Error(), "unauthorized") {
			httputils.RespondWithError(w, http.StatusForbidden, err.Error())
		} else {
			httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) bulkImportTransactions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Erro ao ler o formulário")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Arquivo CSV é obrigatório")
		return
	}
	defer file.Close()

	res, err := h.service.BulkAddTransactions(r.Context(), portfolioID, file)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status := http.StatusOK
	if len(res.Errors) > 0 {
		status = http.StatusPartialContent
	}
	httputils.RespondWithJSON(w, status, res)
}

func (h *Handler) getTreasuryPositions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	positions, err := h.service.GetTreasuryPositions(r.Context(), portfolioID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if positions == nil {
		positions = []TreasuryPosition{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, positions)
}

func (h *Handler) getTreasuryTransactions(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	transactions, err := h.service.GetTreasuryTransactions(r.Context(), portfolioID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if transactions == nil {
		transactions = []TreasuryTxRequest{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, transactions)
}

func (h *Handler) createTreasuryTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	var req TreasuryTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.service.CreateTreasuryTransaction(r.Context(), portfolioID, &req)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, res)
}

func (h *Handler) getTreasuryPerformance(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	performance, err := h.service.GetTreasuryPerformance(r.Context(), portfolioID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if performance == nil {
		performance = []TreasuryPerfPoint{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, performance)
}

func (h *Handler) updateTreasuryTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	txID := chi.URLParam(r, "txID")
	if portfolioID == "" || txID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID and txID are required")
		return
	}

	var req TreasuryTxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.service.UpdateTreasuryTransaction(r.Context(), portfolioID, txID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			httputils.RespondWithError(w, http.StatusForbidden, err.Error())
		} else {
			httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteTreasuryTransaction(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	txID := chi.URLParam(r, "txID")
	if portfolioID == "" || txID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID and txID are required")
		return
	}

	err := h.service.DeleteTreasuryTransaction(r.Context(), portfolioID, txID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			httputils.RespondWithError(w, http.StatusForbidden, err.Error())
		} else {
			httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getTreasuryMonthlyYields(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioID")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "portfolioID is required")
		return
	}

	yields, err := h.service.GetTreasuryMonthlyYields(r.Context(), portfolioID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if yields == nil {
		yields = []MonthlyYield{}
	}
	httputils.RespondWithJSON(w, http.StatusOK, yields)
}
