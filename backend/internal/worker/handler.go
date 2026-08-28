package worker

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	manager *Manager
}

func NewHandler(m *Manager) *Handler {
	return &Handler{manager: m}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.ListWorkers)
	r.Post("/{name}/trigger", h.TriggerWorker)
}

func (h *Handler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	infos := h.manager.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

func (h *Handler) TriggerWorker(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	ok := h.manager.Trigger(name)
	if !ok {
		http.Error(w, "Worker não encontrado", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Worker disparado com sucesso"}`))
}
