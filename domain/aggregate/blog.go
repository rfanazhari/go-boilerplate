package aggregate

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var (
	errBlogDeleted = errors.New("a deleted blog cannot be changed")
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
	if ownerID == uuid.Nil {
		return nil, errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, errMissingOwner)
	}
	stamp := normalizeInstant(now)
	return &Blog{
		id:        id,
		ownerID:   ownerID,
		title:     title,
		content:   content,
		slug:      slug,
		status:    enum.PublicationDraft,
		createdAt: stamp,
		updatedAt: stamp,
	}, nil
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

func (b *Blog) ID() uuid.UUID {
	return b.id
}

func (b *Blog) OwnerID() uuid.UUID {
	return b.ownerID
}

func (b *Blog) Title() BlogTitle {
	return b.title
}

func (b *Blog) Content() BlogContent {
	return b.content
}

func (b *Blog) Slug() BlogSlug {
	return b.slug
}

func (b *Blog) Status() enum.PublicationStatus {
	return b.status
}

func (b *Blog) DeletedAt() *time.Time {
	return cloneTime(b.deletedAt)
}

func (b *Blog) CreatedAt() time.Time {
	return b.createdAt
}

func (b *Blog) UpdatedAt() time.Time {
	return b.updatedAt
}

func (b *Blog) IsSoftDeleted() bool {
	return b.deletedAt != nil
}

func (b *Blog) Publish(now time.Time) error {
	if b.deletedAt != nil {
		return errs.New(errs.CodeBlogDeleted, errs.KindConflict, errBlogDeleted)
	}
	if b.status == enum.PublicationPublished {
		return nil
	}
	if err := validateContentLength(b.content.PlainTextLength(), enum.PublicationPublished); err != nil {
		return err
	}
	b.status = enum.PublicationPublished
	b.updatedAt = normalizeInstant(now)
	return nil
}

func (b *Blog) Unpublish(now time.Time) error {
	if b.deletedAt != nil {
		return errs.New(errs.CodeBlogDeleted, errs.KindConflict, errBlogDeleted)
	}
	if b.status == enum.PublicationDraft {
		return nil
	}
	b.status = enum.PublicationDraft
	b.updatedAt = normalizeInstant(now)
	return nil
}

func (b *Blog) Revise(title BlogTitle, content BlogContent, now time.Time) error {
	if b.deletedAt != nil {
		return errs.New(errs.CodeBlogDeleted, errs.KindConflict, errBlogDeleted)
	}
	if err := validateContentLength(content.PlainTextLength(), b.status); err != nil {
		return err
	}
	b.title = title
	b.content = content
	b.updatedAt = normalizeInstant(now)
	return nil
}

func (b *Blog) SoftDelete(now time.Time) error {
	if b.deletedAt != nil {
		return errs.New(errs.CodeBlogDeleted, errs.KindConflict, errBlogDeleted)
	}
	stamp := normalizeInstant(now)
	b.deletedAt = &stamp
	b.updatedAt = stamp
	return nil
}
