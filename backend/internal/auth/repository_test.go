package auth

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
		t.Fatalf("erro ao criar mock do pool: %v", err)
	}
	repo := NewRepository(mock)
	return mock, repo
}

func TestRepository_CreateUser(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", now, now)

	mock.ExpectQuery(`INSERT INTO "user"`).
		WithArgs("Test", "test@test.com", "hash").
		WillReturnRows(rows)

	user, err := repo.CreateUser(context.Background(), "Test", "test@test.com", "hash")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "Test", user.Name)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_CreateUser_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO "user"`).
		WithArgs("Test", "test@test.com", "hash").
		WillReturnError(errors.New("db error"))

	_, err := repo.CreateUser(context.Background(), "Test", "test@test.com", "hash")
	assert.EqualError(t, err, "erro ao cadastrar usuário: db error")
}

func TestRepository_GetUserByEmail(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", "hash", now, now)

	mock.ExpectQuery(`SELECT id, name, email, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("test@test.com").
		WillReturnRows(rows)

	user, err := repo.GetUserByEmail(context.Background(), "test@test.com")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_GetUserByEmail_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, email, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("notfound@test.com").
		WillReturnError(errors.New("not found"))

	_, err := repo.GetUserByEmail(context.Background(), "notfound@test.com")
	assert.EqualError(t, err, "not found")
}

func TestRepository_GetUserByID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", now, now)

	mock.ExpectQuery(`SELECT id, name, email, created_at, updated_at FROM "user"`).
		WithArgs("1").
		WillReturnRows(rows)

	user, err := repo.GetUserByID(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_GetUserByID_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, email, created_at, updated_at FROM "user"`).
		WithArgs("999").
		WillReturnError(errors.New("not found"))

	_, err := repo.GetUserByID(context.Background(), "999")
	assert.EqualError(t, err, "not found")
}

func TestRepository_GetUserByIDWithHash(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", "hash", now, now)

	mock.ExpectQuery(`SELECT id, name, email, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("1").
		WillReturnRows(rows)

	user, err := repo.GetUserByIDWithHash(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "hash", user.PasswordHash)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_UpdateUser(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "created_at", "updated_at"}).
		AddRow("1", "NewName", "new@test.com", now, now)

	mock.ExpectQuery(`UPDATE "user" SET name = \$2, email = \$3, updated_at = NOW\(\)`).
		WithArgs("1", "NewName", "new@test.com").
		WillReturnRows(rows)

	user, err := repo.UpdateUser(context.Background(), "1", "NewName", "new@test.com")
	assert.NoError(t, err)
	assert.Equal(t, "NewName", user.Name)
	assert.Equal(t, "new@test.com", user.Email)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_UpdatePassword(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow("1")

	mock.ExpectQuery(`UPDATE "user" SET password_hash = \$2, updated_at = NOW\(\)`).
		WithArgs("1", "new_hash").
		WillReturnRows(rows)

	err := repo.UpdatePassword(context.Background(), "1", "new_hash")
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_DeleteUser(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow("1")

	mock.ExpectQuery(`DELETE FROM "user"`).
		WithArgs("1").
		WillReturnRows(rows)

	err := repo.DeleteUser(context.Background(), "1")
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}
