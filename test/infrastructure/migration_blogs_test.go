//go:build integration

package infrastructure_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestBlogsMigration(t *testing.T) {
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

	upMatches, err := filepath.Glob(filepath.Join(absDir, "*_create_blogs.up.sql"))
	require.NoError(t, err)
	require.Len(t, upMatches, 1)
	versionStr := strings.SplitN(filepath.Base(upMatches[0]), "_", 2)[0]
	blogsVersion64, err := strconv.ParseUint(versionStr, 10, 64)
	require.NoError(t, err)
	blogsVersion := uint(blogsVersion64)
	require.Greater(t, blogsVersion, blogsAfterAccountsVersion)

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

	require.NoError(t, m.Migrate(blogsVersion))

	var blogsExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables WHERE table_name = 'blogs'
		)
	`).Scan(&blogsExists)
	require.NoError(t, err)
	require.True(t, blogsExists)

	var slugIndex string
	err = pool.QueryRow(ctx, `
		SELECT indexdef FROM pg_indexes
		WHERE tablename = 'blogs' AND indexname = 'blogs_slug_uidx'
	`).Scan(&slugIndex)
	require.NoError(t, err)
	require.Contains(t, slugIndex, "UNIQUE")
	require.NotContains(t, strings.ToUpper(slugIndex), " WHERE ")

	require.NoError(t, m.Migrate(blogsAfterAccountsVersion))

	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables WHERE table_name = 'blogs'
		)
	`).Scan(&blogsExists)
	require.NoError(t, err)
	require.False(t, blogsExists)

	var accountsExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables WHERE table_name = 'accounts'
		)
	`).Scan(&accountsExists)
	require.NoError(t, err)
	require.True(t, accountsExists)

	var tasksExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables WHERE table_name = 'tasks'
		)
	`).Scan(&tasksExists)
	require.NoError(t, err)
	require.True(t, tasksExists)
}
