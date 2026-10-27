package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	ServiceName       string
	AppEnv            string
	HTTPAddr          string
	DatabaseURL       string
	RabbitURL         string
	RabbitExchange    string
	RabbitQueue       string
	ShutdownTimeout   time.Duration
	InfraOptional     bool
	MigrationsDir     string
	SeedAdminEmail    string
	SeedAdminPassword string
	SeedAdminName     string
}

func FromEnv() (Config, error) {
	cfg := Config{
		ServiceName:       getenv("SERVICE_NAME", "taskdemo"),
		AppEnv:            getenv("APP_ENV", "development"),
		HTTPAddr:          getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		RabbitURL:         os.Getenv("RABBITMQ_URL"),
		RabbitExchange:    getenv("RABBITMQ_EXCHANGE", "taskdemo.events"),
		RabbitQueue:       getenv("RABBITMQ_QUEUE", "taskdemo.task_completed"),
		ShutdownTimeout:   5 * time.Second,
		InfraOptional:     getenv("INFRA_OPTIONAL", "false") == "true",
		MigrationsDir:     getenv("MIGRATIONS_DIR", "migrations/pgsql"),
		SeedAdminEmail:    getenv("SEED_ADMIN_EMAIL", "admin@example.com"),
		SeedAdminPassword: getenv("SEED_ADMIN_PASSWORD", "changeme"),
		SeedAdminName:     getenv("SEED_ADMIN_NAME", "Admin"),
	}
	if raw := os.Getenv("SHUTDOWN_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = d
	}
	if !cfg.InfraOptional {
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("DATABASE_URL is required")
		}
		if cfg.RabbitURL == "" {
			return Config{}, fmt.Errorf("RABBITMQ_URL is required")
		}
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
