package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool cria e retorna um pool de conexões com o PostgreSQL usando pgxpool
func NewPool() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("variável de ambiente DB_URL não encontrada")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("falha ao realizar o parse da DB_URL: %w", err)
	}

	// Limites do Connection Pool para prevenir "too many clients" (RNF02)
	config.MaxConns = 50
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Criação do Pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar o connection pool: %w", err)
	}

	// Validação inicial (Ping)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("falha ao dar ping no banco de dados na inicialização: %w", err)
	}

	return pool, nil
}
