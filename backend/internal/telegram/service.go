package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ConversationState struct {
	Step        string  `json:"step"`
	Ticker      string  `json:"ticker,omitempty"`
	Type        string  `json:"type,omitempty"`
	Quantity    float64 `json:"quantity,omitempty"`
	PortfolioID string  `json:"portfolio_id,omitempty"`
}

type Service interface {
	GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, error)
	LinkAccountWithToken(ctx context.Context, token string, chatID int64) error
	GetUserIDByChatID(ctx context.Context, chatID int64) (uuid.UUID, error)

	SetConversationState(ctx context.Context, chatID int64, state ConversationState) error
	GetConversationState(ctx context.Context, chatID int64) (*ConversationState, error)
	ClearConversationState(ctx context.Context, chatID int64) error
}

type service struct {
	repo  Repository
	redis *redis.Client
}

func NewService(repo Repository, rdb *redis.Client) Service {
	return &service{repo: repo, redis: rdb}
}

func (s *service) GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	err := s.redis.Set(ctx, "telegram_link:"+token, userID.String(), 10*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to save token in redis: %w", err)
	}

	return token, nil
}

func (s *service) LinkAccountWithToken(ctx context.Context, token string, chatID int64) error {
	userIDStr, err := s.redis.Get(ctx, "telegram_link:"+token).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("token inválido ou expirado")
		}
		return fmt.Errorf("failed to get token from redis: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user id in redis: %w", err)
	}

	err = s.repo.LinkAccount(ctx, userID, chatID)
	if err != nil {
		return fmt.Errorf("failed to link account in db: %w", err)
	}

	// Invalida o token após uso com sucesso
	s.redis.Del(ctx, "telegram_link:"+token)

	return nil
}

func (s *service) GetUserIDByChatID(ctx context.Context, chatID int64) (uuid.UUID, error) {
	return s.repo.GetUserIDByChatID(ctx, chatID)
}

func (s *service) SetConversationState(ctx context.Context, chatID int64, state ConversationState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation state: %w", err)
	}
	key := fmt.Sprintf("telegram_state:%d", chatID)
	// Expira em 1 hora por segurança
	err = s.redis.Set(ctx, key, data, 1*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save state in redis: %w", err)
	}
	return nil
}

func (s *service) GetConversationState(ctx context.Context, chatID int64) (*ConversationState, error) {
	key := fmt.Sprintf("telegram_state:%d", chatID)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No state active
		}
		return nil, fmt.Errorf("failed to get state from redis: %w", err)
	}

	var state ConversationState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return &state, nil
}

func (s *service) ClearConversationState(ctx context.Context, chatID int64) error {
	key := fmt.Sprintf("telegram_state:%d", chatID)
	return s.redis.Del(ctx, key).Err()
}
