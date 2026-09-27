package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/onigiri/stock-pulse/backend/internal/database"
)

type TelegramUserLink struct {
	UserID         uuid.UUID `json:"user_id"`
	UserName       string    `json:"user_name"`
	TelegramChatID int64     `json:"telegram_chat_id"`
}

type Repository interface {
	LinkAccount(ctx context.Context, userID uuid.UUID, telegramChatID int64) error
	GetUserIDByChatID(ctx context.Context, telegramChatID int64) (uuid.UUID, error)
	GetChatIDByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetLinkedUsers(ctx context.Context) ([]TelegramUserLink, error)
	UnlinkAccount(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db database.DBTX
}

func NewRepository(db database.DBTX) Repository {
	return &repository{db: db}
}

func (r *repository) LinkAccount(ctx context.Context, userID uuid.UUID, telegramChatID int64) error {
	query := `
		INSERT INTO user_telegram_link (user_id, telegram_chat_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET telegram_chat_id = EXCLUDED.telegram_chat_id;
	`
	_, err := database.GetDB(ctx, r.db).Exec(ctx, query, userID, telegramChatID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to link account: %w", err)
	}
	return nil
}

func (r *repository) GetUserIDByChatID(ctx context.Context, telegramChatID int64) (uuid.UUID, error) {
	var userID uuid.UUID
	query := `SELECT user_id FROM user_telegram_link WHERE telegram_chat_id = $1;`
	err := database.GetDB(ctx, r.db).QueryRow(ctx, query, telegramChatID).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, fmt.Errorf("account not linked")
		}
		return uuid.Nil, fmt.Errorf("failed to query user id: %w", err)
	}
	return userID, nil
}

func (r *repository) GetChatIDByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var chatID int64
	query := `SELECT telegram_chat_id FROM user_telegram_link WHERE user_id = $1;`
	err := database.GetDB(ctx, r.db).QueryRow(ctx, query, userID).Scan(&chatID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("account not linked")
		}
		return 0, fmt.Errorf("failed to query chat id: %w", err)
	}
	return chatID, nil
}

func (r *repository) UnlinkAccount(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_telegram_link WHERE user_id = $1;`
	_, err := database.GetDB(ctx, r.db).Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unlink account: %w", err)
	}
	return nil
}

func (r *repository) GetLinkedUsers(ctx context.Context) ([]TelegramUserLink, error) {
	query := `
		SELECT utl.user_id, u.name, utl.telegram_chat_id
		FROM user_telegram_link utl
		INNER JOIN "user" u ON utl.user_id = u.id
		ORDER BY utl.created_at ASC;
	`
	rows, err := database.GetDB(ctx, r.db).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query linked users: %w", err)
	}
	defer rows.Close()

	var list []TelegramUserLink
	for rows.Next() {
		var item TelegramUserLink
		if err := rows.Scan(&item.UserID, &item.UserName, &item.TelegramChatID); err != nil {
			return nil, fmt.Errorf("failed to scan linked user: %w", err)
		}
		list = append(list, item)
	}
	return list, nil
}
