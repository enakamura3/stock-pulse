package market

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
)

// MarketService define a interface que o Handler consome.
type MarketService interface {
	GetQuote(ctx context.Context, ticker string) (*Quote, error)
	GetQuoteWithCacheStatus(ctx context.Context, symbol string) (*Quote, bool, error)
	SearchAssets(ctx context.Context, query string) ([]SearchResult, error)
	GetBenchmarks(ctx context.Context) (*MarketBenchmarks, error)
	InvalidateQuoteCache(ctx context.Context, symbols []string) (int64, error)
}

// Handler expõe os endpoints HTTP para busca e cotação.
type Handler struct {
	service MarketService
}

// NewHandler cria uma nova instância de Handler de mercado.
func NewHandler(service MarketService) *Handler {
	return &Handler{service: service}
}

// GetQuote obtém e retorna os dados de cotação em tempo real de um ativo.
func (h *Handler) GetQuote(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")
	if ticker == "" {
		httputils.RespondWithError(w, http.StatusBadRequest, "Símbolo do ativo (ticker) é obrigatório")
		return
	}

	quote, hit, err := h.service.GetQuoteWithCacheStatus(r.Context(), ticker)
	if err != nil {
		httputils.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	if hit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	httputils.RespondWithJSON(w, http.StatusOK, quote)
}

// Search realiza a busca de ativos autocomplete.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		httputils.RespondWithJSON(w, http.StatusOK, []SearchResult{})
		return
	}

	results, err := h.service.SearchAssets(r.Context(), query)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao efetuar busca no provedor de mercado")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, results)
}

// GetBenchmarks retorna os principais benchmarks de mercado para comparação intradiária.
func (h *Handler) GetBenchmarks(w http.ResponseWriter, r *http.Request) {
	benchmarks, err := h.service.GetBenchmarks(r.Context())
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao obter benchmarks de mercado")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, benchmarks)
}

// InvalidateCacheRequest representa a requisição para invalidar o cache de cotações.
type InvalidateCacheRequest struct {
	Symbols []string `json:"symbols"`
}

// InvalidateCacheResponse representa a resposta da invalidação de cache.
type InvalidateCacheResponse struct {
	Message string `json:"message"`
	Removed int64  `json:"removed"`
}

// InvalidateCache invalida as chaves de cotação e benchmarks no Redis.
func (h *Handler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	var req InvalidateCacheRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	removed, err := h.service.InvalidateQuoteCache(r.Context(), req.Symbols)
	if err != nil {
		httputils.RespondWithError(w, http.StatusInternalServerError, "Erro ao invalidar cache de cotações")
		return
	}

	httputils.RespondWithJSON(w, http.StatusOK, InvalidateCacheResponse{
		Message: "Cache de cotações invalidado com sucesso",
		Removed: removed,
	})
}
