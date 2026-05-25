package watchlist

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func setupRepoTest(t *testing.T) (pgxmock.PgxPoolIface, *Repository) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("erro ao criar mock: %v", err)
	}
	return mock, NewRepository(mock)
}

func TestRepository_CreateWatchlist(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "user_id", "name", "created_at"}).
		AddRow("1", "u1", "My List", now)

	mock.ExpectQuery(`INSERT INTO watchlist`).
		WithArgs("u1", "My List").
		WillReturnRows(rows)

	w, err := repo.CreateWatchlist(context.Background(), "u1", "My List")
	assert.NoError(t, err)
	assert.Equal(t, "1", w.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateWatchlist_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO watchlist`).
		WithArgs("u1", "My List").
		WillReturnError(errors.New("db err"))

	_, err := repo.CreateWatchlist(context.Background(), "u1", "My List")
	assert.ErrorContains(t, err, "erro ao criar watchlist")
}

func TestRepository_GetWatchlistsByUserID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "user_id", "name", "created_at"}).
		AddRow("1", "u1", "My List", now)

	mock.ExpectQuery(`SELECT id, user_id, name, created_at FROM watchlist`).
		WithArgs("u1").
		WillReturnRows(rows)

	lists, err := repo.GetWatchlistsByUserID(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Len(t, lists, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetWatchlistByID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "user_id", "name", "created_at"}).
		AddRow("1", "u1", "My List", now)

	mock.ExpectQuery(`SELECT id, user_id, name, created_at FROM watchlist`).
		WithArgs("1", "u1").
		WillReturnRows(rows)

	w, err := repo.GetWatchlistByID(context.Background(), "1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, "1", w.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteWatchlist(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM watchlist`).
		WithArgs("1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.DeleteWatchlist(context.Background(), "1", "u1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteWatchlist_NotFound(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM watchlist`).
		WithArgs("1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.DeleteWatchlist(context.Background(), "1", "u1")
	assert.ErrorContains(t, err, "não encontrada")
}

func TestRepository_GetAssetByTicker(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow("a1")
	mock.ExpectQuery(`SELECT id FROM asset`).
		WithArgs("AAPL").
		WillReturnRows(rows)

	id, err := repo.GetAssetByTicker(context.Background(), "aapl")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)
}

func TestRepository_CreateAsset(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow("a1")
	mock.ExpectQuery(`INSERT INTO asset`).
		WithArgs("AAPL", "Apple", "stock", "USD").
		WillReturnRows(rows)

	id, err := repo.CreateAsset(context.Background(), "aapl", "Apple", "stock", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)
}

func TestRepository_AddWatchlistItem(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "watchlist_id", "asset_id", "added_at"}).
		AddRow("i1", "w1", "a1", now)

	mock.ExpectQuery(`INSERT INTO watchlist_item`).
		WithArgs("w1", "a1").
		WillReturnRows(rows)

	item, err := repo.AddWatchlistItem(context.Background(), "w1", "a1")
	assert.NoError(t, err)
	assert.Equal(t, "i1", item.ID)
}

func TestRepository_RemoveWatchlistItem(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM watchlist_item`).
		WithArgs("w1", "AAPL").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.RemoveWatchlistItem(context.Background(), "w1", "aapl")
	assert.NoError(t, err)
}

func TestRepository_GetWatchlistItems(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "watchlist_id", "asset_id", "added_at", "ticker", "name", "asset_type", "currency"}).
		AddRow("i1", "w1", "a1", now, "AAPL", "Apple", "stock", "USD")

	mock.ExpectQuery(`SELECT wi.id`).
		WithArgs("w1").
		WillReturnRows(rows)

	items, err := repo.GetWatchlistItems(context.Background(), "w1")
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "AAPL", items[0].Ticker)
}

// Errors testing for coverage
func TestRepository_GetWatchlistItems_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT wi.id`).
		WithArgs("w1").
		WillReturnError(errors.New("db error"))

	_, err := repo.GetWatchlistItems(context.Background(), "w1")
	assert.ErrorContains(t, err, "db error")
}

func TestRepository_GetWatchlistsByUserID_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id`).
		WithArgs("u1").
		WillReturnError(errors.New("db err"))

	_, err := repo.GetWatchlistsByUserID(context.Background(), "u1")
	assert.ErrorContains(t, err, "db err")
}
