package telegram

import (
	"context"
	"math"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) LinkAccount(ctx context.Context, userID uuid.UUID, chatID int64) error {
	return m.Called(ctx, userID, chatID).Error(0)
}
func (m *MockRepository) GetUserIDByChatID(ctx context.Context, chatID int64) (uuid.UUID, error) {
	args := m.Called(ctx, chatID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
func (m *MockRepository) GetChatIDByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockRepository) UnlinkAccount(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func setupServiceTest(t *testing.T) (*service, *MockRepository, *miniredis.Miniredis, *redis.Client) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	repo := new(MockRepository)
	svc := NewService(repo, rdb).(*service)
	return svc, repo, mr, rdb
}

func TestService_GenerateLinkToken(t *testing.T) {
	svc, _, mr, _ := setupServiceTest(t)

	uID := uuid.New()
	token, err := svc.GenerateLinkToken(context.Background(), uID)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	val, err := mr.Get("telegram_link:" + token)
	assert.NoError(t, err)
	assert.Equal(t, uID.String(), val)
}

func TestService_LinkAccountWithToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, mr, _ := setupServiceTest(t)
		uID := uuid.New()
		mr.Set("telegram_link:mytoken", uID.String())

		repo.On("LinkAccount", mock.Anything, uID, int64(12345)).Return(nil)

		err := svc.LinkAccountWithToken(context.Background(), "mytoken", 12345)
		assert.NoError(t, err)

		// should be deleted
		exists := mr.Exists("telegram_link:mytoken")
		assert.False(t, exists)
	})

	t.Run("invalid token", func(t *testing.T) {
		svc, _, _, _ := setupServiceTest(t)
		err := svc.LinkAccountWithToken(context.Background(), "invalid", 12345)
		assert.ErrorContains(t, err, "token inválido")
	})

	t.Run("invalid user id in redis", func(t *testing.T) {
		svc, _, mr, _ := setupServiceTest(t)
		mr.Set("telegram_link:mytoken2", "not-a-uuid")
		err := svc.LinkAccountWithToken(context.Background(), "mytoken2", 12345)
		assert.ErrorContains(t, err, "invalid user id")
	})

	t.Run("db error", func(t *testing.T) {
		svc, repo, mr, _ := setupServiceTest(t)
		uID := uuid.New()
		mr.Set("telegram_link:mytoken3", uID.String())

		repo.On("LinkAccount", mock.Anything, uID, int64(12345)).Return(assert.AnError)

		err := svc.LinkAccountWithToken(context.Background(), "mytoken3", 12345)
		assert.ErrorContains(t, err, "failed to link account in db")
	})
}

func TestService_GetUserIDByChatID(t *testing.T) {
	svc, repo, _, _ := setupServiceTest(t)
	uID := uuid.New()
	repo.On("GetUserIDByChatID", mock.Anything, int64(123)).Return(uID, nil)

	res, err := svc.GetUserIDByChatID(context.Background(), 123)
	assert.NoError(t, err)
	assert.Equal(t, uID, res)
}

func TestService_ConversationState(t *testing.T) {
	svc, _, _, _ := setupServiceTest(t)

	// Get empty
	state, err := svc.GetConversationState(context.Background(), 111)
	assert.NoError(t, err)
	assert.Nil(t, state)

	// Set state
	newState := ConversationState{Step: "buy", Ticker: "AAPL"}
	err = svc.SetConversationState(context.Background(), 111, newState)
	assert.NoError(t, err)

	// Get state
	state, err = svc.GetConversationState(context.Background(), 111)
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, "buy", state.Step)
	assert.Equal(t, "AAPL", state.Ticker)

	// Clear state
	err = svc.ClearConversationState(context.Background(), 111)
	assert.NoError(t, err)

	// Get empty again
	state, err = svc.GetConversationState(context.Background(), 111)
	assert.NoError(t, err)
	assert.Nil(t, state)
}

func TestService_ActivePortfolio(t *testing.T) {
	svc, _, _, _ := setupServiceTest(t)

	// Get empty
	p, err := svc.GetActivePortfolio(context.Background(), 222)
	assert.NoError(t, err)
	assert.Empty(t, p)

	// Set portfolio
	err = svc.SetActivePortfolio(context.Background(), 222, "port1")
	assert.NoError(t, err)

	// Get portfolio
	p, err = svc.GetActivePortfolio(context.Background(), 222)
	assert.NoError(t, err)
	assert.Equal(t, "port1", p)
}

func TestService_RedisErrors(t *testing.T) {
	svc, _, mr, rdb := setupServiceTest(t)

	// Corrupt data for conversation state
	mr.Set("telegram_state:333", "{invalid-json}")
	_, err := svc.GetConversationState(context.Background(), 333)
	assert.ErrorContains(t, err, "failed to unmarshal state")

	// Close redis to trigger network errors
	rdb.Close()

	uID := uuid.New()
	_, err = svc.GenerateLinkToken(context.Background(), uID)
	assert.ErrorContains(t, err, "failed to save token")

	err = svc.SetConversationState(context.Background(), 111, ConversationState{})
	assert.ErrorContains(t, err, "failed to save state")

	_, err = svc.GetConversationState(context.Background(), 111)
	assert.ErrorContains(t, err, "failed to get state")

	err = svc.SetActivePortfolio(context.Background(), 111, "p1")
	assert.ErrorContains(t, err, "failed to save active portfolio")

	_, err = svc.GetActivePortfolio(context.Background(), 111)
	assert.ErrorContains(t, err, "failed to get active portfolio")

	err = svc.LinkAccountWithToken(context.Background(), "token", 111)
	assert.ErrorContains(t, err, "failed to get token")
}

func TestService_EdgeCases(t *testing.T) {
	svc, _, _, _ := setupServiceTest(t)

	// Test SetConversationState marshal error
	state := ConversationState{
		Quantity: math.NaN(), // Causes json.Marshal to fail
	}
	err := svc.SetConversationState(context.Background(), 111, state)
	assert.ErrorContains(t, err, "failed to marshal conversation state")
}
