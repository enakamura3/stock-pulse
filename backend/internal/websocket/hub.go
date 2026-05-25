package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/onigiri/stockpulse/backend/internal/market"
)

// Client representa uma conexão WebSocket ativa de um usuário.
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string

	subscribed map[string]bool
	mu         sync.Mutex
}

// WSMessage representa o formato de mensagem recebido do frontend.
type WSMessage struct {
	Action  string   `json:"action"`  // "subscribe" ou "unsubscribe"
	Symbols []string `json:"symbols"`
}

// NewClient inicializa um novo cliente WebSocket.
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		Hub:        hub,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		UserID:     userID,
		subscribed: make(map[string]bool),
	}
}

// ReadPump escuta mensagens de entrada enviadas pelo cliente.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512) // Limite de tamanho de mensagem por segurança
	// Configura limites de timeout de leitura
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Erro inesperado ao ler mensagem do cliente WebSocket", "user_id", c.UserID, "error", err)
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			slog.Error("Erro ao deserializar mensagem do cliente WebSocket", "user_id", c.UserID, "error", err)
			continue
		}

		c.mu.Lock()
		switch wsMsg.Action {
		case "subscribe":
			// Sobrescreve as assinaturas anteriores para manter a conexão em sincronia exata com a UI atual
			c.subscribed = make(map[string]bool)
			for _, sym := range wsMsg.Symbols {
				if sym != "" {
					c.subscribed[sym] = true
					slog.Info("Cliente assinou ticker via WebSocket", "user_id", c.UserID, "ticker", sym)
				}
			}
		case "unsubscribe":
			for _, sym := range wsMsg.Symbols {
				delete(c.subscribed, sym)
				slog.Info("Cliente cancelou assinatura do ticker via WebSocket", "user_id", c.UserID, "ticker", sym)
			}
		}
		c.mu.Unlock()
	}
}

// WritePump envia mensagens de saída do Hub para a conexão WebSocket real.
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second) // Envia pings periódicos
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// O Hub fechou o canal.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Consome mensagens extras enfileiradas de uma vez
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte("\n"))
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// IsSubscribed retorna se o cliente está assinado em um ticker específico de forma thread-safe.
func (c *Client) IsSubscribed(symbol string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.subscribed[symbol]
}

// Hub coordena as conexões ativas e a orquestração do broadcast de cotações em tempo real.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	marketSvc  *market.Service
	mu         sync.RWMutex
}

// NewHub inicializa o WebSocket Hub central.
func NewHub(marketSvc *market.Service) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		marketSvc:  marketSvc,
	}
}

// Start gerencia o ciclo de vida de conexões e o loop periódico de broadcast de cotações (5 segundos).
func (h *Hub) Start(ctx context.Context) {
	broadcastTicker := time.NewTicker(5 * time.Second) // Broadcast a cada 5 segundos
	defer broadcastTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("Novo cliente WebSocket conectado", "clients_count", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				slog.Info("Cliente WebSocket desconectado", "clients_count", len(h.clients))
			}
			h.mu.Unlock()

		case <-broadcastTicker.C:
			h.broadcastQuotes(ctx)

		case <-ctx.Done():
			slog.Info("Encerrando o WebSocket Hub central de cotações...")
			return
		}
	}
}

// broadcastQuotes reúne todos os tickers em foco por conexões ativas e atualiza-os.
func (h *Hub) broadcastQuotes(ctx context.Context) {
	h.mu.RLock()
	if len(h.clients) == 0 {
		h.mu.RUnlock()
		return
	}

	// 1. Coleta a lista única de todos os tickers assinados ativamente
	uniqueTickers := make(map[string]bool)
	for client := range h.clients {
		client.mu.Lock()
		for ticker := range client.subscribed {
			uniqueTickers[ticker] = true
		}
		client.mu.Unlock()
	}
	h.mu.RUnlock()

	if len(uniqueTickers) == 0 {
		return
	}

	// 2. Busca a cotação de cada ativo ativamente assinado
	var wg sync.WaitGroup
	var mu sync.Mutex
	quotesMap := make(map[string]*market.Quote)

	for ticker := range uniqueTickers {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			quote, err := h.marketSvc.GetQuote(ctx, t)
			if err != nil {
				slog.Error("Falha ao obter cotação para o WebSocket", "ticker", t, "error", err)
				return
			}
			mu.Lock()
			quotesMap[t] = quote
			mu.Unlock()
		}(ticker)
	}
	wg.Wait()

	// 3. Distribui os preços atualizados aos respectivos assinantes interessados
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		for ticker, quote := range quotesMap {
			if client.IsSubscribed(ticker) {
				payload := map[string]interface{}{
					"type": "quote",
					"data": quote,
				}
				jsonBytes, err := json.Marshal(payload)
				if err != nil {
					continue
				}
				// Envia para o canal individual de forma não-bloqueante
				select {
				case client.Send <- jsonBytes:
				default:
					slog.Warn("Canal de envio do cliente WebSocket cheio. Ignorando broadcast de preço", "user_id", client.UserID, "ticker", ticker)
				}
			}
		}
	}
}
