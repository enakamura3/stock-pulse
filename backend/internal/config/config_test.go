package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("DB_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("JWT_SECRET", "supersecret_key_12345678901234567890")
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
	assert.Equal(t, "supersecret_key_12345678901234567890", Envs.JWTSecret)
	assert.Equal(t, "http://localhost:3000", Envs.FrontendURL)
	assert.Equal(t, 10*time.Minute, Envs.RedisTTLQuotes)
	assert.Equal(t, 24*time.Hour, Envs.RedisTTLFundamentals)
	assert.Equal(t, 15*time.Minute, Envs.JWTAccessTokenTTL)
	assert.Equal(t, 12*time.Hour, Envs.JWTRefreshTokenTTL)
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

func TestLoad_ShortJWTSecret(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost")
	os.Setenv("JWT_SECRET", "short_secret_under_32_chars")
	os.Setenv("FRONTEND_URL", "http://localhost:3000")
	defer func() {
		os.Unsetenv("DB_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("FRONTEND_URL")
	}()

	err := Load()
	assert.ErrorContains(t, err, "pelo menos 32 caracteres")
}

func TestLoad_MissingFrontendURL_Production(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost")
	os.Setenv("JWT_SECRET", "supersecret_key_12345678901234567890")
	os.Setenv("ENV", "production")
	os.Unsetenv("FRONTEND_URL")
	defer func() {
		os.Unsetenv("DB_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ENV")
	}()

	err := Load()
	assert.ErrorContains(t, err, "FRONTEND_URL")
}

func TestLoad_MissingFrontendURL_DevDefault(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost")
	os.Setenv("JWT_SECRET", "supersecret_key_12345678901234567890")
	os.Setenv("ENV", "development")
	os.Unsetenv("FRONTEND_URL")
	defer func() {
		os.Unsetenv("DB_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ENV")
	}()

	err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:3000", Envs.FrontendURL)
}

func TestParseDuration(t *testing.T) {
	assert.Equal(t, 5*time.Minute, parseDuration("", 5*time.Minute))
	assert.Equal(t, 5*time.Minute, parseDuration("invalid", 5*time.Minute))
	assert.Equal(t, 15*time.Minute, parseDuration("15m", 5*time.Minute))
}
