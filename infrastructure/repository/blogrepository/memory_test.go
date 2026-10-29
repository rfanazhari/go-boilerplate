package blogrepository_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/blogrepository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/mocks"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

type publicVisibilityCase struct {
	name      string
	slug      string
	wantFound bool
}

func TestMemoryPublicListAndDetailHideSameFixtures(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	deletedAt := now

	active := testdata.MustAccount(uuid.MustParse("a0000000-0000-4000-8000-000000000001"), "Active Author", enum.AccountActive, nil)
	inactive := testdata.MustAccount(uuid.MustParse("a0000000-0000-4000-8000-000000000002"), "Inactive Author", enum.AccountNonActive, nil)
	deletedAuthor := testdata.MustAccount(uuid.MustParse("a0000000-0000-4000-8000-000000000003"), "Deleted Author", enum.AccountActive, &deletedAt)

	eligible := testdata.MustBlog(uuid.MustParse("b0000000-0000-4000-8000-000000000001"), active.ID(), "eligible-aaaa1111", enum.PublicationPublished, nil, now)
	draft := testdata.MustBlog(uuid.MustParse("b0000000-0000-4000-8000-000000000002"), active.ID(), "draft-bbbb2222", enum.PublicationDraft, nil, now.Add(-time.Second))
	deletedBlog := testdata.MustBlog(uuid.MustParse("b0000000-0000-4000-8000-000000000003"), active.ID(), "deleted-cccc3333", enum.PublicationPublished, &deletedAt, now.Add(-2*time.Second))
	inactiveOwned := testdata.MustBlog(uuid.MustParse("b0000000-0000-4000-8000-000000000004"), inactive.ID(), "inactive-dddd4444", enum.PublicationPublished, nil, now.Add(-3*time.Second))
	deletedOwned := testdata.MustBlog(uuid.MustParse("b0000000-0000-4000-8000-000000000005"), deletedAuthor.ID(), "delauthor-eeee5555", enum.PublicationPublished, nil, now.Add(-4*time.Second))

	accounts := mocks.NewMockAccountRepository(t)
	accounts.EXPECT().GetByID(mock.Anything, active.ID()).Return(active, nil).Maybe()
	accounts.EXPECT().GetByID(mock.Anything, inactive.ID()).Return(inactive, nil).Maybe()
	accounts.EXPECT().GetByID(mock.Anything, deletedAuthor.ID()).Return(deletedAuthor, nil).Maybe()

	repo := &blogrepository.Memory{Accounts: accounts}
	require.NoError(t, repo.Save(ctx, eligible))
	require.NoError(t, repo.Save(ctx, draft))
	require.NoError(t, repo.Save(ctx, deletedBlog))
	require.NoError(t, repo.Save(ctx, inactiveOwned))
	require.NoError(t, repo.Save(ctx, deletedOwned))

	items, total, err := repo.ListPublic(ctx, 1, 20)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, eligible.ID(), items[0].ID)
	assert.Equal(t, "Active Author", items[0].AuthorName)

	cases := []publicVisibilityCase{
		{name: "eligible published is public", slug: eligible.Slug().String(), wantFound: true},
		{name: "draft is hidden", slug: draft.Slug().String(), wantFound: false},
		{name: "deleted blog is hidden", slug: deletedBlog.Slug().String(), wantFound: false},
		{name: "inactive author is hidden", slug: inactiveOwned.Slug().String(), wantFound: false},
		{name: "deleted author is hidden", slug: deletedOwned.Slug().String(), wantFound: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			blog, authorName, err := repo.GetPublicBySlug(ctx, tt.slug)
			if tt.wantFound {
				require.NoError(t, err)
				require.NotNil(t, blog)
				assert.Equal(t, tt.slug, blog.Slug().String())
				assert.Equal(t, "Active Author", authorName)
				return
			}
			require.Error(t, err)
			var domainErr *errs.Error
			require.ErrorAs(t, err, &domainErr)
			assert.Equal(t, errs.CodeBlogNotFound, domainErr.Code)
			assert.Nil(t, blog)
		})
	}
}

