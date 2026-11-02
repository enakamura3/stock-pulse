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
	rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", "USER", now, now)

	mock.ExpectQuery(`INSERT INTO "user"`).
		WithArgs("Test", "test@test.com", "hash", "USER").
		WillReturnRows(rows)

	user, err := repo.CreateUser(context.Background(), "Test", "test@test.com", "hash")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "Test", user.Name)
	assert.Equal(t, "USER", user.Role)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_CreateUserWithRole(t *testing.T) {
	t.Run("Explicit role", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
			AddRow("1", "Admin", "admin@test.com", "ADMIN", now, now)

		mock.ExpectQuery(`INSERT INTO "user"`).
			WithArgs("Admin", "admin@test.com", "hash", "ADMIN").
			WillReturnRows(rows)

		user, err := repo.CreateUserWithRole(context.Background(), "Admin", "admin@test.com", "hash", "ADMIN")
		assert.NoError(t, err)
		assert.Equal(t, "ADMIN", user.Role)
	})

	t.Run("Default empty role to USER", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
			AddRow("1", "Test", "test@test.com", "USER", now, now)

		mock.ExpectQuery(`INSERT INTO "user"`).
			WithArgs("Test", "test@test.com", "hash", "USER").
			WillReturnRows(rows)

		user, err := repo.CreateUserWithRole(context.Background(), "Test", "test@test.com", "hash", "")
		assert.NoError(t, err)
		assert.Equal(t, "USER", user.Role)
	})
}

func TestRepository_CreateUser_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO "user"`).
		WithArgs("Test", "test@test.com", "hash", "USER").
		WillReturnError(errors.New("db error"))

	_, err := repo.CreateUser(context.Background(), "Test", "test@test.com", "hash")
	assert.EqualError(t, err, "erro ao cadastrar usuário: db error")
}

func TestRepository_GetUserByEmail(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "password_hash", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", "USER", "hash", now, now)

	mock.ExpectQuery(`SELECT id, name, email, role, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("test@test.com").
		WillReturnRows(rows)

	user, err := repo.GetUserByEmail(context.Background(), "test@test.com")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "USER", user.Role)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_GetUserByEmail_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, email, role, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("notfound@test.com").
		WillReturnError(errors.New("not found"))

	_, err := repo.GetUserByEmail(context.Background(), "notfound@test.com")
	assert.EqualError(t, err, "not found")
}

func TestRepository_GetUserByID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", "ADMIN", now, now)

	mock.ExpectQuery(`SELECT id, name, email, role, created_at, updated_at FROM "user"`).
		WithArgs("1").
		WillReturnRows(rows)

	user, err := repo.GetUserByID(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "ADMIN", user.Role)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_GetUserByID_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, email, role, created_at, updated_at FROM "user"`).
		WithArgs("999").
		WillReturnError(errors.New("not found"))

	_, err := repo.GetUserByID(context.Background(), "999")
	assert.EqualError(t, err, "not found")
}

func TestRepository_GetUserByIDWithHash(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "password_hash", "created_at", "updated_at"}).
		AddRow("1", "Test", "test@test.com", "USER", "hash", now, now)

	mock.ExpectQuery(`SELECT id, name, email, role, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("1").
		WillReturnRows(rows)

	user, err := repo.GetUserByIDWithHash(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "hash", user.PasswordHash)
	assert.Equal(t, "USER", user.Role)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_UpdateUser(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
		AddRow("1", "NewName", "new@test.com", "USER", now, now)

	mock.ExpectQuery(`UPDATE "user" SET name = \$2, email = \$3, updated_at = NOW\(\)`).
		WithArgs("1", "NewName", "new@test.com").
		WillReturnRows(rows)

	user, err := repo.UpdateUser(context.Background(), "1", "NewName", "new@test.com")
	assert.NoError(t, err)
	assert.Equal(t, "NewName", user.Name)
	assert.Equal(t, "new@test.com", user.Email)
	assert.Equal(t, "USER", user.Role)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations failed: %v", err)
	}
}

func TestRepository_UpdateUserRole(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"id"}).AddRow("1")

		mock.ExpectQuery(`UPDATE "user" SET role = \$2, updated_at = NOW\(\)`).
			WithArgs("1", "ADMIN").
			WillReturnRows(rows)

		err := repo.UpdateUserRole(context.Background(), "1", "ADMIN")
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`UPDATE "user" SET role = \$2, updated_at = NOW\(\)`).
			WithArgs("1", "ADMIN").
			WillReturnError(errors.New("db error"))

		err := repo.UpdateUserRole(context.Background(), "1", "ADMIN")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao atualizar papel do usuário: db error")
	})
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

func TestRepository_GetUserByIDWithHash_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, email, role, password_hash, created_at, updated_at FROM "user"`).
		WithArgs("1").
		WillReturnError(errors.New("db error"))

	_, err := repo.GetUserByIDWithHash(context.Background(), "1")
	assert.EqualError(t, err, "db error")
}

func TestRepository_UpdateUser_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`UPDATE "user" SET name = \$2, email = \$3, updated_at = NOW\(\)`).
		WithArgs("1", "NewName", "new@test.com").
		WillReturnError(errors.New("db error"))

	_, err := repo.UpdateUser(context.Background(), "1", "NewName", "new@test.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "erro ao atualizar usuário: db error")
}

func TestRepository_UpdatePassword_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`UPDATE "user" SET password_hash = \$2, updated_at = NOW\(\)`).
		WithArgs("1", "new_hash").
		WillReturnError(errors.New("db error"))

	err := repo.UpdatePassword(context.Background(), "1", "new_hash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "erro ao atualizar senha: db error")
}

func TestRepository_DeleteUser_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`DELETE FROM "user"`).
		WithArgs("1").
		WillReturnError(errors.New("db error"))

	err := repo.DeleteUser(context.Background(), "1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "erro ao excluir usuário: db error")
}

func TestRepository_ListUsers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
			AddRow("1", "Admin", "admin@test.com", "ADMIN", now, now).
			AddRow("2", "User", "user@test.com", "USER", now, now)

		mock.ExpectQuery(`SELECT id, name, email, role, created_at, updated_at FROM "user" ORDER BY created_at ASC`).
			WillReturnRows(rows)

		users, err := repo.ListUsers(context.Background())
		assert.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, "ADMIN", users[0].Role)
		assert.Equal(t, "USER", users[1].Role)
	})

	t.Run("Query Error", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, name, email, role, created_at, updated_at FROM "user" ORDER BY created_at ASC`).
			WillReturnError(errors.New("query failed"))

		users, err := repo.ListUsers(context.Background())
		assert.Nil(t, users)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao listar usuários")
	})

	t.Run("Scan Error", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
			AddRow("1", "Admin", "admin@test.com", "ADMIN", "invalid-time", time.Now())

		mock.ExpectQuery(`SELECT id, name, email, role, created_at, updated_at FROM "user" ORDER BY created_at ASC`).
			WillReturnRows(rows)

		users, err := repo.ListUsers(context.Background())
		assert.Nil(t, users)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "erro ao ler usuário")
	})

	t.Run("Rows Err", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "name", "email", "role", "created_at", "updated_at"}).
			AddRow("1", "Admin", "admin@test.com", "ADMIN", now, now).
			CloseError(errors.New("iteration failed"))

		mock.ExpectQuery(`SELECT id, name, email, role, created_at, updated_at FROM "user" ORDER BY created_at ASC`).
			WillReturnRows(rows)

		users, err := repo.ListUsers(context.Background())
		assert.Nil(t, users)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "erro após iterar usuários")
	})
}
