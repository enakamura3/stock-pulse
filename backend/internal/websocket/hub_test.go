package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

func TestHub_DirectRegisterLimitExceeded(t *testing.T) {
	ms := new(MockMarketService)
	hub := NewHub(ms)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Start(ctx)
	defer cancel()

	// Simula 5 conexões já ativas para o usuário
	hub.mu.Lock()
	hub.userConns["busy_user"] = MaxConnectionsPerUser
	hub.mu.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		assert.NoError(t, err)
		client := NewClient(hub, conn, "busy_user")
		hub.register <- client
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	defer ws.Close()

	// Como atingiu o limite no hub.register, o servidor deve enviar ClosePolicyViolation e fechar a conexão
	_ = ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, _, readErr := ws.ReadMessage()
	assert.Error(t, readErr)
}

func TestHub_BroadcastQuotes_EdgeCases(t *testing.T) {
	ms := new(MockMarketService)
	// Retorna erro para BAD_TICKER
	ms.On("GetQuote", mock.Anything, "BAD_TICKER").Return(nil, errors.New("provider timeout"))

	hub := NewHub(ms)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Start(ctx)
	defer cancel()

	// 1. Sem clientes inscritos: broadcastQuotes retorna sem erro
	hub.broadcastQuotes(ctx)

	// 2. Com cliente inscrito em BAD_TICKER
	handler := NewHandler(hub)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_err_test"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Add("Origin", "http://localhost:3000")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	defer ws.Close()

	subMsg := WSMessage{
		Action:  "subscribe",
		Symbols: []string{"BAD_TICKER"},
	}
	err = ws.WriteJSON(subMsg)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Força broadcast com ticker que dá erro
	hub.broadcastQuotes(ctx)
}

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]market.DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestHub_AdditionalEdgeCases(t *testing.T) {
	ms := new(MockMarketService)
	ms.On("GetQuote", mock.Anything, mock.Anything).Return(&market.Quote{Symbol: "MOCK", Price: 100}, nil).Maybe()
	hub := NewHub(ms).WithBroadcastInterval(20 * time.Millisecond).WithPingPeriod(20 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Start(ctx)

	handler := NewHandler(hub)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "user_edge_test"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{"Origin": []string{"http://localhost:3000"}}

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	assert.NoError(t, err)
	defer ws.Close()

	// 1. Send invalid JSON message to exercise Unmarshal error in ReadPump
	err = ws.WriteMessage(websocket.TextMessage, []byte("invalid json message"))
	assert.NoError(t, err)

	// 2. Send Pong to exercise SetPongHandler
	err = ws.WriteMessage(websocket.PongMessage, []byte("pong"))
	assert.NoError(t, err)

	time.Sleep(60 * time.Millisecond)

	// 3. Test client without subscriptions (len(uniqueTickers) == 0 in broadcastQuotes)
	hub.broadcastQuotes(ctx)

	// 4. Subscribe to MOCK
	_ = ws.WriteJSON(WSMessage{Action: "subscribe", Symbols: []string{"MOCK"}})
	time.Sleep(30 * time.Millisecond)

	// 5. Test broadcastQuotes when client.Send is full (hits default in select)
	hub.mu.RLock()
	var testClient *Client
	for c := range hub.clients {
		testClient = c
		break
	}
	hub.mu.RUnlock()

	if testClient != nil {
		// Fill Send channel completely
		for i := 0; i < cap(testClient.Send); i++ {
			select {
			case testClient.Send <- []byte("fill"):
			default:
			}
		}

		// Now broadcastQuotes hits default: slog.Warn
		hub.broadcastQuotes(ctx)

		// 6. Test limit exceeded when Send is full (hits default in limitExceeded select)
		symbols := make([]string, 55)
		for i := 0; i < 55; i++ {
			symbols[i] = fmt.Sprintf("SYM%d", i)
		}
		_ = ws.WriteJSON(WSMessage{Action: "subscribe", Symbols: symbols})
		time.Sleep(30 * time.Millisecond)
	}

	// 7. Send normal close to exercise IsUnexpectedCloseError in ReadPump
	_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "normal-close"))
	time.Sleep(30 * time.Millisecond)
}

