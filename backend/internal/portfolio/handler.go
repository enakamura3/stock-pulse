package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stockpulse/backend/internal/auth"
)

// PortfolioService define as operações que o Handler espera.
type PortfolioService interface {
	CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error)
	GetPortfolios(ctx context.Context, userID string) ([]Portfolio, error)
	GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*Portfolio, []Position, error)
	AddTransaction(ctx context.Context, userID string, tx *Transaction) (*Transaction, error)
	DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error
	DeletePortfolio(ctx context.Context, id, userID string) error
	GetPortfolioPerformance(ctx context.Context, portfolioID, userID, period string) ([]PerformancePoint, error)
	
	// Utilizado especificamente pelo Handler para recuperar transações puras
	repoGetTransactionsByPortfolioID(ctx context.Context, portfolioID, userID string) ([]Transaction, error)
}

// Removido do handler.go
// Handler expõe endpoints HTTP seguros para o módulo de Portfólios.
type Handler struct {
	service PortfolioService
}

// NewHandler cria uma nova instância de Handler.
func NewHandler(service PortfolioService) *Handler {
	return &Handler{service: service}
}

type portfolioPayload struct {
	Name         string `json:"name"`
	BaseCurrency string `json:"base_currency"`
}

type transactionPayload struct {
	Ticker       string  `json:"ticker"`
	Type         string  `json:"type"` // "BUY" ou "SELL"
	Quantity     float64 `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	ExchangeRate float64 `json:"exchange_rate"`
	ExecutedAt   string  `json:"executed_at"` // formato "YYYY-MM-DD"
}

// GetPortfolios lista todos os portfólios do usuário (cria "Principal" se vazio).
func (h *Handler) GetPortfolios(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	lists, err := h.service.GetPortfolios(r.Context(), userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao recuperar carteiras")
		return
	}

	h.respondWithJSON(w, http.StatusOK, lists)
}

// CreatePortfolio cria uma nova carteira para o usuário.
func (h *Handler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	var payload portfolioPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	p, err := h.service.CreatePortfolio(ctxOrDefault(r), userID, payload.Name, payload.BaseCurrency)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusCreated, p)
}

// GetPortfolio retorna o consolidado detalhado (posições e lucratividade) de uma carteira.
func (h *Handler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	p, positions, err := h.service.GetPortfolioDetails(r.Context(), portfolioID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"portfolio": p,
		"positions": positions,
	})
}

// DeletePortfolio apaga uma carteira.
func (h *Handler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	err := h.service.DeletePortfolio(r.Context(), portfolioID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Carteira excluída com sucesso"})
}

// GetTransactions lista todas as transações cadastradas na carteira.
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	txs, err := h.service.repoGetTransactionsByPortfolioID(r.Context(), portfolioID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao recuperar transações")
		return
	}

	h.respondWithJSON(w, http.StatusOK, txs)
}

// AddTransaction registra uma nova operação de compra/venda.
func (h *Handler) AddTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	var payload transactionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	payload.Type = strings.ToUpper(strings.TrimSpace(payload.Type))
	if payload.Type != "BUY" && payload.Type != "SELL" {
		h.respondWithError(w, http.StatusBadRequest, "Tipo de transação deve ser BUY ou SELL")
		return
	}

	if payload.Quantity <= 0 || payload.UnitPrice <= 0 {
		h.respondWithError(w, http.StatusBadRequest, "Quantidade e preço unitário devem ser maiores que zero")
		return
	}

	// Trata parsing de datas históricas com fallback seguro
	execTime, err := time.Parse("2006-01-02", payload.ExecutedAt)
	if err != nil {
		execTime, err = time.Parse(time.RFC3339, payload.ExecutedAt)
		if err != nil {
			execTime = time.Now()
		}
	}

	// Configura taxa de câmbio padrão se nula ou vazia
	rate := payload.ExchangeRate
	if rate <= 0 {
		rate = 1.0
	}

	tx := &Transaction{
		PortfolioID:  portfolioID,
		Ticker:       payload.Ticker,
		Type:         payload.Type,
		Quantity:     payload.Quantity,
		UnitPrice:    payload.UnitPrice,
		ExchangeRate: rate,
		ExecutedAt:   execTime.UTC(),
	}

	savedTx, err := h.service.AddTransaction(r.Context(), userID, tx)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusCreated, savedTx)
}

// DeleteTransaction remove uma operação financeira cadastrada.
func (h *Handler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	txID := chi.URLParam(r, "txId")
	if portfolioID == "" || txID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira e ID da transação são obrigatórios")
		return
	}

	err := h.service.DeleteTransaction(r.Context(), txID, portfolioID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Transação removida com sucesso"})
}

// GetPerformance retorna a evolução patrimonial diária consolidada (LOCF).
func (h *Handler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "ALL"
	}

	points, err := h.service.GetPortfolioPerformance(r.Context(), portfolioID, userID, period)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, points)
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

func ctxOrDefault(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
