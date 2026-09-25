package telegram

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/time/rate"
	"gopkg.in/telebot.v3"
)

func TestRateLimitMiddleware_Unit(t *testing.T) {
	t.Run("default constructor RateLimitMiddleware", func(t *testing.T) {
		mw := RateLimitMiddleware()
		assert.NotNil(t, mw)
	})

	t.Run("sender is nil calls next", func(t *testing.T) {
		mw := RateLimitMiddleware()
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

	t.Run("message rate limited when limit exceeded", func(t *testing.T) {
		mw := NewRateLimitMiddleware(rate.Every(time.Second), 2, 10*time.Minute)
		sender := &telebot.User{ID: 1234, Username: "spammer"}

		handler := mw(func(c telebot.Context) error {
			return nil
		})

		// 2 requests allowed
		for i := 0; i < 2; i++ {
			mCtx := new(MockTelebotContext)
			mCtx.On("Sender").Return(sender)
			err := handler(mCtx)
			assert.NoError(t, err)
		}

		// 3rd request rate limited (text message context)
		mCtxLimit := new(MockTelebotContext)
		mCtxLimit.On("Sender").Return(sender)
		mCtxLimit.On("Callback").Return((*telebot.Callback)(nil))
		mCtxLimit.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := handler(mCtxLimit)
		assert.NoError(t, err)
		mCtxLimit.AssertCalled(t, "Send", mock.Anything, mock.Anything)
	})

	t.Run("callback rate limited returns alert response", func(t *testing.T) {
		mw := NewRateLimitMiddleware(rate.Every(time.Second), 1, 10*time.Minute)
		sender := &telebot.User{ID: 5678, Username: "button_spammer"}

		handler := mw(func(c telebot.Context) error {
			return nil
		})

		// 1st request allowed
		mCtx1 := new(MockTelebotContext)
		mCtx1.On("Sender").Return(sender)
		err := handler(mCtx1)
		assert.NoError(t, err)

		// 2nd request rate limited on callback query
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Sender").Return(sender)
		mCtx2.On("Callback").Return(&telebot.Callback{})
		mCtx2.On("Respond", mock.MatchedBy(func(resps []*telebot.CallbackResponse) bool {
			return len(resps) > 0 && resps[0] != nil && resps[0].ShowAlert && resps[0].Text != ""
		})).Return(nil)

		err = handler(mCtx2)
		assert.NoError(t, err)
		mCtx2.AssertCalled(t, "Respond", mock.Anything)
	})

	t.Run("cleanup expired limiters when size exceeds 100", func(t *testing.T) {
		// TTL of 2 milliseconds
		mw := NewRateLimitMiddleware(rate.Every(time.Second), 5, 2*time.Millisecond)

		handler := mw(func(c telebot.Context) error {
			return nil
		})

		// Populate 101 users
		for i := int64(1); i <= 101; i++ {
			mCtx := new(MockTelebotContext)
			mCtx.On("Sender").Return(&telebot.User{ID: i, Username: "user"})
			err := handler(mCtx)
			assert.NoError(t, err)
		}

		// Wait for TTL to expire for all existing entries
		time.Sleep(10 * time.Millisecond)

		// Request for user 102 triggers the cleanup loop (len(limiters) > 100)
		mCtx102 := new(MockTelebotContext)
		mCtx102.On("Sender").Return(&telebot.User{ID: 102, Username: "user102"})
		err := handler(mCtx102)
		assert.NoError(t, err)
	})
}
