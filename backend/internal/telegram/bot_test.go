package telegram

import (
	"testing"

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
}

func TestBotRunner_LifecycleAndUsername(t *testing.T) {
	t.Run("NewBotRunner empty token", func(t *testing.T) {
		runner, err := NewBotRunner("", nil)
		assert.NoError(t, err)
		assert.Nil(t, runner)
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


