package aggregate

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

// Blog owns writing and publication state. Access policy is evaluated separately
// against the current actor before invoking even an idempotent operation.
type Blog struct {
	id        uuid.UUID
	ownerID   uuid.UUID
	slug      string
	title     string
	content   string
	status    enum.BlogStatus
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

func validateBlogCandidate(title, content string, status enum.BlogStatus) (string, string, error) {
	title = strings.TrimSpace(title)
	length := utf8.RuneCountInString(title)
	if !utf8.ValidString(title) || length < 10 || length > 200 {
		return "", "", errs.New(errs.CodeInvalidBlogTitle, errs.KindInvalidInput, nil)
	}
	clean, err := service.ProcessBlogContent(content)
	if err != nil {
		return "", "", err
	}
	if clean.TextLength > 1000 || (status == enum.BlogPublished && clean.TextLength < 100) {
		return "", "", errs.New(errs.CodeInvalidBlogContent, errs.KindInvalidInput, nil)
	}
	return title, clean.HTML, nil
}

func (b *Blog) requireMutable() error {
	if b.deletedAt != nil {
		return errs.New(errs.CodeBlogDeleted, errs.KindConflict, nil)
	}
	if _, ok := enum.ParseBlogStatus(string(b.status)); !ok {
		return errs.New(errs.CodeInvalidTransition, errs.KindConflict, nil)
	}
	return nil
}

// NewBlog creates a draft and derives its permanent slug from its server ID.
func NewBlog(id, ownerID uuid.UUID, title, content string, now time.Time) (*Blog, error) {
	if id == uuid.Nil || ownerID == uuid.Nil {
		return nil, errs.New(errs.CodeInvalidIdentifier, errs.KindInvalidInput, nil)
	}
	title, content, err := validateBlogCandidate(title, content, enum.BlogDraft)
	if err != nil {
		return nil, err
	}
	stamp := normalizeInstant(now)
	return &Blog{
		id: id, ownerID: ownerID, slug: service.BlogSlug(title, id),
		title: title, content: content, status: enum.BlogDraft,
		createdAt: stamp, updatedAt: stamp,
	}, nil
}

// RestoreBlog rehydrates trusted persisted state without regenerating its slug.
// Mutation paths revalidate candidates; restoration never changes stored content.
func RestoreBlog(id, ownerID uuid.UUID, slug, title, content string, status enum.BlogStatus, createdAt, updatedAt time.Time, deletedAt *time.Time) *Blog {
	return &Blog{
		id: id, ownerID: ownerID, slug: slug, title: title, content: content, status: status,
		createdAt: normalizeInstant(createdAt), updatedAt: normalizeInstant(updatedAt), deletedAt: cloneTime(deletedAt),
	}
}

func (b *Blog) ID() uuid.UUID           { return b.id }
func (b *Blog) OwnerID() uuid.UUID      { return b.ownerID }
func (b *Blog) Slug() string            { return b.slug }
func (b *Blog) Title() string           { return b.title }
func (b *Blog) Content() string         { return b.content }
func (b *Blog) Status() enum.BlogStatus { return b.status }
func (b *Blog) CreatedAt() time.Time    { return b.createdAt }
func (b *Blog) UpdatedAt() time.Time    { return b.updatedAt }
func (b *Blog) DeletedAt() *time.Time   { return cloneTime(b.deletedAt) }

// Edit distinguishes omitted fields from an explicitly empty content value.
// Validate the entire candidate before assigning anything to the aggregate.
func (b *Blog) Edit(title, content *string, now time.Time) error {
	if err := b.requireMutable(); err != nil {
		return err
	}
	candidateTitle, candidateContent := b.title, b.content
	if title != nil {
		candidateTitle = *title
	}
	if content != nil {
		candidateContent = *content
	}
	candidateTitle, candidateContent, err := validateBlogCandidate(candidateTitle, candidateContent, b.status)
	if err != nil {
		return err
	}
	if candidateTitle == b.title && candidateContent == b.content {
		return nil
	}
	b.title, b.content = candidateTitle, candidateContent
	b.updatedAt = normalizeInstant(now)
	return nil
}

func (b *Blog) Publish(now time.Time) error {
	if err := b.requireMutable(); err != nil {
		return err
	}
	title, content, err := validateBlogCandidate(b.title, b.content, enum.BlogPublished)
	if err != nil {
		return err
	}
	if b.status == enum.BlogPublished {
		return nil
	}
	b.title, b.content, b.status = title, content, enum.BlogPublished
	b.updatedAt = normalizeInstant(now)
	return nil
}

func (b *Blog) Unpublish(now time.Time) error {
	if err := b.requireMutable(); err != nil {
		return err
	}
	if b.status == enum.BlogDraft {
		return nil
	}
	b.status = enum.BlogDraft
	b.updatedAt = normalizeInstant(now)
	return nil
}

// SoftDelete retains publication state and identity. Repeat deletion is inert;
// the access policy decides which actors may invoke that no-op.
func (b *Blog) SoftDelete(now time.Time) error {
	if b.deletedAt != nil {
		return nil
	}
	if err := b.requireMutable(); err != nil {
		return err
	}
	stamp := normalizeInstant(now)
	b.deletedAt, b.updatedAt = &stamp, stamp
	return nil
}
