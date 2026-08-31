package telegram

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		errAbove := runner.SendAlertMessage(123, "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.NoError(t, errAbove)

		errBelow := runner.SendAlertMessage(123, "User", "PETR4", "Petrobras", 25.0, 30.0, "BELOW", "BRL")
		assert.NoError(t, errBelow)
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


