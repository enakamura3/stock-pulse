package history

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTransactionSource struct {
	mock.Mock
}

func (m *MockTransactionSource) GetUnifiedTransactions(ctx context.Context, portfolioID, userID string) ([]UnifiedTransaction, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]UnifiedTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestNewService(t *testing.T) {
	s := NewService()
	assert.NotNil(t, s)
}

func TestService_GetPortfolioHistory(t *testing.T) {
	t.Run("success multiple sources and sorting", func(t *testing.T) {
		mockSource1 := new(MockTransactionSource)
		mockSource2 := new(MockTransactionSource)

		tx1 := UnifiedTransaction{ID: "1", Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}
		tx2 := UnifiedTransaction{ID: "2", Date: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)} // Newest
		tx3 := UnifiedTransaction{ID: "3", Date: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)}

		mockSource1.On("GetUnifiedTransactions", mock.Anything, "port1", "user1").Return([]UnifiedTransaction{tx1}, nil)
		mockSource2.On("GetUnifiedTransactions", mock.Anything, "port1", "user1").Return([]UnifiedTransaction{tx2, tx3}, nil)

		svc := NewService(mockSource1, mockSource2)
		res, err := svc.GetPortfolioHistory(context.Background(), "port1", "user1")

		assert.NoError(t, err)
		assert.Len(t, res, 3)
		assert.Equal(t, "2", res[0].ID) // Newest first
		assert.Equal(t, "3", res[1].ID)
		assert.Equal(t, "1", res[2].ID)
	})

	t.Run("error from source", func(t *testing.T) {
		mockSource1 := new(MockTransactionSource)
		mockSource1.On("GetUnifiedTransactions", mock.Anything, "port1", "user1").Return(nil, errors.New("db error"))

		svc := NewService(mockSource1)
		res, err := svc.GetPortfolioHistory(context.Background(), "port1", "user1")

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, "db error", err.Error())
	})
}
