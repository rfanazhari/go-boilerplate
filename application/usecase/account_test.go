package usecase_test

import (
	"context"
	"errors"
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
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/mocks"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (fakeHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return errors.New("password mismatch")
	}
	return nil
}

func newAccountUseCase(t *testing.T, clock *testdata.Clock) (usecase.AccountUseCase, *mocks.MockAccountRepository, uuid.UUID) {
	t.Helper()
	accounts := mocks.NewMockAccountRepository(t)
	id := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	uc := usecase.AccountUseCase{
		Accounts: accounts,
		Hasher:   fakeHasher{},
		Clock:    clock,
		IDs:      func() uuid.UUID { return id },
	}
	return uc, accounts, id
}

func restoreAccount(id uuid.UUID, email, name, hash string, role enum.Role, status enum.AccountStatus, now time.Time) *aggregate.Account {
	return aggregate.RestoreAccount(id, email, name, hash, role, status, nil, now, now)
}

func TestAccountUseCaseCreateAccount(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	clock := testdata.NewClock(now)

	t.Run("add account", func(t *testing.T) {
		uc, accounts, newID := newAccountUseCase(t, clock)
		accounts.EXPECT().GetByEmail(mock.Anything, "ada@example.com").Return(nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, errors.New("missing")))
		accounts.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)

		view, err := uc.CreateAccount(ctx, dto.CreateAccountInput{
			Email:    "ada@example.com",
			Name:     "Ada",
			Password: "secret",
			Role:     "user",
		})
		require.NoError(t, err)
		assert.Equal(t, newID, view.ID)
		assert.Equal(t, "ada@example.com", view.Email)
		assert.Equal(t, "user", view.Role)
		assert.Equal(t, string(enum.AccountActive), view.Status)
	})

	t.Run("duplicate live email", func(t *testing.T) {
		uc, accounts, _ := newAccountUseCase(t, clock)
		existing := restoreAccount(uuid.New(), "ada@example.com", "Ada", "hash", enum.RoleUser, enum.AccountActive, now)
		accounts.EXPECT().GetByEmail(mock.Anything, "ada@example.com").Return(existing, nil)

		_, err := uc.CreateAccount(ctx, dto.CreateAccountInput{
			Email:    "ada@example.com",
			Name:     "Other",
			Password: "secret",
			Role:     "user",
		})
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeEmailTaken, appErr.Code)
		assert.Equal(t, errs.KindConflict, appErr.Kind)
	})

	t.Run("email reusable after soft delete", func(t *testing.T) {
		uc, accounts, newID := newAccountUseCase(t, clock)
		accounts.EXPECT().GetByEmail(mock.Anything, "reuse@example.com").Return(nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, errors.New("missing")))
		accounts.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)

		view, err := uc.CreateAccount(ctx, dto.CreateAccountInput{
			Email:    "reuse@example.com",
			Name:     "New",
			Password: "secret",
			Role:     "user",
		})
		require.NoError(t, err)
		assert.Equal(t, newID, view.ID)
	})
}

func TestAccountUseCaseAdminLifecycleGuards(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	clock := testdata.NewClock(now)
	actorID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	targetID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	t.Run("deactivate last admin", func(t *testing.T) {
		uc, accounts, _ := newAccountUseCase(t, clock)
		admin := restoreAccount(targetID, "admin@example.com", "Admin", "hash", enum.RoleAdmin, enum.AccountActive, now)
		accounts.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(func(txCtx context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
		accounts.EXPECT().GetByID(mock.Anything, targetID).Return(admin, nil)
		accounts.EXPECT().CountLiveAdmins(mock.Anything).Return(1, nil)

		_, err := uc.SetAccountStatus(ctx, actorID, targetID, "non_active")
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeLastAdmin, appErr.Code)
	})

	t.Run("deactivate self", func(t *testing.T) {
		uc, accounts, _ := newAccountUseCase(t, clock)
		admin := restoreAccount(actorID, "admin@example.com", "Admin", "hash", enum.RoleAdmin, enum.AccountActive, now)
		accounts.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(func(txCtx context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
		accounts.EXPECT().GetByID(mock.Anything, actorID).Return(admin, nil)

		_, err := uc.SetAccountStatus(ctx, actorID, actorID, "non_active")
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeLastAdmin, appErr.Code)
	})

	t.Run("soft delete self", func(t *testing.T) {
		uc, accounts, _ := newAccountUseCase(t, clock)
		admin := restoreAccount(actorID, "admin@example.com", "Admin", "hash", enum.RoleAdmin, enum.AccountActive, now)
		accounts.EXPECT().RunInTransaction(mock.Anything, mock.Anything).RunAndReturn(func(txCtx context.Context, fn func(context.Context) error) error {
			return fn(txCtx)
		})
		accounts.EXPECT().GetByID(mock.Anything, actorID).Return(admin, nil)

		err := uc.SoftDeleteAccount(ctx, actorID, actorID)
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeLastAdmin, appErr.Code)
	})
}

func TestAccountUseCaseUpdateProfile(t *testing.T) {
	ctx := context.Background()
	now := testdata.Instant()
	clock := testdata.NewClock(now)
	accountID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	t.Run("updates name and password hash only", func(t *testing.T) {
		uc, accounts, _ := newAccountUseCase(t, clock)
		account := restoreAccount(accountID, "user@example.com", "Before", "hash:old", enum.RoleUser, enum.AccountActive, now)
		accounts.EXPECT().GetByID(mock.Anything, accountID).Return(account, nil)
		accounts.EXPECT().Save(mock.Anything, mock.MatchedBy(func(saved *aggregate.Account) bool {
			return saved.Email() == "user@example.com" &&
				saved.Role() == enum.RoleUser &&
				saved.Status() == enum.AccountActive &&
				saved.Name() == "After" &&
				saved.PasswordHash() == "hash:new"
		})).Return(nil)

		name := "After"
		password := "new"
		view, err := uc.UpdateProfile(ctx, accountID, dto.UpdateProfileInput{Name: &name, Password: &password})
		require.NoError(t, err)
		assert.Equal(t, "After", view.Name)
		assert.Equal(t, "user@example.com", view.Email)
	})
}