func TestClient_WritePump_BatchedMessages(t *testing.T) {
	ms := new(MockMarketService)
	hub := NewHub(ms)
	handler := NewHandler(hub)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "batch_user"))
		handler.ServeWS(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"http://localhost:3000"}})
	assert.NoError(t, err)
	defer ws.Close()

	time.Sleep(20 * time.Millisecond)
	hub.mu.RLock()
	var testClient *Client
	for c := range hub.clients {
		testClient = c
		break
	}
	hub.mu.RUnlock()

	if testClient != nil {
		testClient.Send <- []byte("msg1")
		testClient.Send <- []byte("msg2")
		testClient.Send <- []byte("msg3")
	}

	time.Sleep(50 * time.Millisecond)
}

func TestHub_BroadcastQuotes_ChannelFull(t *testing.T) {
	ms := new(MockMarketService)
	ms.On("GetQuote", mock.Anything, "FULL").Return(&market.Quote{Symbol: "FULL", Price: 100}, nil)
	hub := NewHub(ms)
	client := &Client{
		Hub:        hub,
		Send:       make(chan []byte, 1),
		UserID:     "full_user",
		subscribed: map[string]bool{"FULL": true},
	}
	hub.clients[client] = true
	client.Send <- []byte("already full")

	hub.broadcastQuotes(context.Background())
}

func TestHub_ZeroDefaults(t *testing.T) {
	hub := NewHub(nil)
	hub.pingPeriod = 0
	assert.Equal(t, 54*time.Second, hub.GetPingPeriod())

	hub.broadcastInterval = 0
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	hub.Start(ctx)

	client := NewClient(hub, nil, "u").WithPingPeriod(0)
	assert.Equal(t, time.Duration(0), client.pingPeriod)
}

func TestHub_BroadcastQuotes_MarshalError(t *testing.T) {
	ms := new(MockMarketService)
	ms.On("GetQuote", mock.Anything, "NAN").Return(&market.Quote{Symbol: "NAN", Price: math.NaN()}, nil)
	hub := NewHub(ms)
	client := &Client{
		Hub:        hub,
		Send:       make(chan []byte, 10),
		UserID:     "nan_user",
		subscribed: map[string]bool{"NAN": true},
	}
	hub.clients[client] = true
	hub.broadcastQuotes(context.Background())
}

func TestClient_WritePump_ZeroPeriod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{
			Conn: conn,
			Send: make(chan []byte),
		}
		close(client.Send)
		client.WritePump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		_ = ws.Close()
	}
}

func TestClient_ReadPump_LimitExceeded_FullSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub := NewHub(nil)
		client := NewClient(hub, conn, "limit_user")
		// Fill Send channel completely
		for i := 0; i < cap(client.Send); i++ {
			client.Send <- []byte("fill")
		}
		client.ReadPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)

	symbols := make([]string, 55)
	for i := 0; i < 55; i++ {
		symbols[i] = fmt.Sprintf("SYM%d", i)
	}
	_ = ws.WriteJSON(WSMessage{Action: "subscribe", Symbols: symbols})
	time.Sleep(30 * time.Millisecond)
	_ = ws.Close()
}

func TestClient_WritePump_PingWriteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{
			Conn:       conn,
			Send:       make(chan []byte, 10),
			pingPeriod: 10 * time.Millisecond,
		}
		// Wait for remote client to close socket
		time.Sleep(30 * time.Millisecond)
		client.WritePump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	_ = ws.Close()
	time.Sleep(60 * time.Millisecond)
}

func TestClient_WritePump_NextWriterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{
			Conn: conn,
			Send: make(chan []byte, 1),
		}
		// Close underlying conn first
		_ = conn.Close()
		client.Send <- []byte("after close")
		client.WritePump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		_ = ws.Close()
	}
}
