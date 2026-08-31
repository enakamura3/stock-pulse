package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func setupRepoTest(t *testing.T) (pgxmock.PgxPoolIface, Repository) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("erro ao criar mock: %v", err)
	}
	return mock, NewRepository(mock)
}

func TestRepository_LinkAccount(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	uID := uuid.New()
	chatID := int64(123456)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO user_telegram_link`).
			WithArgs(uID, chatID, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.LinkAccount(context.Background(), uID, chatID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO user_telegram_link`).
			WithArgs(uID, chatID, pgxmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		err := repo.LinkAccount(context.Background(), uID, chatID)
		assert.ErrorContains(t, err, "failed to link account")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetUserIDByChatID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	uID := uuid.New()
	chatID := int64(123456)

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"user_id"}).AddRow(uID)
		mock.ExpectQuery(`SELECT user_id FROM user_telegram_link WHERE telegram_chat_id = \$1`).
			WithArgs(chatID).
			WillReturnRows(rows)

		res, err := repo.GetUserIDByChatID(context.Background(), chatID)
		assert.NoError(t, err)
		assert.Equal(t, uID, res)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Not Found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT user_id FROM user_telegram_link WHERE telegram_chat_id = \$1`).
			WithArgs(chatID).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetUserIDByChatID(context.Background(), chatID)
		assert.ErrorContains(t, err, "account not linked")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DB Error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT user_id FROM user_telegram_link WHERE telegram_chat_id = \$1`).
			WithArgs(chatID).
			WillReturnError(errors.New("db fail"))

		_, err := repo.GetUserIDByChatID(context.Background(), chatID)
		assert.ErrorContains(t, err, "failed to query user id")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetChatIDByUserID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	uID := uuid.New()
	chatID := int64(123456)

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"telegram_chat_id"}).AddRow(chatID)
		mock.ExpectQuery(`SELECT telegram_chat_id FROM user_telegram_link WHERE user_id = \$1`).
			WithArgs(uID).
			WillReturnRows(rows)

		res, err := repo.GetChatIDByUserID(context.Background(), uID)
		assert.NoError(t, err)
		assert.Equal(t, chatID, res)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Not Found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT telegram_chat_id FROM user_telegram_link WHERE user_id = \$1`).
			WithArgs(uID).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetChatIDByUserID(context.Background(), uID)
		assert.ErrorContains(t, err, "account not linked")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DB Error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT telegram_chat_id FROM user_telegram_link WHERE user_id = \$1`).
			WithArgs(uID).
			WillReturnError(errors.New("db fail"))

		_, err := repo.GetChatIDByUserID(context.Background(), uID)
		assert.ErrorContains(t, err, "failed to query chat id")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_UnlinkAccount(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	uID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM user_telegram_link WHERE user_id = \$1`).
			WithArgs(uID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.UnlinkAccount(context.Background(), uID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM user_telegram_link WHERE user_id = \$1`).
			WithArgs(uID).
			WillReturnError(errors.New("db fail"))

		err := repo.UnlinkAccount(context.Background(), uID)
		assert.ErrorContains(t, err, "failed to unlink account")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
