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
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
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
}

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]market.DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}
