package blogrepository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/blogrepository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

func TestPostgresSaveExecutesUpsertSQL(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	blog := testdata.MustBlog(
		uuid.MustParse("b5000000-0000-4000-8000-000000000001"),
		uuid.MustParse("a5000000-0000-4000-8000-000000000001"),
		"sql-save-aaa1111",
		enum.PublicationDraft,
		nil,
		testdata.Instant(),
	)
	mock.ExpectExec(blogrepository.SaveSQL).
		WithArgs(
			blog.ID(),
			blog.OwnerID(),
			blog.Slug().String(),
			blog.Title().String(),
			blog.Content().HTML(),
			string(blog.Status()),
			blog.DeletedAt(),
			blog.CreatedAt(),
			blog.UpdatedAt(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := blogrepository.Postgres{DB: db}
	require.NoError(t, repo.Save(context.Background(), blog))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresSaveMapsUniqueSlug(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	blog := testdata.MustBlog(
		uuid.MustParse("b5100000-0000-4000-8000-000000000001"),
		uuid.MustParse("a5100000-0000-4000-8000-000000000001"),
		"sql-dup-bbb2222",
		enum.PublicationDraft,
		nil,
		testdata.Instant(),
	)
	mock.ExpectExec(blogrepository.SaveSQL).
		WithArgs(
			blog.ID(),
			blog.OwnerID(),
			blog.Slug().String(),
			blog.Title().String(),
			blog.Content().HTML(),
			string(blog.Status()),
			blog.DeletedAt(),
			blog.CreatedAt(),
			blog.UpdatedAt(),
		).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "blogs_slug_uidx"})

	repo := blogrepository.Postgres{DB: db}
	err = repo.Save(context.Background(), blog)
	require.ErrorIs(t, err, repository.ErrUniqueSlug)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresGetUsesForUpdateInsideTransaction(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	id := uuid.MustParse("b5200000-0000-4000-8000-000000000001")
	owner := uuid.MustParse("a5200000-0000-4000-8000-000000000001")
	now := testdata.Instant()
	rows := sqlmock.NewRows([]string{"id", "owner_id", "slug", "title", "content", "status", "deleted_at", "created_at", "updated_at"}).
		AddRow(id, owner, "sql-lock-ccc3333", testdata.BlogTitle(), testdata.BlogPublishableHTML(), "draft", nil, now, now)

	mock.ExpectBegin()
	mock.ExpectQuery(blogrepository.GetForUpdateSQL).WithArgs(id).WillReturnRows(rows)
	mock.ExpectCommit()

	repo := blogrepository.Postgres{DB: db}
	err = repo.RunInTransaction(context.Background(), func(ctx context.Context) error {
		got, getErr := repo.Get(ctx, id)
		require.NoError(t, getErr)
		require.NotNil(t, got)
		require.Equal(t, id, got.ID())
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresGetPublicBySlugJoinsAccounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	id := uuid.MustParse("b5300000-0000-4000-8000-000000000001")
	owner := uuid.MustParse("a5300000-0000-4000-8000-000000000001")
	now := testdata.Instant()
	slug := "sql-public-ddd4444"
	rows := sqlmock.NewRows([]string{"id", "owner_id", "slug", "title", "content", "status", "deleted_at", "created_at", "updated_at", "name"}).
		AddRow(id, owner, slug, testdata.BlogTitle(), testdata.BlogPublishableHTML(), "published", nil, now, now, "Public Author")
	mock.ExpectQuery(blogrepository.GetPublicBySlugSQL).WithArgs(slug).WillReturnRows(rows)

	repo := blogrepository.Postgres{DB: db}
	blog, authorName, err := repo.GetPublicBySlug(context.Background(), slug)
	require.NoError(t, err)
	require.NotNil(t, blog)
	require.Equal(t, id, blog.ID())
	assert.Equal(t, "Public Author", authorName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresGetPublicBySlugNotFound(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(blogrepository.GetPublicBySlugSQL).WithArgs("missing-slug").WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "slug", "title", "content", "status", "deleted_at", "created_at", "updated_at", "name"}),
	)
	repo := blogrepository.Postgres{DB: db}
	_, _, err = repo.GetPublicBySlug(context.Background(), "missing-slug")
	var domainErr *errs.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, errs.CodeBlogNotFound, domainErr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresListPublicUsesJoinThenPage(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	id := uuid.MustParse("b5400000-0000-4000-8000-000000000001")
	owner := uuid.MustParse("a5400000-0000-4000-8000-000000000001")
	now := testdata.Instant()
	mock.ExpectQuery(blogrepository.ListPublicCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(blogrepository.ListPublicSQL).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "slug", "title", "status", "name", "created_at", "deleted_at"}).
			AddRow(id, owner, "sql-list-eee5555", testdata.BlogTitle(), "published", "Lister", now, nil))

	repo := blogrepository.Postgres{DB: db}
	items, total, err := repo.ListPublic(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, id, items[0].ID)
	assert.Equal(t, "Lister", items[0].AuthorName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresListAdminExcludeFilter(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(blogrepository.ListAdminExcludeCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(blogrepository.ListAdminExcludeSQL).WithArgs(20, 0).WillReturnRows(
		sqlmock.NewRows([]string{"id", "owner_id", "slug", "title", "status", "name", "created_at", "deleted_at"}),
	)
	repo := blogrepository.Postgres{DB: db}
	items, total, err := repo.ListAdmin(context.Background(), 1, 20, repository.DeletedExclude)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, items)
	require.NoError(t, mock.ExpectationsWereMet())
}
