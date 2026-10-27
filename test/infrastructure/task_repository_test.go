//go:build integration

package infrastructure_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	taskrepository "github.com/rfanazhari/go-boilerplate/infrastructure/repository/taskrepository"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

func TestPostgresTaskRepository(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@127.0.0.1:5432/taskdemo?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skip(err.Error())
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skip(err.Error())
	}
	repo := taskrepository.Postgres{Pool: pool}
	task, err := aggregate.NewTask(uuid.New(), uuid.MustParse("e0000000-0000-4000-8000-000000000001"), testdata.TitlePrepareReport(), nil, testdata.Instant())
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, task))
	got, err := repo.Get(ctx, task.ID())
	require.NoError(t, err)
	require.Equal(t, task.Title(), got.Title())
}
