package usecase

import (
	"context"
	"errors"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
)

var errLoginNotImplemented = errors.New("login not implemented")

type LoginUseCase struct {
	Accounts repository.AccountRepository
	Tokens   repository.TokenRepository
	Hasher   service.PasswordHasher
	TokenRaw func() ([]byte, error)
}

func (LoginUseCase) Login(_ context.Context, _ dto.LoginInput) (dto.LoginResult, error) {
	return dto.LoginResult{}, errLoginNotImplemented
}

func HashToken(_ string) string {
	return ""
}
