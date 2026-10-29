//go:build integration

package infrastructure_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/blogrepository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/pgxtx"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

func TestPostgresBlogRepository(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@127.0.0.1:5432/taskdemo?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skip(err.Error())
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skip(err.Error())
	}

	ownerID := uuid.MustParse("e0000000-0000-4000-8000-000000000001")
	repo := blogrepository.Postgres{Pool: pool}
	blog := testdata.MustBlog(uuid.New(), ownerID, "it-round-"+uuid.NewString()[:8], enum.PublicationPublished, nil, testdata.Instant())
	require.NoError(t, repo.Save(ctx, blog))

	got, err := repo.Get(ctx, blog.ID())
	require.NoError(t, err)
	assert.Equal(t, blog.Title().String(), got.Title().String())
	assert.Equal(t, blog.Slug().String(), got.Slug().String())

	duplicate := testdata.MustBlog(uuid.New(), ownerID, blog.Slug().String(), enum.PublicationDraft, nil, testdata.Instant())
	err = repo.Save(ctx, duplicate)
	require.ErrorIs(t, err, repository.ErrUniqueSlug)

	deletedAt := testdata.Instant()
	deleted := testdata.MustBlog(uuid.New(), ownerID, "it-del-"+uuid.NewString()[:8], enum.PublicationPublished, &deletedAt, testdata.Instant())
	require.NoError(t, repo.Save(ctx, deleted))
	again := testdata.MustBlog(uuid.New(), ownerID, deleted.Slug().String(), enum.PublicationDraft, nil, testdata.Instant())
	err = repo.Save(ctx, again)
	require.ErrorIs(t, err, repository.ErrUniqueSlug)

	require.NoError(t, repo.RunInTransaction(ctx, func(txCtx context.Context) error {
		_, ok := pgxtx.From(txCtx)
		require.True(t, ok)
		locked, getErr := repo.Get(txCtx, blog.ID())
		require.NoError(t, getErr)
		assert.Equal(t, blog.ID(), locked.ID())
		return nil
	}))

	_, _, err = repo.GetPublicBySlug(ctx, deleted.Slug().String())
	require.Error(t, err)
}
