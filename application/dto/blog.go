package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

type BlogActor struct {
	ID   uuid.UUID
	Role enum.Role
}

type CreateBlogInput struct {
	Title   string
	Content string
}

type UpdateBlogInput struct {
	ID      uuid.UUID
	Title   string
	Content string
}

type ListBlogsInput struct {
	Page     *int
	PageSize *int
}

type ListAdminBlogsInput struct {
	Page     *int
	PageSize *int
	Deleted  string
}

type BlogView struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	Slug       string
	Title      string
	Content    string
	Status     string
	AuthorName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type PublicBlogView struct {
	ID         uuid.UUID
	Slug       string
	Title      string
	Content    string
	Status     string
	AuthorName string
	CreatedAt  time.Time
}

type PublicBlogItem struct {
	ID         uuid.UUID
	Slug       string
	Title      string
	AuthorName string
	CreatedAt  time.Time
}

type AdminBlogItem struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	Slug       string
	Title      string
	Status     string
	AuthorName string
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

type PublicBlogPage struct {
	Items    []PublicBlogItem
	Page     int
	PageSize int
	Total    int
}

type AdminBlogPage struct {
	Items    []AdminBlogItem
	Page     int
	PageSize int
	Total    int
}
