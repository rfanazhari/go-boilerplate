package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

const (
	defaultBlogPage     = 1
	defaultBlogPageSize = 20
	maxBlogPage         = 1_000_000
	maxBlogPageSize     = 100
	maxSlugAttempts     = 5
)

type blogOp int

const (
	opGet blogOp = iota
	opEdit
	opPublish
	opUnpublish
	opDelete
)

type BlogUseCase struct {
	Blogs     repository.BlogRepository
	Accounts  repository.AccountRepository
	Sanitizer service.HTMLSanitizer
	Clock     service.Clock
	IDs       func() uuid.UUID
	Suffix    func() string
}

func (u BlogUseCase) now() time.Time {
	if u.Clock == nil {
		return time.Now().UTC().Truncate(time.Millisecond)
	}
	return u.Clock.Now().UTC().Truncate(time.Millisecond)
}

func (u BlogUseCase) nextID() uuid.UUID {
	if u.IDs == nil {
		return uuid.New()
	}
	return u.IDs()
}

func (u BlogUseCase) nextSuffix() (string, error) {
	if u.Suffix != nil {
		return u.Suffix(), nil
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return hex.EncodeToString(b[:]), nil
}

func blogNotFoundErr() error {
	return errs.New(errs.CodeBlogNotFound, errs.KindNotFound, errors.New("blog was not found"))
}

func blogForbiddenErr() error {
	return errs.New(errs.CodeForbidden, errs.KindForbidden, errors.New("forbidden"))
}

func blogValidationErr() error {
	return errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, errors.New("validation failed"))
}

func mapBlogGetError(err error) error {
	var appErr *errs.Error
	if errors.As(err, &appErr) {
		return err
	}
	return wrapInfra(err)
}

func (u BlogUseCase) Create(ctx context.Context, actor dto.BlogActor, input dto.CreateBlogInput) (dto.BlogView, error) {
	if actor.Role != enum.RoleUser {
		return dto.BlogView{}, blogForbiddenErr()
	}
	title, err := aggregate.NewBlogTitle(input.Title)
	if err != nil {
		return dto.BlogView{}, err
	}
	content, err := aggregate.NewBlogContent(u.Sanitizer, input.Content, enum.PublicationDraft)
	if err != nil {
		return dto.BlogView{}, err
	}
	now := u.now()
	id := u.nextID()
	var last error
	for range maxSlugAttempts {
		suffix, err := u.nextSuffix()
		if err != nil {
			return dto.BlogView{}, err
		}
		slug, err := aggregate.NewBlogSlug(title.String(), suffix)
		if err != nil {
			return dto.BlogView{}, err
		}
		blog, err := aggregate.NewBlog(id, actor.ID, title, content, slug, now)
		if err != nil {
			return dto.BlogView{}, err
		}
		if err := u.Blogs.Save(ctx, blog); err != nil {
			if errors.Is(err, repository.ErrUniqueSlug) {
				last = err
				continue
			}
			return dto.BlogView{}, wrapInfra(err)
		}
		return u.managedView(ctx, blog)
	}
	if last == nil {
		last = errors.New("slug retries exhausted")
	}
	return dto.BlogView{}, errs.New(errs.CodeInternalError, errs.KindInternal, last)
}

func (u BlogUseCase) Update(ctx context.Context, actor dto.BlogActor, input dto.UpdateBlogInput) (dto.BlogView, error) {
	return u.mutate(ctx, actor, input.ID, opEdit, func(blog *aggregate.Blog, now time.Time) (bool, error) {
		title, err := aggregate.NewBlogTitle(input.Title)
		if err != nil {
			return false, err
		}
		content, err := aggregate.NewBlogContent(u.Sanitizer, input.Content, blog.Status())
		if err != nil {
			return false, err
		}
		if err := blog.Revise(title, content, now); err != nil {
			return false, err
		}
		return true, nil
	})
}

func (u BlogUseCase) Publish(ctx context.Context, actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
	return u.mutate(ctx, actor, id, opPublish, func(blog *aggregate.Blog, now time.Time) (bool, error) {
		if blog.Status() == enum.PublicationPublished {
			return false, nil
		}
		if err := blog.Publish(now); err != nil {
			return false, err
		}
		return true, nil
	})
}

func (u BlogUseCase) Unpublish(ctx context.Context, actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
	return u.mutate(ctx, actor, id, opUnpublish, func(blog *aggregate.Blog, now time.Time) (bool, error) {
		if blog.Status() == enum.PublicationDraft {
			return false, nil
		}
		if err := blog.Unpublish(now); err != nil {
			return false, err
		}
		return true, nil
	})
}

