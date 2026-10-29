package aggregate

import (
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

type Blog struct {
	id        uuid.UUID
	ownerID   uuid.UUID
	title     BlogTitle
	content   BlogContent
	slug      BlogSlug
	status    enum.PublicationStatus
	deletedAt *time.Time
	createdAt time.Time
	updatedAt time.Time
}

func NewBlog(id, ownerID uuid.UUID, title BlogTitle, content BlogContent, slug BlogSlug, now time.Time) (*Blog, error) {
	return &Blog{}, nil
}

func RestoreBlog(
	id, ownerID uuid.UUID,
	title BlogTitle,
	content BlogContent,
	slug BlogSlug,
	status enum.PublicationStatus,
	deletedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Blog {
	return &Blog{
		id:        id,
		ownerID:   ownerID,
		title:     title,
		content:   content,
		slug:      slug,
		status:    status,
		deletedAt: cloneTime(deletedAt),
		createdAt: normalizeInstant(createdAt),
		updatedAt: normalizeInstant(updatedAt),
	}
}

func (b *Blog) ID() uuid.UUID                  { return b.id }
func (b *Blog) OwnerID() uuid.UUID             { return b.ownerID }
func (b *Blog) Title() BlogTitle               { return b.title }
func (b *Blog) Content() BlogContent           { return b.content }
func (b *Blog) Slug() BlogSlug                 { return b.slug }
func (b *Blog) Status() enum.PublicationStatus { return b.status }
func (b *Blog) DeletedAt() *time.Time          { return cloneTime(b.deletedAt) }
func (b *Blog) CreatedAt() time.Time           { return b.createdAt }
func (b *Blog) UpdatedAt() time.Time           { return b.updatedAt }
func (b *Blog) IsSoftDeleted() bool            { return b.deletedAt != nil }
func (b *Blog) Publish(now time.Time) error    { return nil }
func (b *Blog) Unpublish(now time.Time) error  { return nil }
func (b *Blog) Revise(title BlogTitle, content BlogContent, now time.Time) error {
	return nil
}
func (b *Blog) SoftDelete(now time.Time) error { return nil }
