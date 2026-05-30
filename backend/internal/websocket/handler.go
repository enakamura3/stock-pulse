package websocket

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Valida a origem do request verificando FRONTEND_URL
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		frontendURL := os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "http://localhost:3000"
		}
		return origin == frontendURL
	},
}

// Handler gerencia o endpoint HTTP para estabelecer conexões WebSocket.
type Handler struct {
	Hub *Hub
}

// NewHandler inicializa o WebSocket Handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		Hub: hub,
	}
}

// ServeWS faz o upgrade de HTTP para WSS de conexões seguras e registra o novo cliente no Hub.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		log.Printf("[WS] Tentativa de conexão WebSocket não autorizada")
		http.Error(w, "Sessão não autorizada ou expirada", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Falha ao realizar upgrade da conexão: %v", err)
		return
	}

	// Inicializa e registra o cliente WebSocket no Hub central
	client := NewClient(h.Hub, conn, userID)
	h.Hub.register <- client

	// Dispara Goroutines paralelas para bombear mensagens de entrada e saída
	go client.WritePump()
	go client.ReadPump()
}
