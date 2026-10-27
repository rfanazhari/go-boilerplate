package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/infrastructure/config"
)

func TestFromEnvParsesSeedAdminVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/taskdemo")
	t.Setenv("RABBITMQ_URL", "amqp://localhost")
	t.Setenv("SEED_ADMIN_EMAIL", "ops@example.com")
	t.Setenv("SEED_ADMIN_PASSWORD", "secret-pass")
	t.Setenv("SEED_ADMIN_NAME", "Ops Admin")

	cfg, err := config.FromEnv()
	require.NoError(t, err)
	require.Equal(t, "ops@example.com", cfg.SeedAdminEmail)
	require.Equal(t, "secret-pass", cfg.SeedAdminPassword)
	require.Equal(t, "Ops Admin", cfg.SeedAdminName)
}

func TestFromEnvSeedAdminDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/taskdemo")
	t.Setenv("RABBITMQ_URL", "amqp://localhost")
	t.Setenv("SEED_ADMIN_EMAIL", "")
	t.Setenv("SEED_ADMIN_PASSWORD", "")
	t.Setenv("SEED_ADMIN_NAME", "")

	cfg, err := config.FromEnv()
	require.NoError(t, err)
	require.Equal(t, "admin@example.com", cfg.SeedAdminEmail)
	require.Equal(t, "changeme", cfg.SeedAdminPassword)
	require.Equal(t, "Admin", cfg.SeedAdminName)
}
