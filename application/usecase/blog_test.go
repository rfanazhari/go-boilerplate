package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/mocks"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

type taggedSanitizer struct{}

func (taggedSanitizer) Sanitize(raw string) string {
	if raw == "hello" {
		return "CLEAN:hello"
	}
	return raw
}

type blogCase struct {
	name string
	run  func(t *testing.T, h blogHarness)
}

type blogHarness struct {
	uc       usecase.BlogUseCase
	blogs    *mocks.MockBlogRepository
	accounts *mocks.MockAccountRepository
	clock    *testdata.Clock
}

var (
	blogOwnerID   = uuid.MustParse("b0000000-0000-4000-8000-000000000001")
	blogStranger  = uuid.MustParse("b0000000-0000-4000-8000-000000000002")
	blogAdminID   = uuid.MustParse("b0000000-0000-4000-8000-000000000003")
	blogRecordID  = uuid.MustParse("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	blogCreatedAt = testdata.Instant()
)

func blogUser(id uuid.UUID) dto.BlogActor {
	return dto.BlogActor{ID: id, Role: enum.RoleUser}
}

func blogAdmin(id uuid.UUID) dto.BlogActor {
	return dto.BlogActor{ID: id, Role: enum.RoleAdmin}
}

func newBlogHarness(t *testing.T) blogHarness {
	t.Helper()
	blogs := mocks.NewMockBlogRepository(t)
	accounts := mocks.NewMockAccountRepository(t)
	clock := testdata.NewClock(blogCreatedAt)
	uc := usecase.BlogUseCase{
		Blogs:     blogs,
		Accounts:  accounts,
		Sanitizer: taggedSanitizer{},
		Clock:     clock,
		IDs:       func() uuid.UUID { return blogRecordID },
		Suffix:    func() string { return "a1b2c3d4" },
	}
	return blogHarness{uc: uc, blogs: blogs, accounts: accounts, clock: clock}
}

func expectTx(blogs *mocks.MockBlogRepository) {
	blogs.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)
}

func requireAppCode(t *testing.T, err error, code string) {
	t.Helper()
	require.Error(t, err)
	var appErr *errs.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, code, appErr.Code)
}

func restoredBlog(status enum.PublicationStatus, deletedAt *time.Time, html string) *aggregate.Blog {
	return aggregate.RestoreBlog(
		blogRecordID,
		blogOwnerID,
		aggregate.RestoreBlogTitle(testdata.BlogTitle()),
		aggregate.RestoreBlogContent(html),
		aggregate.RestoreBlogSlug("catatan-go-a1b2c3d4"),
		status,
		deletedAt,
		blogCreatedAt,
		blogCreatedAt,
	)
}

func activeOwner() *aggregate.Account {
	return testdata.MustAccount(blogOwnerID, "Owner", enum.AccountActive, nil)
}

func TestBlogUseCaseScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []blogCase{
		{
			name: "admin create is forbidden and does not save",
			run: func(t *testing.T, h blogHarness) {
				_, err := h.uc.Create(ctx, blogAdmin(blogAdminID), dto.CreateBlogInput{Title: testdata.BlogTitle(), Content: "hello"})
				requireAppCode(t, err, errs.CodeForbidden)
			},
		},
		{
			name: "create assigns owner timestamps draft slug and sanitizer output",
			run: func(t *testing.T, h blogHarness) {
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, blog *aggregate.Blog) {
					assert.Equal(t, blogOwnerID, blog.OwnerID())
					assert.Equal(t, blogRecordID, blog.ID())
					assert.Equal(t, enum.PublicationDraft, blog.Status())
					assert.Equal(t, "catatan-go-a1b2c3d4", blog.Slug().String())
					assert.Equal(t, "CLEAN:hello", blog.Content().HTML())
					assert.Equal(t, blogCreatedAt, blog.CreatedAt())
					assert.Equal(t, blogCreatedAt, blog.UpdatedAt())
					assert.Nil(t, blog.DeletedAt())
				}).Return(nil)
				view, err := h.uc.Create(ctx, blogUser(blogOwnerID), dto.CreateBlogInput{Title: testdata.BlogTitle(), Content: "hello"})
				require.NoError(t, err)
				assert.Equal(t, blogOwnerID, view.OwnerID)
				assert.Equal(t, blogCreatedAt, view.CreatedAt)
				assert.Equal(t, string(enum.PublicationDraft), view.Status)
				assert.Equal(t, "CLEAN:hello", view.Content)
				assert.Equal(t, "catatan-go-a1b2c3d4", view.Slug)
			},
		},
		{
			name: "rename keeps slug",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, testdata.BlogPublishableHTML()), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, blog *aggregate.Blog) {
					assert.Equal(t, "catatan-go-a1b2c3d4", blog.Slug().String())
					assert.Equal(t, "Judul Baru Panjang", blog.Title().String())
				}).Return(nil)
				view, err := h.uc.Update(ctx, blogUser(blogOwnerID), dto.UpdateBlogInput{
					ID:      blogRecordID,
					Title:   "Judul Baru Panjang",
					Content: testdata.BlogPublishableHTML(),
				})
				require.NoError(t, err)
				assert.Equal(t, "catatan-go-a1b2c3d4", view.Slug)
			},
		},
		{
			name: "invalid published edit writes nothing",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML()), nil)
				_, err := h.uc.Update(ctx, blogUser(blogOwnerID), dto.UpdateBlogInput{
					ID:      blogRecordID,
					Title:   testdata.BlogTitle(),
					Content: strings.Repeat("a", 99),
				})
				requireAppCode(t, err, errs.CodeInvalidBlogContent)
			},
		},
		{
			name: "publish content length 99 is invalid",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, strings.Repeat("a", 99)), nil)
				_, err := h.uc.Publish(ctx, blogUser(blogOwnerID), blogRecordID)
				requireAppCode(t, err, errs.CodeInvalidBlogContent)
			},
		},
		{
			name: "publish content length 100 succeeds",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, strings.Repeat("a", 100)), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				view, err := h.uc.Publish(ctx, blogUser(blogOwnerID), blogRecordID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.PublicationPublished), view.Status)
			},
		},
		{
			name: "publish content length 1000 succeeds",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, strings.Repeat("a", 1000)), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				view, err := h.uc.Publish(ctx, blogUser(blogOwnerID), blogRecordID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.PublicationPublished), view.Status)
			},
		},
		{
			name: "publish content length 1001 is invalid",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, strings.Repeat("a", 1001)), nil)
				_, err := h.uc.Publish(ctx, blogUser(blogOwnerID), blogRecordID)
				requireAppCode(t, err, errs.CodeInvalidBlogContent)
			},
		},
		{
			name: "idempotent publish does not save",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML()), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				view, err := h.uc.Publish(ctx, blogUser(blogOwnerID), blogRecordID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.PublicationPublished), view.Status)
			},
		},
		{
			name: "idempotent unpublish does not save",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, testdata.BlogPublishableHTML()), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				view, err := h.uc.Unpublish(ctx, blogUser(blogOwnerID), blogRecordID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.PublicationDraft), view.Status)
			},
		},
		{
			name: "admin cannot edit",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, testdata.BlogPublishableHTML()), nil)
				_, err := h.uc.Update(ctx, blogAdmin(blogAdminID), dto.UpdateBlogInput{
					ID:      blogRecordID,
					Title:   testdata.BlogTitle(),
					Content: "hello",
				})
				requireAppCode(t, err, errs.CodeForbidden)
			},
		},
		{
			name: "admin cannot publish",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, testdata.BlogPublishableHTML()), nil)
				_, err := h.uc.Publish(ctx, blogAdmin(blogAdminID), blogRecordID)
				requireAppCode(t, err, errs.CodeForbidden)
			},
		},
		{
			name: "admin can unpublish",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML()), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, blog *aggregate.Blog) {
					assert.Equal(t, enum.PublicationDraft, blog.Status())
				}).Return(nil)
				view, err := h.uc.Unpublish(ctx, blogAdmin(blogAdminID), blogRecordID)
				require.NoError(t, err)
				assert.Equal(t, string(enum.PublicationDraft), view.Status)
			},
		},
		{
			name: "owner delete then owner get is not found",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				live := restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML())
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(live, nil).Once()
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				require.NoError(t, h.uc.Delete(ctx, blogUser(blogOwnerID), blogRecordID))

				deletedAt := blogCreatedAt
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, &deletedAt, testdata.BlogPublishableHTML()), nil)
				_, err := h.uc.Get(ctx, blogUser(blogOwnerID), blogRecordID)
				requireAppCode(t, err, errs.CodeBlogNotFound)
			},
		},
		{
			name: "admin can delete",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML()), nil)
				h.blogs.EXPECT().Save(mock.Anything, mock.Anything).Run(func(_ context.Context, blog *aggregate.Blog) {
					require.NotNil(t, blog.DeletedAt())
				}).Return(nil)
				require.NoError(t, h.uc.Delete(ctx, blogAdmin(blogAdminID), blogRecordID))
			},
		},
		{
			name: "second owner delete is not found",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				deletedAt := blogCreatedAt
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, &deletedAt, testdata.BlogPublishableHTML()), nil)
				err := h.uc.Delete(ctx, blogUser(blogOwnerID), blogRecordID)
				requireAppCode(t, err, errs.CodeBlogNotFound)
			},
		},
		{
			name: "second admin delete succeeds with no write",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				deletedAt := blogCreatedAt
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, &deletedAt, testdata.BlogPublishableHTML()), nil)
				require.NoError(t, h.uc.Delete(ctx, blogAdmin(blogAdminID), blogRecordID))
			},
		},
		{
			name: "admin edit of deleted blog is blog deleted",
			run: func(t *testing.T, h blogHarness) {
				expectTx(h.blogs)
				deletedAt := blogCreatedAt
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, &deletedAt, testdata.BlogPublishableHTML()), nil)
				_, err := h.uc.Update(ctx, blogAdmin(blogAdminID), dto.UpdateBlogInput{
					ID:      blogRecordID,
					Title:   testdata.BlogTitle(),
					Content: testdata.BlogPublishableHTML(),
				})
				requireAppCode(t, err, errs.CodeBlogDeleted)
			},
		},
		{
			name: "stranger gets forbidden on a public blog",
			run: func(t *testing.T, h blogHarness) {
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML()), nil)
				h.accounts.EXPECT().GetByID(mock.Anything, blogOwnerID).Return(activeOwner(), nil)
				_, err := h.uc.Get(ctx, blogUser(blogStranger), blogRecordID)
				requireAppCode(t, err, errs.CodeForbidden)
			},
		},
		{
			name: "stranger gets not found on a draft",
			run: func(t *testing.T, h blogHarness) {
				h.blogs.EXPECT().Get(mock.Anything, blogRecordID).Return(restoredBlog(enum.PublicationDraft, nil, testdata.BlogPublishableHTML()), nil)
				_, err := h.uc.Get(ctx, blogUser(blogStranger), blogRecordID)
				requireAppCode(t, err, errs.CodeBlogNotFound)
			},
		},
		{
			name: "public get of a draft is not found",
			run: func(t *testing.T, h blogHarness) {
				h.blogs.EXPECT().GetPublicBySlug(mock.Anything, "catatan-go-a1b2c3d4").Return(
					nil, "", errs.New(errs.CodeBlogNotFound, errs.KindNotFound, assert.AnError),
				)
				_, err := h.uc.GetPublic(ctx, "catatan-go-a1b2c3d4")
				requireAppCode(t, err, errs.CodeBlogNotFound)
			},
		},
		{
			name: "inactive author hidden from public then visible when active",
			run: func(t *testing.T, h blogHarness) {
				h.blogs.EXPECT().GetPublicBySlug(mock.Anything, "catatan-go-a1b2c3d4").Return(
					nil, "", errs.New(errs.CodeBlogNotFound, errs.KindNotFound, assert.AnError),
				).Once()
				_, err := h.uc.GetPublic(ctx, "catatan-go-a1b2c3d4")
				requireAppCode(t, err, errs.CodeBlogNotFound)

				published := restoredBlog(enum.PublicationPublished, nil, testdata.BlogPublishableHTML())
				h.blogs.EXPECT().GetPublicBySlug(mock.Anything, "catatan-go-a1b2c3d4").Return(published, "Owner", nil)
				view, err := h.uc.GetPublic(ctx, "catatan-go-a1b2c3d4")
				require.NoError(t, err)
				assert.Equal(t, string(enum.PublicationPublished), view.Status)
				assert.Equal(t, "Owner", view.AuthorName)
			},
		},
		{
			name: "omitted page defaults to 1 and size 20",
			run: func(t *testing.T, h blogHarness) {
				h.blogs.EXPECT().ListPublic(mock.Anything, 1, 20).Return([]repository.BlogSummary{}, 0, nil)
				page, err := h.uc.ListPublic(ctx, dto.ListBlogsInput{})
				require.NoError(t, err)
				assert.Equal(t, 1, page.Page)
				assert.Equal(t, 20, page.PageSize)
			},
		},
		{
			name: "pageSize 101 is rejected",
			run: func(t *testing.T, h blogHarness) {
				size := 101
				_, err := h.uc.ListPublic(ctx, dto.ListBlogsInput{PageSize: &size})
				requireAppCode(t, err, errs.CodeValidationFailed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBlogHarness(t)
			tt.run(t, h)
		})
	}
}
