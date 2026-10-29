package blogrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/pgxtx"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

const (
	SaveSQL = `
INSERT INTO blogs (id, owner_id, slug, title, content, status, deleted_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
	owner_id = EXCLUDED.owner_id,
	slug = EXCLUDED.slug,
	title = EXCLUDED.title,
	content = EXCLUDED.content,
	status = EXCLUDED.status,
	deleted_at = EXCLUDED.deleted_at,
	updated_at = EXCLUDED.updated_at`

	GetSQL = `
SELECT id, owner_id, slug, title, content, status, deleted_at, created_at, updated_at
FROM blogs
WHERE id = $1`

	GetForUpdateSQL = GetSQL + `
FOR UPDATE`

	GetPublicBySlugSQL = `
SELECT blogs.id, blogs.owner_id, blogs.slug, blogs.title, blogs.content, blogs.status, blogs.deleted_at, blogs.created_at, blogs.updated_at, accounts.name
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id
WHERE blogs.slug = $1
AND blogs.status = 'published'
AND blogs.deleted_at IS NULL
AND accounts.status = 'active'
AND accounts.deleted_at IS NULL`

	ListPublicCountSQL = `
SELECT COUNT(*)
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id
WHERE blogs.status = 'published'
AND blogs.deleted_at IS NULL
AND accounts.status = 'active'
AND accounts.deleted_at IS NULL`

	ListPublicSQL = `
SELECT blogs.id, blogs.owner_id, blogs.slug, blogs.title, blogs.status, accounts.name, blogs.created_at, blogs.deleted_at
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id
WHERE blogs.status = 'published'
AND blogs.deleted_at IS NULL
AND accounts.status = 'active'
AND accounts.deleted_at IS NULL
ORDER BY blogs.created_at DESC, blogs.id DESC
LIMIT $1 OFFSET $2`

	ListAdminCountSQL = `
SELECT COUNT(*)
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id`

	ListAdminSQL = `
SELECT blogs.id, blogs.owner_id, blogs.slug, blogs.title, blogs.status, accounts.name, blogs.created_at, blogs.deleted_at
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id
ORDER BY blogs.created_at DESC, blogs.id DESC
LIMIT $1 OFFSET $2`

	ListAdminExcludeCountSQL = ListAdminCountSQL + `
WHERE blogs.deleted_at IS NULL`

	ListAdminExcludeSQL = `
SELECT blogs.id, blogs.owner_id, blogs.slug, blogs.title, blogs.status, accounts.name, blogs.created_at, blogs.deleted_at
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id
WHERE blogs.deleted_at IS NULL
ORDER BY blogs.created_at DESC, blogs.id DESC
LIMIT $1 OFFSET $2`

	ListAdminOnlyCountSQL = ListAdminCountSQL + `
WHERE blogs.deleted_at IS NOT NULL`

	ListAdminOnlySQL = `
SELECT blogs.id, blogs.owner_id, blogs.slug, blogs.title, blogs.status, accounts.name, blogs.created_at, blogs.deleted_at
FROM blogs
JOIN accounts ON accounts.id = blogs.owner_id
WHERE blogs.deleted_at IS NOT NULL
ORDER BY blogs.created_at DESC, blogs.id DESC
LIMIT $1 OFFSET $2`
)

type sqlTxKey struct{}

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type rowScanner interface {
	Scan(dest ...any) error
}

type resultRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type sqlResultRows struct {
	rows *sql.Rows
}

func (r sqlResultRows) Next() bool {
	return r.rows.Next()
}

func (r sqlResultRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r sqlResultRows) Err() error {
	return r.rows.Err()
}

func (r sqlResultRows) Close() {
	_ = r.rows.Close()
}

type Postgres struct {
	Pool *pgxpool.Pool
	DB   *sql.DB
}

func (r Postgres) querier(ctx context.Context) querier {
	if tx, ok := pgxtx.From(ctx); ok {
		return tx
	}
	return r.Pool
}

func withSQLTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, sqlTxKey{}, tx)
}

func sqlTxFrom(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(sqlTxKey{}).(*sql.Tx)
	return tx, ok
}

