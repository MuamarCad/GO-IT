package config

import (
	"fmt"
	"os"
	"strings"
)

const defaultSessionSecret = "dev-session-secret-change-me"

type Config struct {
	Env           string
	HTTPAddr      string
	DatabaseURL   string
	SessionSecret string // ключ подписи cookie сессии (HMAC-SHA256)
}

func Load() (Config, error) {
	cfg := Config{
		Env:           getenv("APP_ENV", "development"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		SessionSecret: getenv("SESSION_SECRET", defaultSessionSecret),
	}
	if cfg.DatabaseURL == "" && cfg.Env == "production" {
		return Config{}, fmt.Errorf("DATABASE_URL is required in production")
	}
	if len(cfg.SessionSecret) < 16 {
		return Config{}, fmt.Errorf("SESSION_SECRET must be at least 16 characters")
	}
	if cfg.Env == "production" && cfg.SessionSecret == defaultSessionSecret {
		return Config{}, fmt.Errorf("SESSION_SECRET must be changed in production")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
