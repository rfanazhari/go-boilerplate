package blogrepository

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
)

type Memory struct {
	Accounts repository.AccountRepository
	mu       sync.Mutex
	blogs    map[uuid.UUID]*aggregate.Blog
}

func (m *Memory) Save(context.Context, *aggregate.Blog) error {
	return nil
}

func (m *Memory) Get(context.Context, uuid.UUID) (*aggregate.Blog, error) {
	return nil, nil
}

func (m *Memory) GetPublicBySlug(context.Context, string) (*aggregate.Blog, string, error) {
	return nil, "", nil
}

func (m *Memory) ListPublic(context.Context, int, int) ([]repository.BlogSummary, int, error) {
	return nil, 0, nil
}

func (m *Memory) ListAdmin(context.Context, int, int, repository.DeletedFilter) ([]repository.BlogSummary, int, error) {
	return nil, 0, nil
}

func (m *Memory) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
