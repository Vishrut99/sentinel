package config

import (
	"fmt"
	"os"
)

// Config stores runtime settings loaded from environment variables.
type Config struct {
	DBURL                string
	JWTSecret            string
	Port                 string
	Env                  string
	REDIS_URL            string
	AI_TRIAGE_URL        string
	AI_TRIAGE_API_KEY    string
	BootstrapAdminSecret string
}

// Load reads environment variables and returns validated config.
func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}

	cfg := Config{
		DBURL:                dbURL,
		JWTSecret:            os.Getenv("JWT_SECRET"),
		Port:                 os.Getenv("PORT"),
		Env:                  os.Getenv("ENV"),
		REDIS_URL:            os.Getenv("REDIS_URL"),
		AI_TRIAGE_URL:        os.Getenv("AI_TRIAGE_URL"),
		AI_TRIAGE_API_KEY:    os.Getenv("AI_TRIAGE_API_KEY"),
		BootstrapAdminSecret: os.Getenv("BOOTSTRAP_ADMIN_SECRET"),
	}

	if cfg.DBURL == "" {
		return Config{}, fmt.Errorf("config.Load: DB_URL or DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("config.Load: JWT_SECRET is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	return cfg, nil
}
