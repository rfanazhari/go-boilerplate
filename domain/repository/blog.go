package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

type BlogDeletionFilter string

const (
	BlogActive  BlogDeletionFilter = "active"
	BlogDeleted BlogDeletionFilter = "deleted"
	BlogAll     BlogDeletionFilter = "all"
)

// BlogPageQuery uses validated, one-based pages and sizes from 1 through 100.
// Callers resolve defaults (1, 20) and reject offset overflow before querying.
type BlogPageQuery struct {
	Page     int
	PageSize int
}

type BlogAdminQuery struct {
	Page     int
	PageSize int
	Filter   BlogDeletionFilter
}

// Read projections contain current joined author names, never account secrets.
// List projections intentionally omit full content.
type BlogPublicSummary struct {
	ID         uuid.UUID
	Slug       string
	Title      string
	AuthorName string
	CreatedAt  time.Time
}

type BlogPublicDetail struct {
	ID         uuid.UUID
	Slug       string
	Title      string
	Content    string
	AuthorName string
	CreatedAt  time.Time
}

type BlogAdminSummary struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	Slug       string
	Title      string
	AuthorName string
	Status     enum.BlogStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type BlogPublicPage struct {
	Items    []BlogPublicSummary
	Page     int
	PageSize int
	Total    int64
}

type BlogAdminPage struct {
	Items    []BlogAdminSummary
	Page     int
	PageSize int
	Total    int64
}

// BlogRepository separates insertion from updates and makes transaction
// ownership explicit. No method may mutate the supplied aggregate.
type BlogRepository interface {
	// Create inserts only; known ID/slug collisions use BLOG_IDENTIFIER_CONFLICT.
	Create(ctx context.Context, blog *aggregate.Blog) error
	// GetByID includes deleted records for application disclosure policy.
	GetByID(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error)
	// GetForUpdate requires a transaction context and locks until its completion.
	GetForUpdate(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error)
	// Update never upserts or changes ID, owner, slug, or createdAt.
	Update(ctx context.Context, blog *aggregate.Blog) error
	// RunInTransaction commits only after callback success and returns commit
	// errors. Errors/panics roll back; callback operations use the passed context.
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	// Public queries require published, non-deleted blogs with active,
	// non-deleted owners. Hidden and missing detail both use BLOG_NOT_FOUND.
	GetPublicBySlug(ctx context.Context, slug string) (*BlogPublicDetail, error)
	// Lists filter before counting/paging and sort createdAt DESC, ID DESC.
	// Total and items share a snapshot, including empty/out-of-range pages.
	ListPublic(ctx context.Context, query BlogPageQuery) (BlogPublicPage, error)
	// Admin reads include all publication/owner states, applying only the
	// requested deletion filter. Current author names come from the same join.
	ListAdmin(ctx context.Context, query BlogAdminQuery) (BlogAdminPage, error)
}
