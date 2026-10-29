package infrastructure_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const blogsMigrationBase = "20261029000100_create_blogs"

func blogsMigrationsDir(t *testing.T) string {
	t.Helper()
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		migrationsDir = filepath.Join(filepath.Dir(file), "..", "..", "migrations", "pgsql")
	}
	absDir, err := filepath.Abs(migrationsDir)
	require.NoError(t, err)
	return absDir
}

func TestCreateBlogsMigrationFilesDefineSchema(t *testing.T) {
	dir := blogsMigrationsDir(t)
	upPath := filepath.Join(dir, blogsMigrationBase+".up.sql")
	downPath := filepath.Join(dir, blogsMigrationBase+".down.sql")

	up, err := os.ReadFile(upPath)
	require.NoError(t, err, "blogs up migration must exist")
	down, err := os.ReadFile(downPath)
	require.NoError(t, err, "blogs down migration must exist")

	upSQL := string(up)
	downSQL := string(down)

	require.Contains(t, upSQL, "CREATE TABLE blogs")
	require.Contains(t, upSQL, "owner_id UUID NOT NULL REFERENCES accounts (id)")
	require.NotContains(t, upSQL, "ON DELETE CASCADE")
	require.Contains(t, upSQL, "CONSTRAINT blogs_slug_key UNIQUE (slug)")
	require.Contains(t, upSQL, "CONSTRAINT blogs_status_check CHECK (status IN ('draft', 'published'))")
	require.Contains(t, upSQL, "char_length(title) >= 10")
	require.Contains(t, upSQL, "char_length(title) <= 200")
	require.Contains(t, upSQL, "content TEXT NOT NULL DEFAULT ''")
	require.Contains(t, upSQL, "deleted_at TIMESTAMPTZ")
	require.Contains(t, upSQL, "created_at TIMESTAMPTZ NOT NULL")
	require.Contains(t, upSQL, "updated_at TIMESTAMPTZ NOT NULL")
	require.Contains(t, upSQL, "blogs_owner_id_INDEX")
	require.Contains(t, upSQL, "blogs_public_list_order_idx")
	require.Contains(t, upSQL, "WHERE status = 'published' AND deleted_at IS NULL")
	require.Contains(t, upSQL, "blogs_admin_list_order_idx")
	require.Contains(t, upSQL, "created_at DESC, id DESC")

	require.Contains(t, downSQL, "DROP TABLE IF EXISTS blogs")
	require.NotContains(t, downSQL, "DROP TABLE IF EXISTS accounts")
	require.NotContains(t, downSQL, "DROP TABLE IF EXISTS tasks")
	require.NotContains(t, downSQL, "DROP TABLE IF EXISTS credential_tokens")
}
