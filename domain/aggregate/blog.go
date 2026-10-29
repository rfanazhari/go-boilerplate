package aggregate

import (
	"github.com/google/uuid"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"time"
)

type Blog struct{}

func NewBlog(id, ownerID uuid.UUID, title, content string, now time.Time) (*Blog, error) {
	return &Blog{}, nil
}
func RestoreBlog(id, ownerID uuid.UUID, slug, title, content string, status enum.BlogStatus, createdAt, updatedAt time.Time, deletedAt *time.Time) *Blog {
	return &Blog{}
}
func (b *Blog) ID() uuid.UUID                                    { return uuid.Nil }
func (b *Blog) OwnerID() uuid.UUID                               { return uuid.Nil }
func (b *Blog) Slug() string                                     { return "" }
func (b *Blog) Title() string                                    { return "" }
func (b *Blog) Content() string                                  { return "" }
func (b *Blog) Status() enum.BlogStatus                          { return "" }
func (b *Blog) CreatedAt() time.Time                             { return time.Time{} }
func (b *Blog) UpdatedAt() time.Time                             { return time.Time{} }
func (b *Blog) DeletedAt() *time.Time                            { return nil }
func (b *Blog) Edit(title, content *string, now time.Time) error { return nil }
func (b *Blog) Publish(now time.Time) error                      { return nil }
func (b *Blog) Unpublish(now time.Time) error                    { return nil }
func (b *Blog) SoftDelete(now time.Time) error                   { return nil }
