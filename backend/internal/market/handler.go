package market

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MarketService define a interface que o Handler consome.
type MarketService interface {
	GetQuote(ctx context.Context, ticker string) (*Quote, error)
	GetQuoteWithCacheStatus(ctx context.Context, symbol string) (*Quote, bool, error)
	SearchAssets(ctx context.Context, query string) ([]SearchResult, error)
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
		h.respondWithError(w, http.StatusBadRequest, "Símbolo do ativo (ticker) é obrigatório")
		return
	}

	quote, hit, err := h.service.GetQuoteWithCacheStatus(r.Context(), ticker)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	if hit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	h.respondWithJSON(w, http.StatusOK, quote)
}

// Search realiza a busca de ativos autocomplete.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.respondWithJSON(w, http.StatusOK, []SearchResult{})
		return
	}

	results, err := h.service.SearchAssets(r.Context(), query)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao efetuar busca no provedor de mercado")
		return
	}

	h.respondWithJSON(w, http.StatusOK, results)
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
