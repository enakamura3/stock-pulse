package telegram

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/telebot.v3"
)

func TestBotRunner_SendAlertMessage(t *testing.T) {
	t.Run("Bot is nil", func(t *testing.T) {
		var runner *BotRunner

		err := runner.SendAlertMessage(123, "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.NoError(t, err)
	})

	t.Run("Bot object is nil", func(t *testing.T) {
		runner := &BotRunner{bot: nil}

		err := runner.SendAlertMessage(123, "User", "AAPL", "Apple", 155.0, 150.0, "BELOW", "USD")
		assert.NoError(t, err)
	})

	t.Run("Mock server SendAlertMessage ABOVE and BELOW", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 1, "chat": {"id": 123}}}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		errAbove := runner.SendAlertMessage(123, "User_Name*", "AAPL_US", "Apple*Corp", 155.0, 150.0, "ABOVE", "USD")
		assert.NoError(t, errAbove)

		errBelow := runner.SendAlertMessage(123, "User", "PETR4", "Petrobras", 25.0, 30.0, "BELOW", "BRL")
		assert.NoError(t, errBelow)

		errMicro := runner.SendAlertMessage(123, "CryptoUser", "PEPE-USD", "Pepe Coin", 0.000034, 0.000030, "ABOVE", "USD")
		assert.NoError(t, errMicro)
	})

	t.Run("Mock server SendAlertMessage blocked by user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ok": false, "error_code": 403, "description": "Forbidden: bot was blocked by the user"}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		errBlocked := runner.SendAlertMessage(123, "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.Error(t, errBlocked)
		assert.True(t, isBlockedByUser(errBlocked))
	})
}

func TestBotRunner_SendDividendPaymentAlert(t *testing.T) {
	t.Run("Bot is nil", func(t *testing.T) {
		var runner *BotRunner
		err := runner.SendDividendPaymentAlert(123, "User", "Carteira", []portfolio.DividendNotificationItem{
			{Ticker: "PETR4", NetAmount: 100},
		})
		assert.NoError(t, err)
	})

	t.Run("Bot object is nil", func(t *testing.T) {
		runner := &BotRunner{bot: nil}
		err := runner.SendDividendPaymentAlert(123, "User", "Carteira", []portfolio.DividendNotificationItem{
			{Ticker: "PETR4", NetAmount: 100},
		})
		assert.NoError(t, err)
	})

	t.Run("Items is empty", func(t *testing.T) {
		runner := &BotRunner{bot: new(telebot.Bot)}
		err := runner.SendDividendPaymentAlert(123, "User", "Carteira", nil)
		assert.NoError(t, err)
	})

	t.Run("Mock server single item", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 1, "chat": {"id": 123}}}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		items := []portfolio.DividendNotificationItem{
			{
				Ticker:         "PETR4",
				AssetName:      "Petrobras",
				Type:           "DIVIDENDO",
				Quantity:       100.0,
				PerShareAmount: 1.50,
				NetAmount:      150.0,
				Currency:       "BRL",
			},
		}
		err = runner.SendDividendPaymentAlert(123, "User_Name*", "Minha_Carteira*", items)
		assert.NoError(t, err)
	})

	t.Run("Mock server multiple items with fractional quantities and USD", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 2, "chat": {"id": 123}}}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		items := []portfolio.DividendNotificationItem{
			{
				Ticker:         "AAPL",
				AssetName:      "Apple",
				Type:           "DIVIDENDO",
				Quantity:       10.5,
				PerShareAmount: 0.25,
				NetAmount:      2.625,
				Currency:       "USD",
			},
			{
				Ticker:         "VALE3",
				AssetName:      "Vale",
				Type:           "JCP",
				Quantity:       200.0,
				PerShareAmount: 2.10,
				NetAmount:      420.0,
				Currency:       "",
			},
		}
		err = runner.SendDividendPaymentAlert(123, "User", "Carteira Global", items)
		assert.NoError(t, err)
	})

	t.Run("Mock server blocked by user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ok": false, "error_code": 403, "description": "Forbidden: bot was blocked by the user"}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		items := []portfolio.DividendNotificationItem{
			{Ticker: "PETR4", NetAmount: 100},
		}
		errBlocked := runner.SendDividendPaymentAlert(123, "User", "Carteira", items)
		assert.Error(t, errBlocked)
		assert.True(t, isBlockedByUser(errBlocked))
	})
}

