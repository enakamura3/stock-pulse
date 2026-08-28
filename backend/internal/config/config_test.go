package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("DB_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("JWT_SECRET", "supersecret")
	os.Setenv("FRONTEND_URL", "http://localhost:3000")
	os.Setenv("REDIS_TTL_QUOTES", "10m")
	defer func() {
		os.Unsetenv("DB_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("FRONTEND_URL")
		os.Unsetenv("REDIS_TTL_QUOTES")
	}()

	err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", Envs.DBURL)
	assert.Equal(t, "supersecret", Envs.JWTSecret)
	assert.Equal(t, "http://localhost:3000", Envs.FrontendURL)
	assert.Equal(t, 10*time.Minute, Envs.RedisTTLQuotes)
	assert.Equal(t, 24*time.Hour, Envs.RedisTTLFundamentals)
}

func TestLoad_MissingDBURL(t *testing.T) {
	os.Unsetenv("DB_URL")
	err := Load()
	assert.ErrorContains(t, err, "DB_URL")
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost")
	os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")

	err := Load()
	assert.ErrorContains(t, err, "JWT_SECRET")
}

func TestLoad_MissingFrontendURL(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost")
	os.Setenv("JWT_SECRET", "secret")
	os.Unsetenv("FRONTEND_URL")
	defer func() {
		os.Unsetenv("DB_URL")
		os.Unsetenv("JWT_SECRET")
	}()

	err := Load()
	assert.ErrorContains(t, err, "FRONTEND_URL")
}

func TestParseDuration(t *testing.T) {
	assert.Equal(t, 5*time.Minute, parseDuration("", 5*time.Minute))
	assert.Equal(t, 5*time.Minute, parseDuration("invalid", 5*time.Minute))
	assert.Equal(t, 15*time.Minute, parseDuration("15m", 5*time.Minute))
}
