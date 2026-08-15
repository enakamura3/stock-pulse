package fixedincome

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAnbimaClient struct {
	mock.Mock
}

func (m *mockAnbimaClient) FetchHolidays(ctx context.Context, year int) ([]brasilAPIHoliday, error) {
	args := m.Called(ctx, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]brasilAPIHoliday), args.Error(1)
}

func TestWorker_SyncRates(t *testing.T) {
	mockRepo := &MockFullRepo{}
	registry := NewIndexRegistry()
	provider := &mockIndexProvider{rates: []IndexRate{{Indexer: "CDI", Rate: 0.05, Date: time.Now().AddDate(-1, 0, 0)}}}
	registry.Register(IndexerConfig{Name: "CDI", PrimaryProvider: provider})

	worker := NewWorker(mockRepo, registry)
	ctx := context.Background()

	// 1. Latest rate exists and is today -> skip
	today := time.Now()
	mockRepo.On("GetLatestIndexRate", mock.Anything, mock.Anything).Return(&IndexRate{Date: today}, nil).Maybe()

	worker.SyncRates(ctx)

	// 2. Latest rate fails/nil -> triggers fetch
	mockRepo2 := &MockFullRepo{}
	worker2 := NewWorker(mockRepo2, registry)
	mockRepo2.On("GetLatestIndexRate", mock.Anything, mock.Anything).Return(nil, errors.New("not found")).Maybe()
	mockRepo2.On("SaveIndexRates", mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3. SaveIndexRates error
	mockRepo3 := &MockFullRepo{}
	worker3 := NewWorker(mockRepo3, registry)
	mockRepo3.On("GetLatestIndexRate", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	mockRepo3.On("SaveIndexRates", mock.Anything, mock.Anything).Return(errors.New("db save error")).Maybe()
	worker3.SyncRates(ctx)

	// 4. Cancelled context exits immediately
	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()
	worker2.SyncRates(ctxCancelled)

	// 5. Context cancelled during iteration
	ctxCancelMid, cancelMid := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelMid()
	worker2.SyncRates(ctxCancelMid)
}

func TestAnbimaHolidayWorker_SyncHolidays(t *testing.T) {
	mockRepo := &MockFullRepo{}
	mockClient := &mockAnbimaClient{}

	w := NewAnbimaHolidayWorker(mockRepo, mockClient)
	ctx := context.Background()

	// 1. GetSeededHolidayYears error
	mockRepo.On("GetSeededHolidayYears", ctx).Return(nil, errors.New("db error")).Once()
	w.SyncHolidays(ctx)

	// 2. Already seeded all years
	currYear := time.Now().Year()
	years := make([]int, 0)
	for y := w.startYear; y <= currYear+1; y++ {
		years = append(years, y)
	}
	mockRepo.On("GetSeededHolidayYears", ctx).Return(years, nil).Once()
	w.SyncHolidays(ctx)

	// 3. Needs sync for missing year
	mockRepo.On("GetSeededHolidayYears", ctx).Return([]int{}, nil).Once()
	mockClient.On("FetchHolidays", ctx, mock.Anything).Return([]brasilAPIHoliday{
		{Date: "2026-01-01", Name: "Ano Novo", Type: "national"},
	}, nil).Maybe()
	mockRepo.On("SaveAnbimaHolidays", ctx, mock.Anything).Return(nil).Maybe()

	w.SyncHolidays(ctx)

	assert.NotNil(t, w)
}
