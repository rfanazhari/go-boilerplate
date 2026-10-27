package taskrepository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var errInfraDisabled = errors.New("postgres is not configured")

type Unavailable struct{}

func (Unavailable) Get(context.Context, uuid.UUID) (*aggregate.Task, error) {
	return nil, errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) Save(context.Context, *aggregate.Task) error {
	return errs.New(errs.CodeInternalError, errs.KindInternal, errInfraDisabled)
}

func (Unavailable) RunInTransaction(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}
