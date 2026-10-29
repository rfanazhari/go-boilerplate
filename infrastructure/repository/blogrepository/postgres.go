package blogrepository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/pgxtx"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Postgres implements command persistence. Public/admin projections are added
// separately before this adapter is wired as the complete BlogRepository.
type Postgres struct{ Pool *pgxpool.Pool }

func (r Postgres) querier(ctx context.Context) querier {
	if tx, ok := pgxtx.From(ctx); ok {
		return tx
	}
	return r.Pool
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		(pgErr.ConstraintName == "blogs_pkey" || pgErr.ConstraintName == "blogs_slug_key") {
		return errs.New(errs.CodeBlogIdentifierConflict, errs.KindConflict, err)
	}
	return errs.New(errs.CodeInternalError, errs.KindInternal, err)
}

func scanBlog(row pgx.Row) (*aggregate.Blog, error) {
	var (
		id, ownerID                  uuid.UUID
		slug, title, content, status string
		createdAt, updatedAt         time.Time
		deletedAt                    *time.Time
	)
	if err := row.Scan(&id, &ownerID, &slug, &title, &content, &status, &createdAt, &updatedAt, &deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.New(errs.CodeBlogNotFound, errs.KindNotFound, err)
		}
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	parsed, ok := enum.ParseBlogStatus(status)
	if !ok {
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, errors.New("invalid stored blog status"))
	}
	return aggregate.RestoreBlog(id, ownerID, slug, title, content, parsed, createdAt, updatedAt, deletedAt), nil
}

// Create performs one insert. The caller owns bounded identifier retries and
// must start a fresh transaction/statement after a unique violation.
func (r Postgres) Create(ctx context.Context, blog *aggregate.Blog) error {
	const query = `INSERT INTO blogs
 (id, owner_id, slug, title, content, status, created_at, updated_at, deleted_at)
 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.querier(ctx).Exec(ctx, query, blog.ID(), blog.OwnerID(), blog.Slug(), blog.Title(), blog.Content(), string(blog.Status()), blog.CreatedAt(), blog.UpdatedAt(), blog.DeletedAt())
	if err != nil {
		return mapWriteError(err)
	}
	return nil
}

// GetByID includes soft-deleted records for application disclosure policy.
func (r Postgres) GetByID(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error) {
	const query = `SELECT id, owner_id, slug, title, content, status, created_at, updated_at, deleted_at
 FROM blogs WHERE id = $1`
	return scanBlog(r.querier(ctx).QueryRow(ctx, query, id))
}

func (r Postgres) GetForUpdate(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error) {
	tx, ok := pgxtx.From(ctx)
	if !ok {
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, errors.New("blog row lock requires a transaction"))
	}
	const query = `SELECT id, owner_id, slug, title, content, status, created_at, updated_at, deleted_at
 FROM blogs WHERE id = $1 FOR UPDATE`
	return scanBlog(tx.QueryRow(ctx, query, id))
}

// Update changes only mutable columns; it never inserts a missing blog.
func (r Postgres) Update(ctx context.Context, blog *aggregate.Blog) error {
	const query = `UPDATE blogs SET title = $2, content = $3, status = $4, updated_at = $5, deleted_at = $6 WHERE id = $1`
	tag, err := r.querier(ctx).Exec(ctx, query, blog.ID(), blog.Title(), blog.Content(), string(blog.Status()), blog.UpdatedAt(), blog.DeletedAt())
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return errs.New(errs.CodeBlogNotFound, errs.KindNotFound, nil)
	}
	return nil
}

func (r Postgres) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	// Covers callback errors, panics and failed commits, including cancellation.
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if err := fn(pgxtx.With(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}
