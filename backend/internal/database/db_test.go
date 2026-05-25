package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPool_EmptyDBURL(t *testing.T) {
	os.Setenv("DB_URL", "")
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.EqualError(t, err, "variável de ambiente DB_URL não encontrada")
}

func TestNewPool_InvalidDBURL(t *testing.T) {
	os.Setenv("DB_URL", "://invalid") // Malformed
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.ErrorContains(t, err, "falha ao realizar o parse da DB_URL")
}

func TestNewPool_ConnectionError(t *testing.T) {
	// A valid URL but points to a non-existent server to trigger NewWithConfig or Ping error
	os.Setenv("DB_URL", "postgres://user:pass@255.255.255.255:5432/db")
	pool, err := NewPool()
	assert.Nil(t, pool)
	assert.Error(t, err)
}
