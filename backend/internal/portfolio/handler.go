package portfolio

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

// PortfolioService define as operações que o Handler espera.
type PortfolioService interface {
	CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error)
	GetPortfolios(ctx context.Context, userID string) ([]Portfolio, error)
	GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*Portfolio, []Position, error)
	SetDefaultPortfolio(ctx context.Context, portfolioID, userID string) error
	AddTransaction(ctx context.Context, userID string, tx *Transaction) (*Transaction, error)
	UpdateTransaction(ctx context.Context, userID, portfolioID, txID string, tx *Transaction) error
	BulkAddTransactions(ctx context.Context, userID, portfolioID string, file multipart.File) (*BulkImportResult, error)
	DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error
	DeletePortfolio(ctx context.Context, id, userID string) error
	GetPortfolioPerformance(ctx context.Context, portfolioID, userID, period string, filterTickers []string) ([]PerformancePoint, error)
	GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]CalculatedDividend, error)

	// Utilizado especificamente pelo Handler para recuperar transações puras
	GetPortfolioTransactions(ctx context.Context, portfolioID, userID string) ([]Transaction, error)
	GetFixedIncomeService() fixedincome.Service
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
	Fee          float64 `json:"fee"`
	ExchangeRate float64 `json:"exchange_rate"`
	ExecutedAt   string  `json:"executed_at"` // formato "YYYY-MM-DD"
}

// GetPortfolios lista todos os portfólios do usuário (cria "Principal" se vazio).
func (h *Handler) GetPortfolios(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	lists, err := h.service.GetPortfolios(r.Context(), userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao recuperar carteiras")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, lists)
}

// CreatePortfolio cria uma nova carteira para o usuário.
func (h *Handler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	var payload portfolioPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	p, err := h.service.CreatePortfolio(ctxOrDefault(r), userID, payload.Name, payload.BaseCurrency)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, p)
}

// SetDefaultPortfolio define uma carteira como padrão para o usuário.
func (h *Handler) SetDefaultPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	err := h.service.SetDefaultPortfolio(r.Context(), portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Carteira definida como padrão com sucesso"})
}

// GetPortfolio retorna o consolidado detalhado (posições e lucratividade) de uma carteira.
func (h *Handler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	p, positions, err := h.service.GetPortfolioDetails(r.Context(), portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"portfolio": p,
		"positions": positions,
	})
}

// DeletePortfolio apaga uma carteira.
func (h *Handler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	err := h.service.DeletePortfolio(r.Context(), portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Carteira excluída com sucesso"})
}

// GetTransactions lista todas as transações cadastradas na carteira.
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	txs, err := h.service.GetPortfolioTransactions(r.Context(), portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao recuperar transações")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, txs)
}

// AddTransaction registra uma nova operação de compra/venda.
func (h *Handler) AddTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	var payload transactionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	payload.Type = strings.ToUpper(strings.TrimSpace(payload.Type))
	if payload.Type != "BUY" && payload.Type != "SELL" && payload.Type != "SPLIT" && payload.Type != "REVERSE_SPLIT" && payload.Type != "BONUS" {
		httputils.RespondWithError(w, http.StatusBadRequest, "Tipo de transação deve ser BUY, SELL, SPLIT, REVERSE_SPLIT ou BONUS")
		return
	}

	if payload.Quantity <= 0 || (payload.Type != "SPLIT" && payload.Type != "REVERSE_SPLIT" && payload.Type != "BONUS" && payload.UnitPrice <= 0) {
		httputils.RespondWithError(w, http.StatusBadRequest, "Quantidade deve ser maior que zero (e preço unitário também, exceto para splits e bônus)")
		return
	}

	if payload.Fee < 0 {
		httputils.RespondWithError(w, http.StatusBadRequest, "Taxa/Corretagem não pode ser negativa")
		return
	}

	fee := payload.Fee
	if payload.Type == "SPLIT" || payload.Type == "REVERSE_SPLIT" || payload.Type == "BONUS" {
		fee = 0
	}

	// Trata parsing de datas históricas com fallback seguro
	execTime, err := time.Parse("2006-01-02", payload.ExecutedAt)
	if err != nil {
		execTime, err = time.Parse(time.RFC3339, payload.ExecutedAt)
		if err != nil {
			execTime = time.Now()
		}
	}

	// A taxa de câmbio agora pode ser 0 ou nula para que o service a busque automaticamente
	rate := payload.ExchangeRate

	tx := &Transaction{
		PortfolioID:  portfolioID,
		Ticker:       payload.Ticker,
		Type:         payload.Type,
		Quantity:     payload.Quantity,
		UnitPrice:    payload.UnitPrice,
		Fee:          fee,
		ExchangeRate: rate,
		ExecutedAt:   execTime.UTC(),
	}

	savedTx, err := h.service.AddTransaction(r.Context(), userID, tx)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusCreated, savedTx)
}

