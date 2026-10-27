package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type LoginUseCase struct {
	Accounts repository.AccountRepository
	Tokens   repository.TokenRepository
	Hasher   service.PasswordHasher
	TokenRaw func() ([]byte, error)
}

func (u LoginUseCase) Login(ctx context.Context, input dto.LoginInput) (dto.LoginResult, error) {
	account, err := u.Accounts.GetByEmail(ctx, input.Email)
	if err != nil {
		if isAccountNotFound(err) {
			return dto.LoginResult{}, invalidCredentials()
		}
		return dto.LoginResult{}, wrapInfra(err)
	}
	if account.IsSoftDeleted() || account.Status() != enum.AccountActive {
		return dto.LoginResult{}, invalidCredentials()
	}
	if err := u.Hasher.Compare(account.PasswordHash(), input.Password); err != nil {
		return dto.LoginResult{}, invalidCredentials()
	}
	raw, err := u.randomTokenBytes()
	if err != nil {
		return dto.LoginResult{}, wrapInfra(err)
	}
	token := hex.EncodeToString(raw)
	hash := HashToken(token)
	if err := u.Tokens.Save(ctx, account.ID(), hash); err != nil {
		return dto.LoginResult{}, wrapInfra(err)
	}
	return dto.LoginResult{Token: token, TokenType: "Bearer"}, nil
}

func (u LoginUseCase) randomTokenBytes() ([]byte, error) {
	if u.TokenRaw != nil {
		return u.TokenRaw()
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func invalidCredentials() error {
	return errs.New(errs.CodeInvalidCredentials, errs.KindUnauthenticated, errors.New("invalid credentials"))
}
