package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, error)
	LinkAccountWithToken(ctx context.Context, token string, chatID int64) error
	GetUserIDByChatID(ctx context.Context, chatID int64) (uuid.UUID, error)
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
