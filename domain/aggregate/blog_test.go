package aggregate_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlog(t *testing.T) {
	type testCase struct {
		name                 string
		wantErr              bool
		title, content, code string
	}
	tests := []testCase{
		{name: "empty draft", title: "  Valid blog title  "},
		{name: "unicode minimum", title: strings.Repeat("界", 10)},
		{name: "maximum title and content", title: strings.Repeat("界", 200), content: strings.Repeat("界", 1000)},
		{name: "short title", wantErr: true, title: "123456789", code: errs.CodeInvalidBlogTitle},
		{name: "long title", wantErr: true, title: strings.Repeat("界", 201), code: errs.CodeInvalidBlogTitle},
		{name: "invalid title utf8", wantErr: true, title: "Valid title" + string([]byte{255}), code: errs.CodeInvalidBlogTitle},
		{name: "long content", wantErr: true, title: "Valid blog title", content: strings.Repeat("界", 1001), code: errs.CodeInvalidBlogContent},
		{name: "invalid content", wantErr: true, title: "Valid blog title", content: string([]byte{255}), code: errs.CodeInvalidBlogContent},
		{name: "sanitized draft", title: "Valid blog title", content: "<p onclick='x'>hello</p><script>bad</script>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, owner := uuid.New(), uuid.New()
			now := testdata.Instant().In(time.FixedZone("offset", 3600)).Add(1234567 * time.Nanosecond)
			blog, err := aggregate.NewBlog(id, owner, tt.title, tt.content, now)
			if tt.wantErr {
				require.Error(t, err)
				var publicErr *errs.Error
				require.ErrorAs(t, err, &publicErr)
				assert.Equal(t, tt.code, publicErr.Code)
				assert.Nil(t, blog)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, blog)
			clean, err := service.ProcessBlogContent(tt.content)
			require.NoError(t, err)
			assert.Equal(t, clean.HTML, blog.Content())
			assert.Equal(t, strings.TrimSpace(tt.title), blog.Title())
			assert.Equal(t, service.BlogSlug(tt.title, id), blog.Slug())
			assert.Equal(t, id, blog.ID())
			assert.Equal(t, owner, blog.OwnerID())
			assert.Equal(t, enum.BlogDraft, blog.Status())
			assert.Nil(t, blog.DeletedAt())
			assert.Equal(t, now.UTC().Truncate(time.Millisecond), blog.CreatedAt())
			assert.Equal(t, blog.CreatedAt(), blog.UpdatedAt())
		})
	}
}

func TestBlogLifecycle(t *testing.T) {
	type testCase struct {
		name               string
		wantErr            bool
		operation          string
		published, deleted bool
		title, content     *string
		code               string
		changed            bool
	}
	title, short, valid := "Renamed blog title", "short", "<p>"+strings.Repeat("界", 100)+"</p>"
	tests := []testCase{
		{name: "publish draft", operation: "publish", changed: true},
		{name: "publish no-op", operation: "publish", published: true},
		{name: "unpublish", operation: "unpublish", published: true, changed: true},
		{name: "unpublish no-op", operation: "unpublish"},
		{name: "delete", operation: "delete", published: true, changed: true},
		{name: "repeat delete no-op", operation: "delete", deleted: true},
		{name: "rename stable slug", operation: "edit", title: &title, changed: true},
		{name: "published edit", operation: "edit", published: true, content: &valid, changed: true},
		{name: "rejected candidate atomic", wantErr: true, operation: "edit", published: true, title: &title, content: &short, code: errs.CodeInvalidBlogContent},
		{name: "rejected title atomic", wantErr: true, operation: "edit", title: &short, content: &valid, code: errs.CodeInvalidBlogTitle},
		{name: "empty edit no-op", operation: "edit"},
		{name: "deleted edit", wantErr: true, operation: "edit", deleted: true, title: &title, code: errs.CodeBlogDeleted},
		{name: "deleted publish", wantErr: true, operation: "publish", deleted: true, code: errs.CodeBlogDeleted},
		{name: "deleted unpublish", wantErr: true, operation: "unpublish", deleted: true, code: errs.CodeBlogDeleted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := testdata.NewClock(testdata.Instant())
			blog, err := aggregate.NewBlog(uuid.New(), uuid.New(), "Original blog title", strings.Repeat("a", 100), clock.Now())
			require.NoError(t, err)
			if tt.published {
				require.NoError(t, blog.Publish(clock.Now()))
			}
			if tt.deleted {
				require.NoError(t, blog.SoftDelete(clock.Now()))
			}
			before := *blog
			clock.Advance(time.Hour)
			switch tt.operation {
			case "edit":
				err = blog.Edit(tt.title, tt.content, clock.Now())
			case "publish":
				err = blog.Publish(clock.Now())
			case "unpublish":
				err = blog.Unpublish(clock.Now())
			case "delete":
				err = blog.SoftDelete(clock.Now())
			}
			if tt.wantErr {
				require.Error(t, err)
				var publicErr *errs.Error
				require.ErrorAs(t, err, &publicErr)
				assert.Equal(t, tt.code, publicErr.Code)
			} else {
				require.NoError(t, err)
			}
			if !tt.changed {
				assert.Equal(t, before, *blog)
			} else {
				assert.Equal(t, clock.Now(), blog.UpdatedAt())
			}
			assert.Equal(t, before.ID(), blog.ID())
			assert.Equal(t, before.OwnerID(), blog.OwnerID())
			assert.Equal(t, before.Slug(), blog.Slug())
			assert.Equal(t, before.CreatedAt(), blog.CreatedAt())
			if !tt.wantErr {
				switch tt.operation {
				case "publish":
					assert.Equal(t, enum.BlogPublished, blog.Status())
				case "unpublish":
					assert.Equal(t, enum.BlogDraft, blog.Status())
				case "delete":
					require.NotNil(t, blog.DeletedAt())
					stamp := *blog.DeletedAt()
					escaped := blog.DeletedAt()
					*escaped = time.Time{}
					assert.Equal(t, stamp, *blog.DeletedAt())
				case "edit":
					if tt.title != nil {
						assert.Equal(t, *tt.title, blog.Title())
					}
					if tt.content != nil {
						assert.Equal(t, *tt.content, blog.Content())
					}
				}
			}
		})
	}
}

func TestBlogPublishValidation(t *testing.T) {
	type testCase struct {
		name    string
		wantErr bool
		content string
	}
	tests := []testCase{
		{name: "empty", wantErr: true},
		{name: "below minimum", wantErr: true, content: strings.Repeat("界", 99)},
		{name: "minimum", content: strings.Repeat("界", 100)},
		{name: "maximum", content: strings.Repeat("界", 1000)},
		{name: "restored oversized", wantErr: true, content: strings.Repeat("界", 1001)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := testdata.Instant()
			blog := aggregate.RestoreBlog(uuid.New(), uuid.New(), "stored-slug", "Original blog title", tt.content, enum.BlogDraft, now, now, nil)
			before := *blog
			err := blog.Publish(now.Add(time.Hour))
			if tt.wantErr {
				require.Error(t, err)
				var publicErr *errs.Error
				require.ErrorAs(t, err, &publicErr)
				assert.Equal(t, errs.CodeInvalidBlogContent, publicErr.Code)
				assert.Equal(t, before, *blog)
			} else {
				require.NoError(t, err)
				assert.Equal(t, enum.BlogPublished, blog.Status())
			}
			assert.Equal(t, "stored-slug", blog.Slug())
		})
	}
}
