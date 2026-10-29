package aggregate_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var (
	blogOwnerID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	blogID      = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)

func mustTitle(t *testing.T, raw string) aggregate.BlogTitle {
	t.Helper()
	title, err := aggregate.NewBlogTitle(raw)
	require.NoError(t, err)
	return title
}

func mustDraftContent(t *testing.T, raw string) aggregate.BlogContent {
	t.Helper()
	content, err := aggregate.NewBlogContent(identitySanitizer{}, raw, enum.PublicationDraft)
	require.NoError(t, err)
	return content
}

func mustSlug(t *testing.T, title, suffix string) aggregate.BlogSlug {
	t.Helper()
	slug, err := aggregate.NewBlogSlug(title, suffix)
	require.NoError(t, err)
	return slug
}

func TestNewBlog(t *testing.T) {
	now := time.Date(2026, 10, 30, 3, 0, 0, 0, time.UTC)

	t.Run("reject missing owner id", func(t *testing.T) {
		_, err := aggregate.NewBlog(
			blogID,
			uuid.Nil,
			mustTitle(t, "Catatan Go"),
			mustDraftContent(t, ""),
			mustSlug(t, "Catatan Go", "a1b2c3d4"),
			now,
		)
		require.Error(t, err)
		var domainErr *errs.Error
		require.ErrorAs(t, err, &domainErr)
		assert.Equal(t, errs.CodeValidationFailed, domainErr.Code)
	})

	t.Run("new blog is a draft owned by creator", func(t *testing.T) {
		title := mustTitle(t, "Catatan Go")
		content := mustDraftContent(t, "")
		slug := mustSlug(t, "Catatan Go", "a1b2c3d4")
		blog, err := aggregate.NewBlog(blogID, blogOwnerID, title, content, slug, now)
		require.NoError(t, err)
		require.NotNil(t, blog)
		assert.Equal(t, blogID, blog.ID())
		assert.Equal(t, blogOwnerID, blog.OwnerID())
		assert.Equal(t, enum.PublicationDraft, blog.Status())
		assert.Equal(t, "Catatan Go", blog.Title().String())
		assert.Equal(t, "catatan-go-a1b2c3d4", blog.Slug().String())
		assert.Nil(t, blog.DeletedAt())
		assert.Equal(t, now, blog.CreatedAt())
		assert.Equal(t, now, blog.UpdatedAt())
	})
}

type blogLifecycleCase struct {
	name     string
	setup    func(t *testing.T, now time.Time) *aggregate.Blog
	action   func(blog *aggregate.Blog, now time.Time) error
	wantErr  bool
	wantCode string
	assert   func(t *testing.T, blog *aggregate.Blog, now time.Time)
}

