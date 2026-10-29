package testdata

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
)

type Clock struct {
	mu  sync.Mutex
	now time.Time
}

func NewClock(now time.Time) *Clock {
	return &Clock{now: now.UTC().Truncate(time.Millisecond)}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TitlePrepareReport() string {
	return "Prepare report"
}

func Instant() time.Time {
	return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
}

type IdentitySanitizer struct{}

func (IdentitySanitizer) Sanitize(raw string) string {
	return raw
}

func BlogTitle() string {
	return "Catatan Go"
}

func BlogPublishableHTML() string {
	return "<p>" + strings.Repeat("a", 100) + "</p>"
}

func MustAccount(id uuid.UUID, name string, status enum.AccountStatus, deletedAt *time.Time) *aggregate.Account {
	now := Instant()
	return aggregate.RestoreAccount(id, name+"@example.com", name, "hash", enum.RoleUser, status, deletedAt, now, now)
}

func MustBlog(id, ownerID uuid.UUID, slug string, status enum.PublicationStatus, deletedAt *time.Time, createdAt time.Time) *aggregate.Blog {
	title, err := aggregate.NewBlogTitle(BlogTitle())
	if err != nil {
		panic(err)
	}
	content, err := aggregate.NewBlogContent(IdentitySanitizer{}, BlogPublishableHTML(), enum.PublicationDraft)
	if err != nil {
		panic(err)
	}
	blog := aggregate.RestoreBlog(id, ownerID, title, content, aggregate.RestoreBlogSlug(slug), status, deletedAt, createdAt, createdAt)
	return blog
}

var _ service.HTMLSanitizer = IdentitySanitizer{}
