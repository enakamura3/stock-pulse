package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}
func (m *MockService) LinkAccountWithToken(ctx context.Context, token string, chatID int64) error {
	return m.Called(ctx, token, chatID).Error(0)
}
func (m *MockService) GetUserIDByChatID(ctx context.Context, chatID int64) (uuid.UUID, error) {
	args := m.Called(ctx, chatID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
func (m *MockService) SetConversationState(ctx context.Context, chatID int64, state ConversationState) error {
	return m.Called(ctx, chatID, state).Error(0)
}
func (m *MockService) GetConversationState(ctx context.Context, chatID int64) (*ConversationState, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) != nil {
		return args.Get(0).(*ConversationState), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockService) ClearConversationState(ctx context.Context, chatID int64) error {
	return m.Called(ctx, chatID).Error(0)
}
func (m *MockService) SetActivePortfolio(ctx context.Context, chatID int64, portfolioID string) error {
	return m.Called(ctx, chatID, portfolioID).Error(0)
}
func (m *MockService) GetActivePortfolio(ctx context.Context, chatID int64) (string, error) {
	args := m.Called(ctx, chatID)
	return args.String(0), args.Error(1)
}

func TestHTTPHandler_GenerateLinkToken(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		svc := new(MockService)
		h := NewHTTPHandler(svc, "testbot")

		req := httptest.NewRequest("POST", "/link", nil)
		rec := httptest.NewRecorder()

		h.GenerateLinkToken(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid user id", func(t *testing.T) {
		svc := new(MockService)
		h := NewHTTPHandler(svc, "testbot")

		req := httptest.NewRequest("POST", "/link", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "invalid-uuid"))
		rec := httptest.NewRecorder()

		h.GenerateLinkToken(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("failed to generate token", func(t *testing.T) {
		svc := new(MockService)
		h := NewHTTPHandler(svc, "testbot")
		uID := uuid.New()

		svc.On("GenerateLinkToken", mock.Anything, uID).Return("", errors.New("error generating"))

		req := httptest.NewRequest("POST", "/link", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, uID.String()))
		rec := httptest.NewRecorder()

		h.GenerateLinkToken(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := new(MockService)
		h := NewHTTPHandler(svc, "testbot")
		uID := uuid.New()

		svc.On("GenerateLinkToken", mock.Anything, uID).Return("mytoken123", nil)

		req := httptest.NewRequest("POST", "/link", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, uID.String()))
		rec := httptest.NewRecorder()

		h.GenerateLinkToken(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]string
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "mytoken123", resp["token"])
		assert.Equal(t, "testbot", resp["bot_username"])
	})
}
