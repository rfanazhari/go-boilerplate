package blogrepository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
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

type Postgres struct {
	Pool *pgxpool.Pool
	DB   *sql.DB
}

func (r Postgres) Save(context.Context, *aggregate.Blog) error {
	return nil
}

func (r Postgres) Get(context.Context, uuid.UUID) (*aggregate.Blog, error) {
	return nil, nil
}

func (r Postgres) GetPublicBySlug(context.Context, string) (*aggregate.Blog, string, error) {
	return nil, "", nil
}

func (r Postgres) ListPublic(context.Context, int, int) ([]repository.BlogSummary, int, error) {
	return nil, 0, nil
}

func (r Postgres) ListAdmin(context.Context, int, int, repository.DeletedFilter) ([]repository.BlogSummary, int, error) {
	return nil, 0, nil
}

func (r Postgres) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
