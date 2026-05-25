package alert

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onigiri/stockpulse/backend/internal/market"
	"github.com/stretchr/testify/mock"
)

type MockMailService struct {
	mock.Mock
}

func (m *MockMailService) SendAlertEmail(toEmail, toName, ticker, assetName string, currentVal, targetVal float64, condition, currency string) error {
	args := m.Called(toEmail, toName, ticker, assetName, currentVal, targetVal, condition, currency)
	return args.Error(0)
}

func TestAlertWorker_StartAndStop(t *testing.T) {
	repo := new(MockAlertRepo)
	ms := new(MockMarketService)
	mail := new(MockMailService)

	w := NewAlertWorker(repo, ms, mail)
	w.interval = 10 * time.Millisecond

	// Setup expectations that might occur during the short run
	repo.On("GetActiveAlerts", mock.Anything).Return(([]*Alert)(nil), nil).Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	go w.Start(ctx)
	time.Sleep(25 * time.Millisecond)
	cancel()
}

func TestAlertWorker_process(t *testing.T) {
	t.Run("DB Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		repo.On("GetActiveAlerts", mock.Anything).Return(([]*Alert)(nil), errors.New("db err"))

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Empty Alerts", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		repo.On("GetActiveAlerts", mock.Anything).Return([]*Alert{}, nil)

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Market Provider Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return((*market.Quote)(nil), errors.New("api err"))

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())
		repo.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("Not Triggered", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 140.0}, nil)

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Triggered ABOVE", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE", UserEmail: "u@u.com"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 155.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(nil)
		mail.On("SendAlertEmail", "u@u.com", mock.Anything, "AAPL", mock.Anything, 155.0, 150.0, "ABOVE", mock.Anything).Return(nil)

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())

		// give time for async email sending
		time.Sleep(10 * time.Millisecond)

		repo.AssertExpectations(t)
		mail.AssertExpectations(t)
	})

	t.Run("Triggered BELOW", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "BELOW", UserEmail: "u@u.com"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 145.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(nil)
		mail.On("SendAlertEmail", "u@u.com", mock.Anything, "AAPL", mock.Anything, 145.0, 150.0, "BELOW", mock.Anything).Return(nil)

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())
		time.Sleep(10 * time.Millisecond)

		repo.AssertExpectations(t)
	})

	t.Run("Mark Triggered DB Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		mail := new(MockMailService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE", UserEmail: "u@u.com"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 155.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(errors.New("already triggered"))

		w := NewAlertWorker(repo, ms, mail)
		w.checkActiveAlerts(context.Background())

		repo.AssertExpectations(t)
		mail.AssertNotCalled(t, "SendAlertEmail")
	})
}
