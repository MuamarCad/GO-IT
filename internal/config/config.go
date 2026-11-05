package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Env           string
	HTTPAddr      string
	DatabaseURL   string
	SessionSecret string
	CSRFSecret    string
}

func Load() (Config, error) {
	cfg := Config{
		Env:           getenv("APP_ENV", "development"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		SessionSecret: getenv("SESSION_SECRET", "dev-session-secret-change-me"),
		CSRFSecret:    getenv("CSRF_SECRET", "dev-csrf-secret-change-me-32b!!"),
	}
	if cfg.DatabaseURL == "" && cfg.Env == "production" {
		return Config{}, fmt.Errorf("DATABASE_URL is required in production")
	}
	if len(cfg.CSRFSecret) < 16 {
		return Config{}, fmt.Errorf("CSRF_SECRET must be at least 16 characters")
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
