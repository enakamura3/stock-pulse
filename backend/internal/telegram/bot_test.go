package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBotRunner_SendAlertMessage(t *testing.T) {
	t.Run("Bot is nil", func(t *testing.T) {
		var runner *BotRunner

		err := runner.SendAlertMessage(123, "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.NoError(t, err)
	})

	t.Run("Bot object is nil", func(t *testing.T) {
		runner := &BotRunner{bot: nil}

		err := runner.SendAlertMessage(123, "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.NoError(t, err)
	})
}
