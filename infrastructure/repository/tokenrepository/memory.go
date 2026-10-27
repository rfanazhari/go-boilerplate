package tokenrepository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Memory struct{}

func (Memory) Save(_ context.Context, _ uuid.UUID, _ string) error {
	return errors.New("tokenrepository.Memory.Save not implemented")
}

func (Memory) GetByHash(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, errs.New(errs.CodeUnauthenticated, errs.KindUnauthenticated, errors.New("token not found"))
}

func (Memory) DeleteByAccountID(_ context.Context, _ uuid.UUID) error {
	return nil
}
