package config

import (
	"fmt"
	"os"
	"time"
)

type Environment struct {
	DBURL                 string
	RedisURL              string
	JWTSecret             string
	FrontendURL           string
	TelegramBotToken      string
	SMTPHost              string
	SMTPPort              string
	SMTPFrom              string
	LogLevel              string
	AlertCheckInterval    string
	BrapiToken            string
	Env                   string
	RedisTTLQuotes        time.Duration
	RedisTTLFundamentals  time.Duration
	RedisTTLExchangeRates time.Duration
	RedisTTLDividends     time.Duration
}

var Envs Environment

func Load() error {
	Envs = Environment{
		DBURL:              os.Getenv("DB_URL"),
		RedisURL:           os.Getenv("REDIS_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		FrontendURL:        os.Getenv("FRONTEND_URL"),
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           os.Getenv("SMTP_PORT"),
		SMTPFrom:           os.Getenv("SMTP_FROM"),
		LogLevel:           os.Getenv("LOG_LEVEL"),
		AlertCheckInterval: os.Getenv("ALERT_CHECK_INTERVAL"),
		BrapiToken:         os.Getenv("BRAPI_TOKEN"),
		Env:                os.Getenv("ENV"),

		RedisTTLQuotes:        parseDuration(os.Getenv("REDIS_TTL_QUOTES"), 5*time.Minute),
		RedisTTLFundamentals:  parseDuration(os.Getenv("REDIS_TTL_FUNDAMENTALS"), 24*time.Hour),
		RedisTTLExchangeRates: parseDuration(os.Getenv("REDIS_TTL_EXCHANGE_RATES"), 1*time.Hour),
		RedisTTLDividends:     parseDuration(os.Getenv("REDIS_TTL_DIVIDENDS"), 12*time.Hour),
	}

	if Envs.DBURL == "" {
		return fmt.Errorf("variável de ambiente obrigatória não configurada: DB_URL")
	}
	if Envs.JWTSecret == "" {
		return fmt.Errorf("variável de ambiente obrigatória não configurada: JWT_SECRET")
	}
	if Envs.FrontendURL == "" {
		return fmt.Errorf("variável de ambiente obrigatória não configurada: FRONTEND_URL")
	}

	return nil
}

func parseDuration(val string, defaultVal time.Duration) time.Duration {
	if val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
