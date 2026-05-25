package alert

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func setupRepoTest(t *testing.T) (pgxmock.PgxPoolIface, *Repository) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	repo := NewRepository(mock)
	return mock, repo
}

func TestRepository_CreateAlert(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	a := &Alert{
		UserID:      "u1",
		AssetID:     "a1",
		TargetPrice: 150.0,
		Condition:   "ABOVE",
		Status:      "ACTIVE",
	}

	mock.ExpectQuery("INSERT INTO alert").
		WithArgs(a.UserID, a.AssetID, a.TargetPrice, a.Condition, a.Status).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow("1", time.Now()))

	err := repo.CreateAlert(context.Background(), a)
	assert.NoError(t, err)
	assert.Equal(t, "1", a.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAlertsByUserID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT a.id, a.user_id, a.asset_id").
		WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "asset_id", "ticker", "name", "currency", "target_price", "condition", "status", "triggered_at", "created_at"}).
			AddRow("1", "u1", "a1", "AAPL", "Apple", "USD", 150.0, "ABOVE", "ACTIVE", &now, now))

	alerts, err := repo.GetAlertsByUserID(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAlertByID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT a.id, a.user_id, a.asset_id").
		WithArgs("1", "u1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "asset_id", "ticker", "name", "currency", "target_price", "condition", "status", "triggered_at", "created_at"}).
			AddRow("1", "u1", "a1", "AAPL", "Apple", "USD", 150.0, "ABOVE", "ACTIVE", &now, now))

	alert, err := repo.GetAlertByID(context.Background(), "1", "u1")
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, "1", alert.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteAlert(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM alert").
		WithArgs("1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.DeleteAlert(context.Background(), "1", "u1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ToggleAlertStatus(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery("UPDATE alert").
		WithArgs("1", "u1").
		WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DISABLED"))

	status, err := repo.ToggleAlertStatus(context.Background(), "1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, "DISABLED", status)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetActiveAlerts(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectQuery("SELECT a.id, a.user_id, a.asset_id").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "asset_id", "ticker", "name", "currency", "target_price", "condition", "status", "triggered_at", "created_at", "user_name", "user_email"}).
			AddRow("1", "u1", "a1", "AAPL", "Apple", "USD", 150.0, "ABOVE", "ACTIVE", &now, now, "User", "user@test.com"))

	alerts, err := repo.GetActiveAlerts(context.Background())
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_MarkAlertTriggered(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec("UPDATE alert").
		WithArgs("1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.MarkAlertTriggered(context.Background(), "1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAssetByTicker(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT id FROM asset").
		WithArgs("AAPL").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("a1"))

	id, err := repo.GetAssetByTicker(context.Background(), "AAPL")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateAsset(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO asset").
		WithArgs("AAPL", "Apple", "EQUITY_US", "USD").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("a1"))

	id, err := repo.CreateAsset(context.Background(), "AAPL", "Apple", "EQUITY_US", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)

	assert.NoError(t, mock.ExpectationsWereMet())
}
