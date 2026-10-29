package blogrepository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var errInfraDisabled = errors.New("postgres is not configured")

type Unavailable struct{}

func (Unavailable) Save(context.Context, *aggregate.Blog) error {
	return errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) Get(context.Context, uuid.UUID) (*aggregate.Blog, error) {
	return nil, errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) GetPublicBySlug(context.Context, string) (*aggregate.Blog, string, error) {
	return nil, "", errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) ListPublic(context.Context, int, int) ([]repository.BlogSummary, int, error) {
	return nil, 0, errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) ListAdmin(context.Context, int, int, repository.DeletedFilter) ([]repository.BlogSummary, int, error) {
	return nil, 0, errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) RunInTransaction(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}
