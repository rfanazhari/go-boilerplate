package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
)

var errNotImplemented = errors.New("not implemented")

type AccountUseCase struct {
	Accounts repository.AccountRepository
	Hasher   service.PasswordHasher
	Clock    service.Clock
	IDs      func() uuid.UUID
}

func (u AccountUseCase) CreateAccount(_ context.Context, _ dto.CreateAccountInput) (dto.AccountView, error) {
	return dto.AccountView{}, errNotImplemented
}

func (u AccountUseCase) SetAccountStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (dto.AccountView, error) {
	return dto.AccountView{}, errNotImplemented
}

func (u AccountUseCase) SoftDeleteAccount(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return errNotImplemented
}

func (u AccountUseCase) UpdateProfile(_ context.Context, _ uuid.UUID, _ dto.UpdateProfileInput) (dto.AccountView, error) {
	return dto.AccountView{}, errNotImplemented
}
