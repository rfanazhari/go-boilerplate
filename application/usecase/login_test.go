package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/accountrepository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/tokenrepository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/mocks"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

func newLoginUseCase(t *testing.T, accounts *mocks.MockAccountRepository, tokens *mocks.MockTokenRepository) usecase.LoginUseCase {
	t.Helper()
	return usecase.LoginUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   fakeHasher{},
		TokenRaw: func() ([]byte, error) {
			return []byte("fixed-32-byte-opaque-token!!!!"), nil
		},
	}
}

func TestLoginUseCaseLogin(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	accountID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")

	t.Run("success stores hash not raw token", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepository(t)
		tokens := mocks.NewMockTokenRepository(t)
		uc := newLoginUseCase(t, accounts, tokens)

		account := restoreAccount(accountID, "user@example.com", "User", "hash:secret", enum.RoleUser, enum.AccountActive, now)
		accounts.EXPECT().GetByEmail(mock.Anything, "user@example.com").Return(account, nil)
		var savedHash string
		tokens.EXPECT().Save(mock.Anything, accountID, mock.Anything).Run(func(_ context.Context, _ uuid.UUID, tokenHash string) {
			savedHash = tokenHash
		}).Return(nil)

		result, err := uc.Login(ctx, dto.LoginInput{Email: "user@example.com", Password: "secret"})
		require.NoError(t, err)
		assert.NotEmpty(t, result.Token)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.NotEqual(t, result.Token, savedHash)
		assert.NotEmpty(t, savedHash)
	})

	t.Run("wrong password", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepository(t)
		tokens := mocks.NewMockTokenRepository(t)
		uc := newLoginUseCase(t, accounts, tokens)

		account := restoreAccount(accountID, "user@example.com", "User", "hash:secret", enum.RoleUser, enum.AccountActive, now)
		accounts.EXPECT().GetByEmail(mock.Anything, "user@example.com").Return(account, nil)

		_, err := uc.Login(ctx, dto.LoginInput{Email: "user@example.com", Password: "wrong"})
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidCredentials, appErr.Code)
		assert.Equal(t, errs.KindUnauthenticated, appErr.Kind)
	})

	t.Run("inactive account", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepository(t)
		tokens := mocks.NewMockTokenRepository(t)
		uc := newLoginUseCase(t, accounts, tokens)

		account := restoreAccount(accountID, "user@example.com", "User", "hash:secret", enum.RoleUser, enum.AccountNonActive, now)
		accounts.EXPECT().GetByEmail(mock.Anything, "user@example.com").Return(account, nil)

		_, err := uc.Login(ctx, dto.LoginInput{Email: "user@example.com", Password: "secret"})
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidCredentials, appErr.Code)
	})

	t.Run("missing account", func(t *testing.T) {
		accounts := mocks.NewMockAccountRepository(t)
		tokens := mocks.NewMockTokenRepository(t)
		uc := newLoginUseCase(t, accounts, tokens)

		accounts.EXPECT().GetByEmail(mock.Anything, "gone@example.com").Return(nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, errors.New("missing")))

		_, err := uc.Login(ctx, dto.LoginInput{Email: "gone@example.com", Password: "secret"})
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidCredentials, appErr.Code)
	})
}

func TestLoginUseCaseRevokeOnDeactivate(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	clock := testdata.NewClock(now)
	actorID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	targetID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	accounts := &accountrepository.Memory{}
	tokens := &tokenrepository.Memory{}
	require.NoError(t, accounts.Save(ctx, restoreAccount(targetID, "user@example.com", "User", "hash:secret", enum.RoleUser, enum.AccountActive, now)))

	loginUC := usecase.LoginUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   fakeHasher{},
		TokenRaw: func() ([]byte, error) {
			return []byte("session-token-for-revoke-test!!"), nil
		},
	}
	result, err := loginUC.Login(ctx, dto.LoginInput{Email: "user@example.com", Password: "secret"})
	require.NoError(t, err)

	accountUC := usecase.AccountUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   fakeHasher{},
		Clock:    clock,
	}
	_, err = accountUC.SetAccountStatus(ctx, actorID, targetID, "non_active")
	require.NoError(t, err)

	_, err = tokens.GetByHash(ctx, usecase.HashToken(result.Token))
	require.Error(t, err)
	var appErr *errs.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, errs.CodeUnauthenticated, appErr.Code)
}