func TestMemoryListPublicPagesAfterFilter(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	active := testdata.MustAccount(uuid.MustParse("a1000000-0000-4000-8000-000000000001"), "Pager", enum.AccountActive, nil)
	accounts := mocks.NewMockAccountRepository(t)
	accounts.EXPECT().GetByID(mock.Anything, active.ID()).Return(active, nil).Maybe()
	repo := &blogrepository.Memory{Accounts: accounts}

	older := testdata.MustBlog(uuid.MustParse("b1000000-0000-4000-8000-000000000001"), active.ID(), "page-older1111", enum.PublicationPublished, nil, now.Add(-2*time.Second))
	mid := testdata.MustBlog(uuid.MustParse("b1000000-0000-4000-8000-000000000002"), active.ID(), "page-mid222222", enum.PublicationPublished, nil, now.Add(-time.Second))
	newest := testdata.MustBlog(uuid.MustParse("b1000000-0000-4000-8000-000000000003"), active.ID(), "page-new333333", enum.PublicationPublished, nil, now)
	draft := testdata.MustBlog(uuid.MustParse("b1000000-0000-4000-8000-000000000004"), active.ID(), "page-draft4444", enum.PublicationDraft, nil, now.Add(time.Second))
	require.NoError(t, repo.Save(ctx, older))
	require.NoError(t, repo.Save(ctx, mid))
	require.NoError(t, repo.Save(ctx, newest))
	require.NoError(t, repo.Save(ctx, draft))

	page1, total, err := repo.ListPublic(ctx, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page1, 2)
	assert.Equal(t, newest.ID(), page1[0].ID)
	assert.Equal(t, mid.ID(), page1[1].ID)

	page2, total, err := repo.ListPublic(ctx, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page2, 1)
	assert.Equal(t, older.ID(), page2[0].ID)
}

func TestMemorySaveRejectsDuplicateSlugIncludingDeleted(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	owner := uuid.MustParse("a2000000-0000-4000-8000-000000000001")
	deletedAt := now
	first := testdata.MustBlog(uuid.MustParse("b2000000-0000-4000-8000-000000000001"), owner, "taken-slug1111", enum.PublicationPublished, &deletedAt, now)
	second := testdata.MustBlog(uuid.MustParse("b2000000-0000-4000-8000-000000000002"), owner, "taken-slug1111", enum.PublicationDraft, nil, now)
	repo := &blogrepository.Memory{}
	require.NoError(t, repo.Save(ctx, first))
	err := repo.Save(ctx, second)
	require.ErrorIs(t, err, repository.ErrUniqueSlug)
}

func TestMemoryRunInTransactionHoldsMutex(t *testing.T) {
	ctx := context.Background()
	repo := &blogrepository.Memory{}
	started := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := repo.RunInTransaction(ctx, func(context.Context) error {
			close(started)
			<-release
			return nil
		})
		require.NoError(t, err)
	}()
	<-started

	saved := make(chan struct{})
	blog := testdata.MustBlog(uuid.MustParse("b3000000-0000-4000-8000-000000000001"), uuid.MustParse("a3000000-0000-4000-8000-000000000001"), "tx-lock-aaaa1111", enum.PublicationDraft, nil, testdata.Instant())
	go func() {
		_ = repo.Save(ctx, blog)
		close(saved)
	}()

	select {
	case <-saved:
		t.Fatal("Save interleaved with RunInTransaction")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	wg.Wait()
	<-saved
}

func TestMemoryListAdminDeletedFilter(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	active := testdata.MustAccount(uuid.MustParse("a4000000-0000-4000-8000-000000000001"), "AdminList", enum.AccountActive, nil)
	accounts := mocks.NewMockAccountRepository(t)
	accounts.EXPECT().GetByID(mock.Anything, active.ID()).Return(active, nil).Maybe()
	repo := &blogrepository.Memory{Accounts: accounts}
	live := testdata.MustBlog(uuid.MustParse("b4000000-0000-4000-8000-000000000001"), active.ID(), "admin-live1111", enum.PublicationDraft, nil, now)
	gone := testdata.MustBlog(uuid.MustParse("b4000000-0000-4000-8000-000000000002"), active.ID(), "admin-gone2222", enum.PublicationPublished, &now, now.Add(-time.Second))
	require.NoError(t, repo.Save(ctx, live))
	require.NoError(t, repo.Save(ctx, gone))

	type adminFilterCase struct {
		name   string
		filter repository.DeletedFilter
		want   int
	}
	cases := []adminFilterCase{
		{name: "exclude omits deleted", filter: repository.DeletedExclude, want: 1},
		{name: "only returns deleted", filter: repository.DeletedOnly, want: 1},
		{name: "all returns both", filter: repository.DeletedAll, want: 2},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			items, total, err := repo.ListAdmin(ctx, 1, 20, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.want, total)
			assert.Len(t, items, tt.want)
		})
	}
}
