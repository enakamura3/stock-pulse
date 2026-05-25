package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// User representa o modelo do usuário conforme mapeado no banco de dados.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DBTX define a interface necessária para realizar queries, abstraindo o pgxpool.Pool para facilitar os testes.
type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository encapsula a conexão para operações na tabela user.
type Repository struct {
	db DBTX
}

// NewRepository cria uma nova instância de Repository.
func NewRepository(db DBTX) *Repository {
	return &Repository{db: db}
}

// CreateUser insere um novo registro de usuário na tabela.
func (r *Repository) CreateUser(ctx context.Context, name, email, passwordHash string) (*User, error) {
	query := `
		INSERT INTO "user" (name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, name, email, created_at, updated_at
	`
	user := &User{}
	err := r.db.QueryRow(ctx, query, name, email, passwordHash).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao cadastrar usuário: %w", err)
	}
	return user, nil
}

// GetUserByEmail busca um usuário pelo e-mail e retorna incluindo o hash de senha para validação de login.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM "user"
		WHERE email = $1
	`
	user := &User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID busca os dados públicos do usuário pelo ID (usado na rota /me).
func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, name, email, created_at, updated_at
		FROM "user"
		WHERE id = $1
	`
	user := &User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
