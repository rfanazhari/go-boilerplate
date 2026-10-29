package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

var ErrUniqueSlug = errors.New("blog slug already exists")

type DeletedFilter string

const (
	DeletedExclude DeletedFilter = "exclude"
	DeletedOnly    DeletedFilter = "only"
	DeletedAll     DeletedFilter = "all"
)

type BlogSummary struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	Slug       string
	Title      string
	Status     enum.PublicationStatus
	AuthorName string
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

type BlogRepository interface {
	Save(ctx context.Context, blog *aggregate.Blog) error
	Get(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error)
	GetPublicBySlug(ctx context.Context, slug string) (*aggregate.Blog, string, error)
	ListPublic(ctx context.Context, page, pageSize int) ([]BlogSummary, int, error)
	ListAdmin(ctx context.Context, page, pageSize int, deletedFilter DeletedFilter) ([]BlogSummary, int, error)
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
