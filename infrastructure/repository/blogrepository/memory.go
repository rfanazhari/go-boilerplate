package blogrepository

import (
	"bytes"
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type memoryLockedKey struct{}

type Memory struct {
	Accounts repository.AccountRepository
	mu       sync.Mutex
	blogs    map[uuid.UUID]*aggregate.Blog
}

func (m *Memory) ensure() {
	if m.blogs == nil {
		m.blogs = map[uuid.UUID]*aggregate.Blog{}
	}
}

func (m *Memory) lock(ctx context.Context) func() {
	if locked, _ := ctx.Value(memoryLockedKey{}).(bool); locked {
		return func() {}
	}
	m.mu.Lock()
	return m.mu.Unlock
}

func cloneBlog(blog *aggregate.Blog) *aggregate.Blog {
	return aggregate.RestoreBlog(
		blog.ID(),
		blog.OwnerID(),
		aggregate.RestoreBlogTitle(blog.Title().String()),
		aggregate.RestoreBlogContent(blog.Content().HTML()),
		aggregate.RestoreBlogSlug(blog.Slug().String()),
		blog.Status(),
		blog.DeletedAt(),
		blog.CreatedAt(),
		blog.UpdatedAt(),
	)
}

func (m *Memory) Save(ctx context.Context, blog *aggregate.Blog) error {
	unlock := m.lock(ctx)
	defer unlock()
	m.ensure()
	slug := blog.Slug().String()
	for _, existing := range m.blogs {
		if existing.ID() != blog.ID() && existing.Slug().String() == slug {
			return repository.ErrUniqueSlug
		}
	}
	m.blogs[blog.ID()] = cloneBlog(blog)
	return nil
}

func (m *Memory) Get(ctx context.Context, id uuid.UUID) (*aggregate.Blog, error) {
	unlock := m.lock(ctx)
	defer unlock()
	m.ensure()
	blog, ok := m.blogs[id]
	if !ok {
		return nil, errBlogNotFound
	}
	return cloneBlog(blog), nil
}

func (m *Memory) GetPublicBySlug(ctx context.Context, slug string) (*aggregate.Blog, string, error) {
	unlock := m.lock(ctx)
	defer unlock()
	m.ensure()
	for _, blog := range m.blogs {
		if blog.Slug().String() != slug {
			continue
		}
		account, authorName, ok := m.lookupAuthor(ctx, blog.OwnerID())
		if !ok || !isPubliclyEligible(blog, account) {
			break
		}
		return cloneBlog(blog), authorName, nil
	}
	return nil, "", errs.New(errs.CodeBlogNotFound, errs.KindNotFound, errBlogNotFound)
}

func (m *Memory) ListPublic(ctx context.Context, page, pageSize int) ([]repository.BlogSummary, int, error) {
	unlock := m.lock(ctx)
	defer unlock()
	m.ensure()
	var eligible []repository.BlogSummary
	for _, blog := range m.blogs {
		account, authorName, ok := m.lookupAuthor(ctx, blog.OwnerID())
		if !ok || !isPubliclyEligible(blog, account) {
			continue
		}
		eligible = append(eligible, toSummary(blog, authorName))
	}
	sortSummaries(eligible)
	return pageSummaries(eligible, page, pageSize), len(eligible), nil
}

func (m *Memory) ListAdmin(ctx context.Context, page, pageSize int, deletedFilter repository.DeletedFilter) ([]repository.BlogSummary, int, error) {
	unlock := m.lock(ctx)
	defer unlock()
	m.ensure()
	var items []repository.BlogSummary
	for _, blog := range m.blogs {
		if !matchesDeletedFilter(blog, deletedFilter) {
			continue
		}
		_, authorName, _ := m.lookupAuthor(ctx, blog.OwnerID())
		items = append(items, toSummary(blog, authorName))
	}
	sortSummaries(items)
	return pageSummaries(items, page, pageSize), len(items), nil
}

func (m *Memory) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(context.WithValue(ctx, memoryLockedKey{}, true))
}

func (m *Memory) lookupAuthor(ctx context.Context, ownerID uuid.UUID) (*aggregate.Account, string, bool) {
	if m.Accounts == nil {
		return nil, "", false
	}
	account, err := m.Accounts.GetByID(ctx, ownerID)
	if err != nil || account == nil {
		return nil, "", false
	}
	return account, account.Name(), true
}

func isPubliclyEligible(blog *aggregate.Blog, account *aggregate.Account) bool {
	return blog.Status() == enum.PublicationPublished &&
		!blog.IsSoftDeleted() &&
		account.Status() == enum.AccountActive &&
		!account.IsSoftDeleted()
}

func matchesDeletedFilter(blog *aggregate.Blog, filter repository.DeletedFilter) bool {
	switch filter {
	case repository.DeletedOnly:
		return blog.IsSoftDeleted()
	case repository.DeletedAll:
		return true
	default:
		return !blog.IsSoftDeleted()
	}
}

func toSummary(blog *aggregate.Blog, authorName string) repository.BlogSummary {
	return repository.BlogSummary{
		ID:         blog.ID(),
		OwnerID:    blog.OwnerID(),
		Slug:       blog.Slug().String(),
		Title:      blog.Title().String(),
		Status:     blog.Status(),
		AuthorName: authorName,
		CreatedAt:  blog.CreatedAt(),
		DeletedAt:  blog.DeletedAt(),
	}
}

func sortSummaries(items []repository.BlogSummary) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return bytes.Compare(items[i].ID[:], items[j].ID[:]) > 0
	})
}

func pageSummaries(items []repository.BlogSummary, page, pageSize int) []repository.BlogSummary {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		return []repository.BlogSummary{}
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []repository.BlogSummary{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

var errBlogNotFound = errs.New(errs.CodeBlogNotFound, errs.KindNotFound, nil)
