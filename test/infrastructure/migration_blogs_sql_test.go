package infrastructure_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blogsAfterAccountsVersion = uint(20261027225508)

type blogsMigrationSQLCase struct {
	name      string
	kind      string
	wantFrags []string
	forbid    []string
}

func TestBlogsMigrationSQL(t *testing.T) {
	cases := []blogsMigrationSQLCase{
		{
			name: "up creates blogs with global slug uniqueness and separate soft delete",
			kind: "up",
			wantFrags: []string{
				"CREATE TABLE blogs (",
				"owner_id UUID NOT NULL REFERENCES accounts (id)",
				"deleted_at TIMESTAMPTZ",
				"CONSTRAINT blogs_status_check CHECK (status IN ('draft', 'published'))",
				"CONSTRAINT blogs_slug_check CHECK (",
				"CREATE UNIQUE INDEX blogs_slug_uidx ON blogs (slug)",
				`CREATE INDEX "blogs_owner_id_INDEX" ON blogs (owner_id)`,
				`CREATE INDEX "blogs_public_list_INDEX"`,
				"WHERE deleted_at IS NULL AND status = 'published'",
			},
			forbid: []string{
				"INSERT INTO blogs",
				"ALTER TABLE accounts",
				"ALTER TABLE tasks",
				"CREATE UNIQUE INDEX blogs_slug_uidx ON blogs (slug) WHERE",
			},
		},
		{
			name: "down drops only blogs",
			kind: "down",
			wantFrags: []string{
				"DROP TABLE IF EXISTS blogs",
			},
			forbid: []string{
				"DROP TABLE accounts",
				"DROP TABLE tasks",
				"DROP TABLE credential_tokens",
				"ALTER TABLE",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			sqlText, version := loadBlogsMigrationSQL(t, tt.kind)
			require.Greater(t, version, blogsAfterAccountsVersion)

			for _, frag := range tt.wantFrags {
				assert.Contains(t, sqlText, frag)
			}
			for _, frag := range tt.forbid {
				assert.NotContains(t, sqlText, frag)
			}

			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			execBlogsMigrationSQL(t, db, mock, sqlText)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func loadBlogsMigrationSQL(t *testing.T, kind string) (string, uint) {
	t.Helper()
	dir := blogsMigrationsDir(t)
	pattern := filepath.Join(dir, "*_create_blogs."+kind+".sql")
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.Len(t, matches, 1, "expected one create_blogs %s migration", kind)

	base := filepath.Base(matches[0])
	versionStr := strings.SplitN(base, "_", 2)[0]
	version, err := strconv.ParseUint(versionStr, 10, 64)
	require.NoError(t, err)

	body, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	sqlText := strings.TrimSpace(string(body))
	require.NotEmpty(t, sqlText)
	return sqlText, uint(version)
}

func blogsMigrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations", "pgsql")
}

func execBlogsMigrationSQL(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock, sqlText string) {
	t.Helper()
	stmts := splitSQLStatements(sqlText)
	require.NotEmpty(t, stmts)
	for _, stmt := range stmts {
		mock.ExpectExec("^" + regexp.QuoteMeta(stmt) + "$").
			WillReturnResult(sqlmock.NewResult(0, 0))
		_, err := db.Exec(stmt)
		require.NoError(t, err, stmt)
	}
}

func splitSQLStatements(sqlText string) []string {
	var stmts []string
	for _, part := range strings.Split(sqlText, ";") {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