func TestBlogLifecycle(t *testing.T) {
	created := time.Date(2026, 10, 30, 3, 0, 0, 0, time.UTC)
	later := created.Add(time.Minute)

	tests := []blogLifecycleCase{
		{
			name: "second publish is idempotent and leaves updated at",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, strings.Repeat("a", 100)),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				require.NoError(t, blog.Publish(later))
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				return blog.Publish(now)
			},
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, enum.PublicationPublished, blog.Status())
				assert.Equal(t, later, blog.UpdatedAt())
			},
		},
		{
			name: "second unpublish is idempotent and leaves updated at",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, strings.Repeat("a", 100)),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				require.NoError(t, blog.Publish(later))
				require.NoError(t, blog.Unpublish(later))
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				return blog.Unpublish(now)
			},
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, enum.PublicationDraft, blog.Status())
				assert.Equal(t, later, blog.UpdatedAt())
			},
		},
		{
			name: "publish of 99 runes keeps draft",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, strings.Repeat("a", 99)),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				return blog.Publish(now)
			},
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogContent,
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, enum.PublicationDraft, blog.Status())
				assert.Equal(t, created, blog.UpdatedAt())
			},
		},
		{
			name: "revise does not change slug or status",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, ""),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				title, err := aggregate.NewBlogTitle("Judul baru blog")
				if err != nil {
					return err
				}
				content, err := aggregate.NewBlogContent(identitySanitizer{}, "updated body", enum.PublicationDraft)
				if err != nil {
					return err
				}
				return blog.Revise(title, content, now)
			},
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, "Judul baru blog", blog.Title().String())
				assert.Equal(t, "updated body", blog.Content().HTML())
				assert.Equal(t, "catatan-go-a1b2c3d4", blog.Slug().String())
				assert.Equal(t, enum.PublicationDraft, blog.Status())
				assert.Equal(t, now, blog.UpdatedAt())
			},
		},
		{
			name: "invalid published revise changes no field",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, strings.Repeat("a", 100)),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				require.NoError(t, blog.Publish(later))
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				title, err := aggregate.NewBlogTitle("Judul baru blog")
				if err != nil {
					return err
				}
				content, err := aggregate.NewBlogContent(identitySanitizer{}, strings.Repeat("a", 99), enum.PublicationDraft)
				if err != nil {
					return err
				}
				return blog.Revise(title, content, now)
			},
			wantErr:  true,
			wantCode: errs.CodeInvalidBlogContent,
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, "Catatan Go", blog.Title().String())
				assert.Equal(t, strings.Repeat("a", 100), blog.Content().HTML())
				assert.Equal(t, enum.PublicationPublished, blog.Status())
				assert.Equal(t, later, blog.UpdatedAt())
			},
		},
		{
			name: "soft delete refuses later mutations and keeps status",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, strings.Repeat("a", 100)),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				require.NoError(t, blog.Publish(later))
				require.NoError(t, blog.SoftDelete(later))
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				return blog.Publish(now)
			},
			wantErr:  true,
			wantCode: errs.CodeBlogDeleted,
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, enum.PublicationPublished, blog.Status())
				assert.Equal(t, later, blog.UpdatedAt())
				require.NotNil(t, blog.DeletedAt())
				assert.Equal(t, later, *blog.DeletedAt())
			},
		},
		{
			name: "second soft delete reports already deleted",
			setup: func(t *testing.T, now time.Time) *aggregate.Blog {
				blog, err := aggregate.NewBlog(
					blogID,
					blogOwnerID,
					mustTitle(t, "Catatan Go"),
					mustDraftContent(t, ""),
					mustSlug(t, "Catatan Go", "a1b2c3d4"),
					now,
				)
				require.NoError(t, err)
				require.NoError(t, blog.SoftDelete(later))
				return blog
			},
			action: func(blog *aggregate.Blog, now time.Time) error {
				return blog.SoftDelete(now)
			},
			wantErr:  true,
			wantCode: errs.CodeBlogDeleted,
			assert: func(t *testing.T, blog *aggregate.Blog, now time.Time) {
				assert.Equal(t, enum.PublicationDraft, blog.Status())
				assert.Equal(t, later, blog.UpdatedAt())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blog := tt.setup(t, created)
			err := tt.action(blog, later.Add(time.Hour))
			if tt.wantErr {
				require.Error(t, err)
				var domainErr *errs.Error
				require.ErrorAs(t, err, &domainErr)
				assert.Equal(t, tt.wantCode, domainErr.Code)
			} else {
				require.NoError(t, err)
			}
			tt.assert(t, blog, later.Add(time.Hour))
		})
	}
}

func TestRestoreBlog(t *testing.T) {
	created := time.Date(2026, 10, 30, 3, 0, 0, 123000000, time.UTC)
	deletedAt := created.Add(time.Hour)

	t.Run("loads a short title without rewriting it", func(t *testing.T) {
		blog := aggregate.RestoreBlog(
			blogID,
			blogOwnerID,
			aggregate.RestoreBlogTitle("short"),
			aggregate.RestoreBlogContent("<p>x</p>"),
			aggregate.RestoreBlogSlug("legacy-slug-aabbccdd"),
			enum.PublicationDraft,
			nil,
			created,
			created,
		)
		assert.Equal(t, "short", blog.Title().String())
		assert.Equal(t, "<p>x</p>", blog.Content().HTML())
		assert.Equal(t, "legacy-slug-aabbccdd", blog.Slug().String())
		assert.Equal(t, enum.PublicationDraft, blog.Status())
		assert.Equal(t, created.UTC().Truncate(time.Millisecond), blog.CreatedAt())
	})

	t.Run("loads a deleted published row without rewriting it", func(t *testing.T) {
		blog := aggregate.RestoreBlog(
			blogID,
			blogOwnerID,
			aggregate.RestoreBlogTitle("Catatan Go"),
			aggregate.RestoreBlogContent(strings.Repeat("a", 100)),
			aggregate.RestoreBlogSlug("catatan-go-a1b2c3d4"),
			enum.PublicationPublished,
			&deletedAt,
			created,
			deletedAt,
		)
		assert.Equal(t, enum.PublicationPublished, blog.Status())
		require.NotNil(t, blog.DeletedAt())
		assert.Equal(t, deletedAt.UTC().Truncate(time.Millisecond), *blog.DeletedAt())
		assert.True(t, blog.IsSoftDeleted())
	})
}
