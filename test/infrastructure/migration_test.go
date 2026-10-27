//go:build integration

package infrastructure_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/infrastructure/seed"
)

const tasksOnlyMigrationVersion = uint(20261027204911)
const accountsMigrationVersion = uint(20261027225508)

func TestAccountsTokensTaskOwnerMigrationUpDown(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@127.0.0.1:5432/taskdemo?sslmode=disable"
	}
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		migrationsDir = filepath.Join(filepath.Dir(file), "..", "..", "migrations", "pgsql")
	}
	absDir, err := filepath.Abs(migrationsDir)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skip(err.Error())
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skip(err.Error())
	}

	m, err := migrate.New("file://"+absDir, dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = m.Close()
	})

	require.NoError(t, m.Migrate(tasksOnlyMigrationVersion))

	taskID := uuid.New()
	now := time.Now().UTC()
	_, err = pool.Exec(ctx, `
		INSERT INTO tasks (id, title, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (id) DO NOTHING
	`, taskID, "legacy task", "todo", now)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(accountsMigrationVersion))

	var ownerID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT owner_id FROM tasks WHERE id = $1`, taskID).Scan(&ownerID)
	require.NoError(t, err)
	require.Equal(t, seed.BootstrapAdminID, ownerID)

	var adminCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounts WHERE id = $1 AND deleted_at IS NULL
	`, seed.BootstrapAdminID).Scan(&adminCount)
	require.NoError(t, err)
	require.Equal(t, 1, adminCount)

	require.NoError(t, m.Migrate(tasksOnlyMigrationVersion))

	var hasOwnerCol bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'tasks' AND column_name = 'owner_id'
		)
	`).Scan(&hasOwnerCol)
	require.NoError(t, err)
	require.False(t, hasOwnerCol)

	var accountsExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables WHERE table_name = 'accounts'
		)
	`).Scan(&accountsExists)
	require.NoError(t, err)
	require.False(t, accountsExists)
}
