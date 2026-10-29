package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
)

type BlogUseCase struct {
	Blogs     repository.BlogRepository
	Accounts  repository.AccountRepository
	Sanitizer service.HTMLSanitizer
	Clock     service.Clock
	IDs       func() uuid.UUID
	Suffix    func() string
}

func (u BlogUseCase) Create(ctx context.Context, actor dto.BlogActor, input dto.CreateBlogInput) (dto.BlogView, error) {
	return dto.BlogView{}, nil
}

func (u BlogUseCase) Update(ctx context.Context, actor dto.BlogActor, input dto.UpdateBlogInput) (dto.BlogView, error) {
	return dto.BlogView{}, nil
}

func (u BlogUseCase) Publish(ctx context.Context, actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
	return dto.BlogView{}, nil
}

func (u BlogUseCase) Unpublish(ctx context.Context, actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
	return dto.BlogView{}, nil
}

func (u BlogUseCase) Delete(ctx context.Context, actor dto.BlogActor, id uuid.UUID) error {
	return nil
}

func (u BlogUseCase) Get(ctx context.Context, actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
	return dto.BlogView{}, nil
}

func (u BlogUseCase) GetPublic(ctx context.Context, slug string) (dto.PublicBlogView, error) {
	return dto.PublicBlogView{}, nil
}

func (u BlogUseCase) ListPublic(ctx context.Context, input dto.ListBlogsInput) (dto.PublicBlogPage, error) {
	return dto.PublicBlogPage{}, nil
}

func (u BlogUseCase) ListAdmin(ctx context.Context, actor dto.BlogActor, input dto.ListAdminBlogsInput) (dto.AdminBlogPage, error) {
	return dto.AdminBlogPage{}, nil
}