func (u BlogUseCase) Delete(ctx context.Context, actor dto.BlogActor, id uuid.UUID) error {
	return u.Blogs.RunInTransaction(ctx, func(txCtx context.Context) error {
		blog, err := u.Blogs.Get(txCtx, id)
		if err != nil {
			return mapBlogGetError(err)
		}
		if err := u.authorize(txCtx, actor, blog, opDelete); err != nil {
			return err
		}
		if blog.IsSoftDeleted() {
			return nil
		}
		if err := blog.SoftDelete(u.now()); err != nil {
			return err
		}
		if err := u.Blogs.Save(txCtx, blog); err != nil {
			return wrapInfra(err)
		}
		return nil
	})
}

func (u BlogUseCase) Get(ctx context.Context, actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
	blog, err := u.Blogs.Get(ctx, id)
	if err != nil {
		return dto.BlogView{}, mapBlogGetError(err)
	}
	if err := u.authorize(ctx, actor, blog, opGet); err != nil {
		return dto.BlogView{}, err
	}
	return u.managedView(ctx, blog)
}

func (u BlogUseCase) GetPublic(ctx context.Context, slug string) (dto.PublicBlogView, error) {
	blog, authorName, err := u.Blogs.GetPublicBySlug(ctx, slug)
	if err != nil {
		return dto.PublicBlogView{}, mapBlogGetError(err)
	}
	return dto.PublicBlogView{
		ID:         blog.ID(),
		Slug:       blog.Slug().String(),
		Title:      blog.Title().String(),
		Content:    blog.Content().HTML(),
		Status:     string(blog.Status()),
		AuthorName: authorName,
		CreatedAt:  blog.CreatedAt(),
	}, nil
}

func (u BlogUseCase) ListPublic(ctx context.Context, input dto.ListBlogsInput) (dto.PublicBlogPage, error) {
	page, pageSize, err := normalizeBlogPage(input.Page, input.PageSize)
	if err != nil {
		return dto.PublicBlogPage{}, err
	}
	items, total, err := u.Blogs.ListPublic(ctx, page, pageSize)
	if err != nil {
		return dto.PublicBlogPage{}, wrapInfra(err)
	}
	out := make([]dto.PublicBlogItem, 0, len(items))
	for _, item := range items {
		out = append(out, dto.PublicBlogItem{
			ID:         item.ID,
			Slug:       item.Slug,
			Title:      item.Title,
			AuthorName: item.AuthorName,
			CreatedAt:  item.CreatedAt,
		})
	}
	return dto.PublicBlogPage{Items: out, Page: page, PageSize: pageSize, Total: total}, nil
}

func (u BlogUseCase) ListAdmin(ctx context.Context, actor dto.BlogActor, input dto.ListAdminBlogsInput) (dto.AdminBlogPage, error) {
	if actor.Role != enum.RoleAdmin {
		return dto.AdminBlogPage{}, blogForbiddenErr()
	}
	page, pageSize, err := normalizeBlogPage(input.Page, input.PageSize)
	if err != nil {
		return dto.AdminBlogPage{}, err
	}
	filter, err := parseDeletedFilter(input.Deleted)
	if err != nil {
		return dto.AdminBlogPage{}, err
	}
	items, total, err := u.Blogs.ListAdmin(ctx, page, pageSize, filter)
	if err != nil {
		return dto.AdminBlogPage{}, wrapInfra(err)
	}
	out := make([]dto.AdminBlogItem, 0, len(items))
	for _, item := range items {
		out = append(out, dto.AdminBlogItem{
			ID:         item.ID,
			OwnerID:    item.OwnerID,
			Slug:       item.Slug,
			Title:      item.Title,
			Status:     string(item.Status),
			AuthorName: item.AuthorName,
			CreatedAt:  item.CreatedAt,
			DeletedAt:  item.DeletedAt,
		})
	}
	return dto.AdminBlogPage{Items: out, Page: page, PageSize: pageSize, Total: total}, nil
}

