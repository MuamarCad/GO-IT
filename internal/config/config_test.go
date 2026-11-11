package config

import "testing"

func TestLoadDefaultsAndValidation(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err != nil {
		t.Fatalf("значения по умолчанию должны работать в development: %v", err)
	}

	t.Setenv("SESSION_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Error("короткий SESSION_SECRET должен отклоняться")
	}

	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SESSION_SECRET", defaultSessionSecret)
	if _, err := Load(); err == nil {
		t.Error("в production секрет по умолчанию недопустим")
	}
	t.Setenv("SESSION_SECRET", "a-real-secret-0123456789")
	if _, err := Load(); err != nil {
		t.Errorf("корректный production-конфиг отклонён: %v", err)
	}
}
