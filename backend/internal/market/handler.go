package market

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler expõe os endpoints HTTP para busca e cotação.
type Handler struct {
	service *Service
}

// NewHandler cria uma nova instância de Handler de mercado.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetQuote obtém e retorna os dados de cotação em tempo real de um ativo.
func (h *Handler) GetQuote(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")
	if ticker == "" {
		h.respondWithError(w, http.StatusBadRequest, "Símbolo do ativo (ticker) é obrigatório")
		return
	}

	quote, err := h.service.GetQuote(r.Context(), ticker)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
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
