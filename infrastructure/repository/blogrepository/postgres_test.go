package blogrepository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/blogrepository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

// BeforeConnect injects connection errors without opening a socket. This checks
// the public adapter error boundary, not server constraints or transactions.
func TestPostgresCreateErrors(t *testing.T) {
	type testCase struct {
		name    string
		wantErr bool
		source  error
		code    string
		kind    errs.Kind
	}
	cases := []testCase{
		{name: "slug collision", wantErr: true, source: &pgconn.PgError{Code: "23505", ConstraintName: "blogs_slug_key"}, code: errs.CodeBlogIdentifierConflict, kind: errs.KindConflict},
		{name: "id collision", wantErr: true, source: &pgconn.PgError{Code: "23505", ConstraintName: "blogs_pkey"}, code: errs.CodeBlogIdentifierConflict, kind: errs.KindConflict},
		{name: "unknown uniqueness", wantErr: true, source: &pgconn.PgError{Code: "23505", ConstraintName: "other_key"}, code: errs.CodeInternalError, kind: errs.KindInternal},
		{name: "foreign key", wantErr: true, source: &pgconn.PgError{Code: "23503", ConstraintName: "blogs_owner_id_fkey"}, code: errs.CodeInternalError, kind: errs.KindInternal},
		{name: "wrong sqlstate", wantErr: true, source: &pgconn.PgError{Code: "23514", ConstraintName: "blogs_slug_key"}, code: errs.CodeInternalError, kind: errs.KindInternal},
		{name: "connection failure", wantErr: true, source: errors.New("connection unavailable"), code: errs.CodeInternalError, kind: errs.KindInternal},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			config, err := pgxpool.ParseConfig("postgres://unused:unused@127.0.0.1:1/unused?sslmode=disable")
			require.NoError(t, err)
			config.BeforeConnect = func(context.Context, *pgx.ConnConfig) error { return tt.source }
			pool, err := pgxpool.NewWithConfig(ctx, config)
			require.NoError(t, err)
			defer pool.Close()
			blog, err := aggregate.NewBlog(uuid.New(), uuid.New(), "A valid blog title", "", testdata.Instant())
			require.NoError(t, err)
			err = (blogrepository.Postgres{Pool: pool}).Create(ctx, blog)
			require.Equal(t, tt.wantErr, err != nil)
			var appErr *errs.Error
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, tt.code, appErr.Code)
			require.Equal(t, tt.kind, appErr.Kind)
			require.ErrorIs(t, err, tt.source)
		})
	}
}

func TestPostgresGetForUpdateRequiresTransaction(t *testing.T) {
	repo := blogrepository.Postgres{}
	blog, err := repo.GetForUpdate(context.Background(), uuid.New())
	require.Nil(t, blog)
	var appErr *errs.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, errs.CodeInternalError, appErr.Code)
}