func TestBotRunner_SendDailyDigest(t *testing.T) {
	t.Run("Bot is nil", func(t *testing.T) {
		var runner *BotRunner
		err := runner.SendDailyDigest(123, "User", "Carteira", "BRL", 1000, 10, 1.0, 1.0, nil, nil, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Bot object is nil", func(t *testing.T) {
		runner := &BotRunner{bot: nil}
		err := runner.SendDailyDigest(123, "User", "Carteira", "BRL", 1000, 10, 1.0, 1.0, nil, nil, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Mock server success positive change and return with dividends and movers", func(t *testing.T) {
		var receivedBody string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, _ := io.ReadAll(r.Body)
			receivedBody = string(bodyBytes)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 10, "chat": {"id": 123}}}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		todayDivs := []portfolio.CalculatedDividend{
			{Ticker: "PETR4", Type: "DIVIDENDO", Currency: "R$", NetAmount: 150.0},
		}
		upcomingDivs := []portfolio.CalculatedDividend{
			{Ticker: "VALE3", Type: "JCP", Currency: "R$", NetAmount: 200.0, PaymentDate: time.Now().AddDate(0, 0, 3)},
		}
		topGainers := []DailyMover{
			{Ticker: "PETR4", DailyChangePercent: 3.5, Currency: "R$", OpenPrice: 34.0, CurrentPrice: 35.0},
		}
		topLosers := []DailyMover{
			{Ticker: "VALE3", DailyChangePercent: -2.1, Currency: "R$", OpenPrice: 61.5, CurrentPrice: 60.0},
		}

		err = runner.SendDailyDigest(123, "Test_User*", "Carteira_Acoes*", "R$", 50000.0, 350.50, 0.70, 5.25, topGainers, topLosers, todayDivs, upcomingDivs)
		assert.NoError(t, err)
		assert.Contains(t, receivedBody, "Abert.")
		assert.Contains(t, receivedBody, "Atual")
	})

	t.Run("Mock server success negative change and return without dividends", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 11, "chat": {"id": 123}}}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		err = runner.SendDailyDigest(123, "User", "Carteira", "USD", 25000.0, -120.0, -0.48, -2.15, nil, nil, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Mock server success neutral change and return", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 12, "chat": {"id": 123}}}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		err = runner.SendDailyDigest(123, "User", "Carteira", "BRL", 10000.0, 0.0, 0.0, 0.0, nil, nil, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Mock server blocked by user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ok": false, "error_code": 403, "description": "Forbidden: bot was blocked by the user"}`))
		}))
		defer server.Close()

		b, err := telebot.NewBot(telebot.Settings{
			URL:     server.URL,
			Token:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Offline: true,
		})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}
		err = runner.SendDailyDigest(123, "User", "Carteira", "BRL", 10000.0, 10.0, 0.10, 0.5, nil, nil, nil, nil)
		assert.Error(t, err)
		assert.True(t, isBlockedByUser(err))
	})
}

func TestBotRunner_LifecycleAndUsername(t *testing.T) {
	t.Run("NewBotRunner empty token", func(t *testing.T) {
		runner, err := NewBotRunner("", nil)
		assert.NoError(t, err)
		assert.Nil(t, runner)
	})

	t.Run("NewBotRunner invalid token returns error", func(t *testing.T) {
		h, _, _, _, _, _ := setupHandlersTest()
		runner, err := NewBotRunner("invalid_token_xyz", h)
		assert.Error(t, err)
		assert.Nil(t, runner)
	})

	t.Run("NewBotRunnerWithSettings success", func(t *testing.T) {
		h, _, _, _, _, _ := setupHandlersTest()
		runner, err := NewBotRunnerWithSettings(telebot.Settings{Offline: true}, h)
		assert.NoError(t, err)
		assert.NotNil(t, runner)
	})

	t.Run("Start and Stop on nil runner", func(t *testing.T) {
		var runner *BotRunner
		runner.Start()
		runner.Stop()
		assert.Empty(t, runner.GetUsername())
	})

	t.Run("Start and Stop on empty bot", func(t *testing.T) {
		runner := &BotRunner{}
		runner.Start()
		runner.Stop()
		assert.Empty(t, runner.GetUsername())
	})

	t.Run("GetUsername with user", func(t *testing.T) {
		b, err := telebot.NewBot(telebot.Settings{Offline: true})
		assert.NoError(t, err)
		b.Me = &telebot.User{Username: "test_bot"}

		runner := &BotRunner{bot: b}
		assert.Equal(t, "test_bot", runner.GetUsername())

		go runner.Start()
		time.Sleep(10 * time.Millisecond)
		runner.Stop()
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	mw := rateLimitMiddleware()

	t.Run("sender is nil", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Sender").Return((*telebot.User)(nil))

		called := false
		handler := mw(func(c telebot.Context) error {
			called = true
			return nil
		})

		err := handler(mCtx)
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("sender allowed and rate limited", func(t *testing.T) {
		sender := &telebot.User{ID: 999, Username: "testuser"}
		handler := mw(func(c telebot.Context) error {
			return nil
		})

		// First 3 calls should succeed (burst = 3)
		for i := 0; i < 3; i++ {
			mCtx := new(MockTelebotContext)
			mCtx.On("Sender").Return(sender)
			err := handler(mCtx)
			assert.NoError(t, err)
		}

		// 4th call should hit rate limit
		mCtxLimit := new(MockTelebotContext)
		mCtxLimit.On("Sender").Return(sender)
		mCtxLimit.On("Send", mock.Anything, mock.Anything).Return(nil)
		err := handler(mCtxLimit)
		assert.NoError(t, err)
	})
}

func TestFormatDailyMover(t *testing.T) {
	p := message.NewPrinter(language.Portuguese)

	t.Run("Gainer with open and current price", func(t *testing.T) {
		m := DailyMover{
			Ticker:             "PETR4",
			DailyChangePercent: 2.35,
			Currency:           "R$",
			OpenPrice:          38.10,
			CurrentPrice:       38.95,
		}
		res := formatDailyMover(p, m, "BRL", true)
		assert.Contains(t, res, "• *PETR4*: +2,35%")
		assert.Contains(t, res, "Abert. R$ 38,10 → Atual R$ 38,95")
	})

	t.Run("Loser with open and current price", func(t *testing.T) {
		m := DailyMover{
			Ticker:             "VALE3",
			DailyChangePercent: -1.80,
			Currency:           "R$",
			OpenPrice:          60.00,
			CurrentPrice:       58.92,
		}
		res := formatDailyMover(p, m, "BRL", false)
		assert.Contains(t, res, "• *VALE3*: -1,80%")
		assert.Contains(t, res, "Abert. R$ 60,00 → Atual R$ 58,92")
	})

	t.Run("Current price only without open price", func(t *testing.T) {
		m := DailyMover{
			Ticker:             "AAPL",
			DailyChangePercent: 1.50,
			Currency:           "USD",
			CurrentPrice:       150.00,
		}
		res := formatDailyMover(p, m, "USD", true)
		assert.Contains(t, res, "• *AAPL*: +1,50%")
		assert.Contains(t, res, "Atual USD 150,00")
		assert.NotContains(t, res, "Abert.")
	})

	t.Run("No prices available", func(t *testing.T) {
		m := DailyMover{
			Ticker:             "MOCK",
			DailyChangePercent: -0.50,
		}
		res := formatDailyMover(p, m, "BRL", false)
		assert.Equal(t, "  • *MOCK*: -0,50%\n", res)
	})

	t.Run("Fallback to default currency and nil printer", func(t *testing.T) {
		m := DailyMover{
			Ticker:             "BBDC4",
			DailyChangePercent: 3.10,
			OpenPrice:          14.00,
			CurrentPrice:       14.43,
		}
		res := formatDailyMover(nil, m, "BRL", true)
		assert.Contains(t, res, "• *BBDC4*: +3,10%")
		assert.Contains(t, res, "Abert. BRL 14,00 → Atual BRL 14,43")
	})
}


