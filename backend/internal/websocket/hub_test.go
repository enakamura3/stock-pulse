package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMarketService struct {
	mock.Mock
}

func (m *MockMarketService) GetQuote(ctx context.Context, ticker string) (*market.Quote, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*market.Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestHub_ClientLifecycleAndSubscription(t *testing.T) {
	ms := new(MockMarketService)
	// Return a dummy quote
	ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Symbol: "AAPL", Price: 150.0}, nil)

	hub := NewHub(ms)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Start(ctx)
	defer cancel()

	handler := NewHandler(hub)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock User ID in context for auth
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "test_user"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WS
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	defer ws.Close()

	// 1. Subscribe to AAPL
	subMsg := WSMessage{
		Action:  "subscribe",
		Symbols: []string{"AAPL"},
	}
	err = ws.WriteJSON(subMsg)
	assert.NoError(t, err)

	// Wait for subscription to process and force broadcast
	time.Sleep(50 * time.Millisecond)
	go hub.broadcastQuotes(context.Background())

	// Read message (could be a ping or quote)
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("Error reading message: %v", err)
		}

		// Unmarshal
		var payload map[string]interface{}
		if err := json.Unmarshal(p, &payload); err == nil && payload["type"] == "quote" {
			data := payload["data"].(map[string]interface{})
			assert.Equal(t, "AAPL", data["symbol"])
			assert.Equal(t, 150.0, data["price"])
			break // Got our quote
		}
	}

	// 2. Unsubscribe from AAPL
	unsubMsg := WSMessage{
		Action:  "unsubscribe",
		Symbols: []string{"AAPL"},
	}
	err = ws.WriteJSON(unsubMsg)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Test handler unauthorized
	req := httptest.NewRequest("GET", "/ws", nil)
	rec := httptest.NewRecorder()
	handler.ServeWS(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Test handler bad upgrade (authorized but missing WS headers)
	req2 := httptest.NewRequest("GET", "/ws", nil)
	req2 = req2.WithContext(context.WithValue(req2.Context(), auth.UserIDKey, "test_user"))
	rec2 := httptest.NewRecorder()
	handler.ServeWS(rec2, req2)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
}

func TestHub_MaxConnectionsPerUser(t *testing.T) {
	ms := new(MockMarketService)
	hub := NewHub(ms)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Start(ctx)
	defer cancel()

	handler := NewHandler(hub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_limit_test"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	// Abre até MaxConnectionsPerUser conexões (5)
	conns := make([]*websocket.Conn, MaxConnectionsPerUser)
	for i := 0; i < MaxConnectionsPerUser; i++ {
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		assert.NoError(t, err)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		conns[i] = ws
	}

	// Aguarda processamento do registro no Hub
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, MaxConnectionsPerUser, hub.ActiveConnectionsCount("user_limit_test"))

	// A 6ª conexão deve ser rejeitada com HTTP 429
	_, resp6, err6 := websocket.DefaultDialer.Dial(wsURL, header)
	assert.Error(t, err6)
	if resp6 != nil {
		assert.Equal(t, http.StatusTooManyRequests, resp6.StatusCode)
		if resp6.Body != nil {
			_ = resp6.Body.Close()
		}
	}

	// Fecha uma conexão e verifica que uma nova é permitida
	_ = conns[0].Close()
	time.Sleep(50 * time.Millisecond)

	wsNew, respNew, errNew := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, errNew)
	if respNew != nil && respNew.Body != nil {
		_ = respNew.Body.Close()
	}
	defer wsNew.Close()

	// Limpa conexões abertas
	for i := 1; i < MaxConnectionsPerUser; i++ {
		_ = conns[i].Close()
	}
}

func TestHub_MaxSubscriptionsPerClient(t *testing.T) {
	ms := new(MockMarketService)
	hub := NewHub(ms)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Start(ctx)
	defer cancel()

	handler := NewHandler(hub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_sub_test"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	defer ws.Close()

	// Tenta assinar 60 tickers (excedendo o limite de 50)
	tickers := make([]string, 60)
	for i := 0; i < 60; i++ {
		tickers[i] = "TICK" + strings.Repeat("X", 2) + string(rune('A'+(i%26))) + string(rune('0'+(i/26)))
	}

	subMsg := WSMessage{
		Action:  "subscribe",
		Symbols: tickers,
	}
	err = ws.WriteJSON(subMsg)
	assert.NoError(t, err)

	// O cliente deve receber uma mensagem com erro sobre o limite
	_ = ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	foundError := false
	for {
		_, p, readErr := ws.ReadMessage()
		if readErr != nil {
			break
		}
		var payload map[string]interface{}
		if json.Unmarshal(p, &payload) == nil {
			if payload["type"] == "error" && strings.Contains(payload["error"].(string), "Limite máximo") {
				foundError = true
				break
			}
		}
	}
	assert.True(t, foundError, "deveria ter recebido mensagem de erro por exceder limite de inscrições")
}

func TestHub_MaxMessageSizeLimit(t *testing.T) {
	ms := new(MockMarketService)
	hub := NewHub(ms)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Start(ctx)
	defer cancel()

	handler := NewHandler(hub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_msg_size_test"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	defer ws.Close()

	// Envia mensagem maior que MaxMessageSize (4096 bytes)
	hugePayload := strings.Repeat("A", 5000)
	err = ws.WriteMessage(websocket.TextMessage, []byte(hugePayload))
	assert.NoError(t, err)

	// A conexão deve fechar pelo servidor por exceder ReadLimit
	_ = ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, _, readErr := ws.ReadMessage()
	assert.Error(t, readErr, "servidor deve fechar conexão se a mensagem exceder MaxMessageSize")
}

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]market.DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}