// DeleteTransaction remove uma operação financeira cadastrada.
func (h *Handler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	txID := chi.URLParam(r, "txId")
	if portfolioID == "" || txID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira e ID da transação são obrigatórios")
		return
	}

	err := h.service.DeleteTransaction(r.Context(), txID, portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Transação removida com sucesso"})
}

// GetPerformance retorna a evolução patrimonial diária consolidada (LOCF).
func (h *Handler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "ALL"
	}

	tickersParam := r.URL.Query().Get("tickers")
	var filterTickers []string
	if tickersParam != "" {
		filterTickers = strings.Split(tickersParam, ",")
	}

	points, err := h.service.GetPortfolioPerformance(r.Context(), portfolioID, userID, period, filterTickers)
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, points)
}

func ctxOrDefault(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

func (h *Handler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	txID := chi.URLParam(r, "txId")
	if portfolioID == "" || txID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira e da transação são obrigatórios")
		return
	}

	var payload transactionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Payload inválido")
		return
	}

	payload.Type = strings.ToUpper(strings.TrimSpace(payload.Type))
	if payload.Type != "BUY" && payload.Type != "SELL" && payload.Type != "SPLIT" && payload.Type != "REVERSE_SPLIT" && payload.Type != "BONUS" {
		httputils.RespondWithError(w, http.StatusBadRequest, "Tipo de transação deve ser BUY, SELL, SPLIT, REVERSE_SPLIT ou BONUS")
		return
	}

	if payload.Quantity <= 0 || (payload.Type != "SPLIT" && payload.Type != "REVERSE_SPLIT" && payload.Type != "BONUS" && payload.UnitPrice <= 0) {
		httputils.RespondWithError(w, http.StatusBadRequest, "Quantidade deve ser maior que zero (e preço unitário também, exceto para splits e bônus)")
		return
	}

	if payload.Fee < 0 {
		httputils.RespondWithError(w, http.StatusBadRequest, "Taxa/Corretagem não pode ser negativa")
		return
	}

	fee := payload.Fee
	if payload.Type == "SPLIT" || payload.Type == "REVERSE_SPLIT" || payload.Type == "BONUS" {
		fee = 0
	}

	execTime, err := time.Parse("2006-01-02", payload.ExecutedAt)
	if err != nil {
		execTime, err = time.Parse(time.RFC3339, payload.ExecutedAt)
		if err != nil {
			execTime = time.Now()
		}
	}

	rate := payload.ExchangeRate

	tx := &Transaction{
		Ticker:       payload.Ticker,
		Type:         payload.Type,
		Quantity:     payload.Quantity,
		UnitPrice:    payload.UnitPrice,
		Fee:          fee,
		ExchangeRate: rate,
		ExecutedAt:   execTime.UTC(),
	}

	if err := h.service.UpdateTransaction(ctxOrDefault(r), userID, portfolioID, txID, tx); err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Transação atualizada com sucesso"})
}

// BulkImportTransactions processa um arquivo CSV contendo múltiplas transações.
func (h *Handler) BulkImportTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Falha ao processar arquivo")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httputils.RespondWithError(w, http.StatusBadRequest, "Arquivo ausente ou inválido")
		return
	}
	defer file.Close()

	res, err := h.service.BulkAddTransactions(ctxOrDefault(r), userID, portfolioID, file)
	if err != nil {
		httputils.RespondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Erro durante a importação",
			"details": err.Error(),
		})
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, res)
}

// GetDividends calcula e retorna o histórico de proventos pagos baseando-se no histórico de transações.
func (h *Handler) GetDividends(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		httputils.RespondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	divs, err := h.service.GetPortfolioDividends(r.Context(), portfolioID, userID)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, divs)
}
