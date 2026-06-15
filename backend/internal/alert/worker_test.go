package alert

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/mock"
)

type MockTelegramService struct {
	mock.Mock
}

func (m *MockTelegramService) SendAlertMessage(chatID int64, userName, ticker, assetName string, currentVal, targetVal float64, condition, currency string) error {
	args := m.Called(chatID, userName, ticker, assetName, currentVal, targetVal, condition, currency)
	return args.Error(0)
}

func TestAlertWorker_StartAndStop(t *testing.T) {
	repo := new(MockAlertRepo)
	ms := new(MockMarketService)
	tg := new(MockTelegramService)

	w := NewAlertWorker(repo, ms, tg)
	w.interval = 10 * time.Millisecond

	// Setup expectations that might occur during the short run
	repo.On("GetActiveAlerts", mock.Anything).Return(([]*Alert)(nil), nil).Maybe()
	
	w.CheckActiveAlerts(context.Background())
}

func TestAlertWorker_process(t *testing.T) {
	t.Run("DB Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		repo.On("GetActiveAlerts", mock.Anything).Return(([]*Alert)(nil), errors.New("db err"))

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Empty Alerts", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		repo.On("GetActiveAlerts", mock.Anything).Return([]*Alert{}, nil)

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Market Provider Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return((*market.Quote)(nil), errors.New("api err"))

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		repo.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("Not Triggered", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE"},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 140.0}, nil)

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		repo.AssertExpectations(t)
	})

	t.Run("Triggered ABOVE", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		chatId := int64(123)
		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE", TelegramChatID: &chatId},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 155.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(nil)
		tg.On("SendAlertMessage", int64(123), mock.Anything, "AAPL", mock.Anything, 155.0, 150.0, "ABOVE", mock.Anything).Return(nil)

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())

		// give time for async email sending
		time.Sleep(10 * time.Millisecond)

		repo.AssertExpectations(t)
		tg.AssertExpectations(t)
	})

	t.Run("Triggered BELOW", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		chatId := int64(123)
		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "BELOW", TelegramChatID: &chatId},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 145.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(nil)
		tg.On("SendAlertMessage", int64(123), mock.Anything, "AAPL", mock.Anything, 145.0, 150.0, "BELOW", mock.Anything).Return(nil)

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		time.Sleep(10 * time.Millisecond)

		repo.AssertExpectations(t)
	})

	t.Run("Mark Triggered DB Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		chatId := int64(123)
		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE", TelegramChatID: &chatId},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 155.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(errors.New("already triggered"))

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())

		repo.AssertExpectations(t)
		tg.AssertNotCalled(t, "SendAlertMessage")
	})

	t.Run("No Telegram Link", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE", TelegramChatID: nil},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 155.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(nil)

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		time.Sleep(10 * time.Millisecond)

		repo.AssertExpectations(t)
		tg.AssertNotCalled(t, "SendAlertMessage")
	})

	t.Run("Telegram API Error", func(t *testing.T) {
		repo := new(MockAlertRepo)
		ms := new(MockMarketService)
		tg := new(MockTelegramService)

		chatId := int64(123)
		alerts := []*Alert{
			{ID: "1", Ticker: "AAPL", TargetPrice: 150.0, Condition: "ABOVE", TelegramChatID: &chatId},
		}
		repo.On("GetActiveAlerts", mock.Anything).Return(alerts, nil)
		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 155.0}, nil)
		repo.On("MarkAlertTriggered", mock.Anything, "1").Return(nil)
		tg.On("SendAlertMessage", int64(123), mock.Anything, "AAPL", mock.Anything, 155.0, 150.0, "ABOVE", mock.Anything).Return(errors.New("telegram error"))

		w := NewAlertWorker(repo, ms, tg)
		w.CheckActiveAlerts(context.Background())
		time.Sleep(10 * time.Millisecond)

		repo.AssertExpectations(t)
		tg.AssertExpectations(t)
	})
}
