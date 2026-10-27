package taskrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type ctxKey struct{}

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Postgres struct {
	Pool *pgxpool.Pool
}

func (r Postgres) querier(ctx context.Context) querier {
	if tx, ok := ctx.Value(ctxKey{}).(pgx.Tx); ok {
		return tx
	}
	return r.Pool
}

func (r Postgres) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	txCtx := context.WithValue(ctx, ctxKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}

func (r Postgres) Save(ctx context.Context, task *aggregate.Task) error {
	now := time.Now().UTC()
	const q = `
INSERT INTO tasks (id, owner_id, title, status, due_date, created_at, updated_at, completed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
	title = EXCLUDED.title,
	status = EXCLUDED.status,
	due_date = EXCLUDED.due_date,
	updated_at = EXCLUDED.updated_at,
	completed_at = EXCLUDED.completed_at`
	_, err := r.querier(ctx).Exec(ctx, q, task.ID(), task.OwnerID(), task.Title(), string(task.Status()), task.DueDate(), task.CreatedAt(), now, task.CompletedAt())
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}

func (r Postgres) Get(ctx context.Context, id uuid.UUID) (*aggregate.Task, error) {
	const q = `
SELECT id, owner_id, title, status, due_date, created_at, completed_at
FROM tasks
WHERE id = $1`
	var (
		rowID       uuid.UUID
		ownerID     uuid.UUID
		title       string
		status      string
		dueDate     *time.Time
		createdAt   time.Time
		completedAt *time.Time
	)
	err := r.querier(ctx).QueryRow(ctx, q, id).Scan(&rowID, &ownerID, &title, &status, &dueDate, &createdAt, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.New(errs.CodeTaskNotFound, errs.KindNotFound, fmt.Errorf("task %s not found", id))
	}
	if err != nil {
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return aggregate.RestoreTask(rowID, ownerID, title, enum.Status(status), dueDate, createdAt, completedAt), nil
}
