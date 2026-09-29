package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewPool_EmptyDBURL(t *testing.T) {
	config.Envs.DBURL = ""
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.EqualError(t, err, "variável de ambiente DB_URL não encontrada")
}

func TestNewPool_InvalidDBURL(t *testing.T) {
	config.Envs.DBURL = "://invalid" // Malformed
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.ErrorContains(t, err, "falha ao realizar o parse da DB_URL")
}

func TestNewPool_ConnectionError(t *testing.T) {
	// A valid URL but points to a non-existent server to trigger NewWithConfig or Ping error
	config.Envs.DBURL = "postgres://user:pass@255.255.255.255:5432/db"
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.Error(t, err)
}

func TestNewPool_NewWithConfigError(t *testing.T) {
	origNew := newPoolWithConfig
	defer func() { newPoolWithConfig = origNew }()

	newPoolWithConfig = func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
		return nil, errors.New("simulated pool creation error")
	}

	config.Envs.DBURL = "postgres://user:pass@localhost:5432/db"
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.ErrorContains(t, err, "falha ao criar o connection pool")
}

func TestNewPool_Success(t *testing.T) {
	origNew := newPoolWithConfig
	origPing := pingPool
	defer func() {
		newPoolWithConfig = origNew
		pingPool = origPing
	}()

	dummyPool := &pgxpool.Pool{}
	newPoolWithConfig = func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
		return dummyPool, nil
	}
	pingPool = func(p *pgxpool.Pool, ctx context.Context) error {
		return nil
	}

	config.Envs.DBURL = "postgres://user:pass@localhost:5432/db"
	pool, err := NewPool()
	assert.NoError(t, err)
	assert.Equal(t, dummyPool, pool)
}
