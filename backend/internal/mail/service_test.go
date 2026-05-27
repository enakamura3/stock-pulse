package mail

import (
	"errors"
	"net/smtp"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	os.Setenv("SMTP_HOST", "test_host")
	os.Setenv("SMTP_PORT", "test_port")
	os.Setenv("SMTP_FROM", "test_from")

	svc := NewService()

	assert.Equal(t, "test_host", svc.host)
	assert.Equal(t, "test_port", svc.port)
	assert.Equal(t, "test_from", svc.from)

	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_FROM")

	svcDefault := NewService()

	assert.Equal(t, "localhost", svcDefault.host)
	assert.Equal(t, "1025", svcDefault.port)
	assert.Equal(t, "no-reply@stock-pulse.com", svcDefault.from)
}

func TestService_SendAlertEmail(t *testing.T) {
	originalSendMail := SendMailFunc
	defer func() { SendMailFunc = originalSendMail }()

	t.Run("Success ABOVE", func(t *testing.T) {
		SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			assert.Equal(t, "localhost:1025", addr)
			assert.Equal(t, "no-reply@stock-pulse.com", from)
			assert.Equal(t, []string{"user@test.com"}, to)
			assert.Contains(t, string(msg), "atingiu o preço alvo")
			assert.Contains(t, string(msg), "subiu para")
			return nil
		}

		svc := NewService()
		err := svc.SendAlertEmail("user@test.com", "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.NoError(t, err)
	})

	t.Run("Success BELOW", func(t *testing.T) {
		SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			assert.Contains(t, string(msg), "caiu para")
			return nil
		}

		svc := NewService()
		err := svc.SendAlertEmail("user@test.com", "User", "AAPL", "Apple", 145.0, 150.0, "BELOW", "USD")
		assert.NoError(t, err)
	})

	t.Run("SendMail Error", func(t *testing.T) {
		SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			return errors.New("smtp error")
		}

		svc := NewService()
		err := svc.SendAlertEmail("user@test.com", "User", "AAPL", "Apple", 155.0, 150.0, "ABOVE", "USD")
		assert.ErrorContains(t, err, "smtp error")
	})
}