func (r Postgres) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if r.DB != nil {
		tx, err := r.DB.BeginTx(ctx, nil)
		if err != nil {
			return errs.New(errs.CodeInternalError, errs.KindInternal, err)
		}
		if err := fn(withSQLTx(ctx, tx)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return errs.New(errs.CodeInternalError, errs.KindInternal, err)
		}
		return nil
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	if err := fn(pgxtx.With(ctx, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}

func (r Postgres) Save(ctx context.Context, blog *aggregate.Blog) error {
	err := r.exec(
		ctx,
		SaveSQL,
		blog.ID(),
		blog.OwnerID(),
		blog.Slug().String(),
		blog.Title().String(),
		blog.Content().HTML(),
		string(blog.Status()),
		blog.DeletedAt(),
		blog.CreatedAt(),
		blog.UpdatedAt(),
	)
	if err != nil {
		return mapSaveError(err)
	}
	return nil
}

func (r Postgres) Get(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error) {
	q := GetSQL
	if _, ok := sqlTxFrom(ctx); ok {
		q = GetForUpdateSQL
	}
	if _, ok := pgxtx.From(ctx); ok {
		q = GetForUpdateSQL
	}
	blog, _, err := scanBlog(r.queryRow(ctx, q, id), false)
	return blog, err
}

func (r Postgres) GetPublicBySlug(ctx context.Context, slug string) (*aggregate.Blog, string, error) {
	return scanBlog(r.queryRow(ctx, GetPublicBySlugSQL, slug), true)
}

func (r Postgres) ListPublic(ctx context.Context, page, pageSize int) ([]repository.BlogSummary, int, error) {
	return r.list(ctx, ListPublicCountSQL, ListPublicSQL, page, pageSize)
}

func (r Postgres) ListAdmin(ctx context.Context, page, pageSize int, deletedFilter repository.DeletedFilter) ([]repository.BlogSummary, int, error) {
	countSQL, listSQL := adminListSQL(deletedFilter)
	return r.list(ctx, countSQL, listSQL, page, pageSize)
}

func adminListSQL(filter repository.DeletedFilter) (string, string) {
	switch filter {
	case repository.DeletedOnly:
		return ListAdminOnlyCountSQL, ListAdminOnlySQL
	case repository.DeletedAll:
		return ListAdminCountSQL, ListAdminSQL
	default:
		return ListAdminExcludeCountSQL, ListAdminExcludeSQL
	}
}

func (r Postgres) list(ctx context.Context, countSQL, listSQL string, page, pageSize int) ([]repository.BlogSummary, int, error) {
	var total int
	if err := r.queryRow(ctx, countSQL).Scan(&total); err != nil {
		return nil, 0, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * pageSize
	}
	rows, err := r.query(ctx, listSQL, pageSize, offset)
	if err != nil {
		return nil, 0, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	defer rows.Close()
	items := []repository.BlogSummary{}
	for rows.Next() {
		item, scanErr := scanSummary(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return items, total, nil
}

func (r Postgres) exec(ctx context.Context, q string, args ...any) error {
	if tx, ok := sqlTxFrom(ctx); ok {
		_, err := tx.ExecContext(ctx, q, args...)
		return err
	}
	if r.DB != nil {
		_, err := r.DB.ExecContext(ctx, q, args...)
		return err
	}
	_, err := r.querier(ctx).Exec(ctx, q, args...)
	return err
}

func (r Postgres) queryRow(ctx context.Context, q string, args ...any) rowScanner {
	if tx, ok := sqlTxFrom(ctx); ok {
		return tx.QueryRowContext(ctx, q, args...)
	}
	if r.DB != nil {
		return r.DB.QueryRowContext(ctx, q, args...)
	}
	return r.querier(ctx).QueryRow(ctx, q, args...)
}

func (r Postgres) query(ctx context.Context, q string, args ...any) (resultRows, error) {
	if tx, ok := sqlTxFrom(ctx); ok {
		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		return sqlResultRows{rows: rows}, nil
	}
	if r.DB != nil {
		rows, err := r.DB.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		return sqlResultRows{rows: rows}, nil
	}
	return r.querier(ctx).Query(ctx, q, args...)
}

func scanBlog(row rowScanner, withAuthor bool) (*aggregate.Blog, string, error) {
	var (
		id         uuid.UUID
		ownerID    uuid.UUID
		slug       string
		title      string
		content    string
		status     string
		deletedAt  sql.NullTime
		createdAt  time.Time
		updatedAt  time.Time
		authorName string
	)
	dest := []any{&id, &ownerID, &slug, &title, &content, &status, &deletedAt, &createdAt, &updatedAt}
	if withAuthor {
		dest = append(dest, &authorName)
	}
	if err := row.Scan(dest...); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, "", errBlogNotFound
		}
		return nil, "", errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	parsed, ok := enum.ParsePublicationStatus(status)
	if !ok {
		return nil, "", errs.New(errs.CodeInternalError, errs.KindInternal, fmt.Errorf("invalid stored status %q", status))
	}
	blog := aggregate.RestoreBlog(
		id,
		ownerID,
		aggregate.RestoreBlogTitle(title),
		aggregate.RestoreBlogContent(content),
		aggregate.RestoreBlogSlug(slug),
		parsed,
		nullableTime(deletedAt),
		createdAt,
		updatedAt,
	)
	return blog, authorName, nil
}

func scanSummary(row rowScanner) (repository.BlogSummary, error) {
	var (
		item      repository.BlogSummary
		status    string
		deletedAt sql.NullTime
	)
	if err := row.Scan(&item.ID, &item.OwnerID, &item.Slug, &item.Title, &status, &item.AuthorName, &item.CreatedAt, &deletedAt); err != nil {
		return repository.BlogSummary{}, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	parsed, ok := enum.ParsePublicationStatus(status)
	if !ok {
		return repository.BlogSummary{}, errs.New(errs.CodeInternalError, errs.KindInternal, fmt.Errorf("invalid stored status %q", status))
	}
	item.Status = parsed
	item.DeletedAt = nullableTime(deletedAt)
	return item, nil
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	stamp := value.Time
	return &stamp
}

func mapSaveError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "blogs_slug_uidx" {
		return repository.ErrUniqueSlug
	}
	return errs.New(errs.CodeInternalError, errs.KindInternal, err)
}
