package fixedincome

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestRepository_ValidatePortfolioOwnership(t *testing.T) {
	ctx := context.Background()

	t.Run("Success - Portfolio belongs to user", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		assert.NoError(t, err)
		defer mock.Close()

		repo := NewRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM portfolio WHERE id = \$1 AND user_id = \$2\)`).
			WithArgs("p1", "u1").
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		err = repo.ValidatePortfolioOwnership(ctx, "p1", "u1")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Forbidden/Not Found - Portfolio does not exist or not owned", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		assert.NoError(t, err)
		defer mock.Close()

		repo := NewRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM portfolio WHERE id = \$1 AND user_id = \$2\)`).
			WithArgs("p2", "u1").
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		err = repo.ValidatePortfolioOwnership(ctx, "p2", "u1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "carteira não encontrada ou permissão negada")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database Error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		assert.NoError(t, err)
		defer mock.Close()

		repo := NewRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM portfolio WHERE id = \$1 AND user_id = \$2\)`).
			WithArgs("p1", "u1").
			WillReturnError(errors.New("connection failed"))

		err = repo.ValidatePortfolioOwnership(ctx, "p1", "u1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