func (u BlogUseCase) mutate(
	ctx context.Context,
	actor dto.BlogActor,
	id uuid.UUID,
	op blogOp,
	fn func(*aggregate.Blog, time.Time) (bool, error),
) (dto.BlogView, error) {
	var result dto.BlogView
	err := u.Blogs.RunInTransaction(ctx, func(txCtx context.Context) error {
		blog, err := u.Blogs.Get(txCtx, id)
		if err != nil {
			return mapBlogGetError(err)
		}
		if err := u.authorize(txCtx, actor, blog, op); err != nil {
			return err
		}
		changed, err := fn(blog, u.now())
		if err != nil {
			return err
		}
		if changed {
			if err := u.Blogs.Save(txCtx, blog); err != nil {
				return wrapInfra(err)
			}
		}
		view, err := u.managedView(txCtx, blog)
		if err != nil {
			return err
		}
		result = view
		return nil
	})
	if err != nil {
		return dto.BlogView{}, err
	}
	return result, nil
}

func (u BlogUseCase) authorize(ctx context.Context, actor dto.BlogActor, blog *aggregate.Blog, op blogOp) error {
	visible, err := u.visible(ctx, actor, blog)
	if err != nil {
		return err
	}
	if !visible {
		return blogNotFoundErr()
	}
	isOwner := actor.ID == blog.OwnerID()
	isAdmin := actor.Role == enum.RoleAdmin
	if blog.IsSoftDeleted() {
		switch op {
		case opDelete:
			if isAdmin {
				return nil
			}
			return blogNotFoundErr()
		case opGet:
			if isAdmin {
				return nil
			}
			return blogNotFoundErr()
		default:
			if isAdmin {
				return errs.New(errs.CodeBlogDeleted, errs.KindConflict, errors.New("a deleted blog cannot be changed"))
			}
			return blogNotFoundErr()
		}
	}
	switch op {
	case opGet:
		if isOwner || isAdmin {
			return nil
		}
		return blogForbiddenErr()
	case opEdit, opPublish:
		if isOwner {
			return nil
		}
		return blogForbiddenErr()
	case opUnpublish, opDelete:
		if isOwner || isAdmin {
			return nil
		}
		return blogForbiddenErr()
	default:
		return blogForbiddenErr()
	}
}

func (u BlogUseCase) visible(ctx context.Context, actor dto.BlogActor, blog *aggregate.Blog) (bool, error) {
	if actor.Role == enum.RoleAdmin {
		return true, nil
	}
	if actor.ID == blog.OwnerID() && !blog.IsSoftDeleted() {
		return true, nil
	}
	if blog.IsSoftDeleted() || blog.Status() != enum.PublicationPublished {
		return false, nil
	}
	account, err := u.Accounts.GetByID(ctx, blog.OwnerID())
	if err != nil {
		var appErr *errs.Error
		if errors.As(err, &appErr) && appErr.Code == errs.CodeAccountNotFound {
			return false, nil
		}
		return false, wrapInfra(err)
	}
	if account.Status() != enum.AccountActive || account.IsSoftDeleted() {
		return false, nil
	}
	return true, nil
}

func (u BlogUseCase) managedView(ctx context.Context, blog *aggregate.Blog) (dto.BlogView, error) {
	account, err := u.Accounts.GetByID(ctx, blog.OwnerID())
	authorName := ""
	if err != nil {
		var appErr *errs.Error
		if !errors.As(err, &appErr) || appErr.Code != errs.CodeAccountNotFound {
			return dto.BlogView{}, wrapInfra(err)
		}
	} else {
		authorName = account.Name()
	}
	return dto.BlogView{
		ID:         blog.ID(),
		OwnerID:    blog.OwnerID(),
		Slug:       blog.Slug().String(),
		Title:      blog.Title().String(),
		Content:    blog.Content().HTML(),
		Status:     string(blog.Status()),
		AuthorName: authorName,
		CreatedAt:  blog.CreatedAt(),
		UpdatedAt:  blog.UpdatedAt(),
		DeletedAt:  blog.DeletedAt(),
	}, nil
}

func normalizeBlogPage(page, pageSize *int) (int, int, error) {
	p := defaultBlogPage
	s := defaultBlogPageSize
	if page != nil {
		p = *page
	}
	if pageSize != nil {
		s = *pageSize
	}
	if p < 1 || p > maxBlogPage || s < 1 || s > maxBlogPageSize {
		return 0, 0, blogValidationErr()
	}
	if p > 1 && s > math.MaxInt/(p-1) {
		return 0, 0, blogValidationErr()
	}
	return p, s, nil
}

func parseDeletedFilter(value string) (repository.DeletedFilter, error) {
	switch value {
	case "", string(repository.DeletedExclude):
		return repository.DeletedExclude, nil
	case string(repository.DeletedOnly):
		return repository.DeletedOnly, nil
	case string(repository.DeletedAll):
		return repository.DeletedAll, nil
	default:
		return "", blogValidationErr()
	}
}
