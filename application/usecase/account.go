package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

const maxPasswordBytes = 72

type AccountUseCase struct {
	Accounts repository.AccountRepository
	Tokens   repository.TokenRepository
	Hasher   service.PasswordHasher
	Clock    service.Clock
	IDs      func() uuid.UUID
}

func (u AccountUseCase) now() time.Time {
	if u.Clock == nil {
		return time.Now().UTC().Truncate(time.Millisecond)
	}
	return u.Clock.Now().UTC().Truncate(time.Millisecond)
}

func (u AccountUseCase) nextID() uuid.UUID {
	if u.IDs == nil {
		return uuid.New()
	}
	return u.IDs()
}

func accountView(account *aggregate.Account) dto.AccountView {
	return dto.AccountView{
		ID:        account.ID(),
		Email:     account.Email(),
		Name:      account.Name(),
		Role:      string(account.Role()),
		Status:    string(account.Status()),
		CreatedAt: account.CreatedAt(),
		UpdatedAt: account.UpdatedAt(),
	}
}

func (u AccountUseCase) CreateAccount(ctx context.Context, input dto.CreateAccountInput) (dto.AccountView, error) {
	role, ok := enum.ParseRole(input.Role)
	if !ok {
		return dto.AccountView{}, errs.New(errs.CodeInvalidRole, errs.KindInvalidInput, errors.New("invalid account role"))
	}
	if err := validatePassword(input.Password); err != nil {
		return dto.AccountView{}, err
	}
	existing, err := u.Accounts.GetByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return dto.AccountView{}, errs.New(errs.CodeEmailTaken, errs.KindConflict, errors.New("email already taken"))
	}
	if err != nil && !isAccountNotFound(err) {
		return dto.AccountView{}, wrapInfra(err)
	}
	hash, err := u.Hasher.Hash(input.Password)
	if err != nil {
		return dto.AccountView{}, wrapInfra(err)
	}
	now := u.now()
	account, err := aggregate.NewAccount(u.nextID(), input.Email, input.Name, hash, role, now)
	if err != nil {
		return dto.AccountView{}, err
	}
	if err := u.Accounts.Save(ctx, account); err != nil {
		return dto.AccountView{}, mapAccountSaveError(err)
	}
	return accountView(account), nil
}

func (u AccountUseCase) SetAccountStatus(ctx context.Context, actorID, accountID uuid.UUID, status string) (dto.AccountView, error) {
	parsed, ok := enum.ParseAccountStatus(status)
	if !ok {
		return dto.AccountView{}, errs.New(errs.CodeInvalidStatus, errs.KindInvalidInput, errors.New("invalid account status"))
	}
	var result dto.AccountView
	err := u.Accounts.RunInTransaction(ctx, func(txCtx context.Context) error {
		account, err := u.loadLiveAccount(txCtx, accountID)
		if err != nil {
			return err
		}
		if err := u.guardAdminTarget(txCtx, actorID, account); err != nil {
			return err
		}
		now := u.now()
		switch parsed {
		case enum.AccountActive:
			if err := account.Activate(now); err != nil {
				return err
			}
		case enum.AccountNonActive:
			if err := account.Deactivate(now); err != nil {
				return err
			}
		}
		if err := u.Accounts.Save(txCtx, account); err != nil {
			return mapAccountSaveError(err)
		}
		if parsed == enum.AccountNonActive && u.Tokens != nil {
			if err := u.Tokens.DeleteByAccountID(txCtx, account.ID()); err != nil {
				return wrapInfra(err)
			}
		}
		result = accountView(account)
		return nil
	})
	if err != nil {
		return dto.AccountView{}, err
	}
	return result, nil
}

func (u AccountUseCase) SoftDeleteAccount(ctx context.Context, actorID, accountID uuid.UUID) error {
	return u.Accounts.RunInTransaction(ctx, func(txCtx context.Context) error {
		account, err := u.loadLiveAccount(txCtx, accountID)
		if err != nil {
			return err
		}
		if err := u.guardAdminTarget(txCtx, actorID, account); err != nil {
			return err
		}
		if err := account.SoftDelete(u.now()); err != nil {
			return err
		}
		if err := mapAccountSaveError(u.Accounts.Save(txCtx, account)); err != nil {
			return err
		}
		if u.Tokens != nil {
			return wrapInfra(u.Tokens.DeleteByAccountID(txCtx, account.ID()))
		}
		return nil
	})
}

func (u AccountUseCase) UpdateProfile(ctx context.Context, accountID uuid.UUID, input dto.UpdateProfileInput) (dto.AccountView, error) {
	if input.Name == nil && input.Password == nil {
		return dto.AccountView{}, errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, errors.New("profile update requires name or password"))
	}
	account, err := u.loadLiveAccount(ctx, accountID)
	if err != nil {
		return dto.AccountView{}, err
	}
	name := account.Name()
	if input.Name != nil {
		name = *input.Name
	}
	hash := account.PasswordHash()
	if input.Password != nil {
		if err := validatePassword(*input.Password); err != nil {
			return dto.AccountView{}, err
		}
		hash, err = u.Hasher.Hash(*input.Password)
		if err != nil {
			return dto.AccountView{}, wrapInfra(err)
		}
	}
	if err := account.UpdateProfile(name, hash, u.now()); err != nil {
		return dto.AccountView{}, err
	}
	if err := u.Accounts.Save(ctx, account); err != nil {
		return dto.AccountView{}, mapAccountSaveError(err)
	}
	return accountView(account), nil
}

func (u AccountUseCase) loadLiveAccount(ctx context.Context, id uuid.UUID) (*aggregate.Account, error) {
	account, err := u.Accounts.GetByID(ctx, id)
	if err != nil {
		if isAccountNotFound(err) {
			return nil, err
		}
		return nil, wrapInfra(err)
	}
	if account.IsSoftDeleted() {
		return nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, errors.New("account not found"))
	}
	return account, nil
}

func (u AccountUseCase) guardAdminTarget(ctx context.Context, actorID uuid.UUID, target *aggregate.Account) error {
	if actorID == target.ID() {
		return errs.New(errs.CodeLastAdmin, errs.KindConflict, errors.New("cannot target own admin account"))
	}
	if target.Role() != enum.RoleAdmin {
		return nil
	}
	count, err := u.Accounts.CountLiveAdmins(ctx)
	if err != nil {
		return wrapInfra(err)
	}
	if count <= 1 {
		return errs.New(errs.CodeLastAdmin, errs.KindConflict, errors.New("last live admin"))
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return errs.New(errs.CodeInvalidPassword, errs.KindInvalidInput, errors.New("password is empty"))
	}
	if len(password) > maxPasswordBytes {
		return errs.New(errs.CodeInvalidPassword, errs.KindInvalidInput, errors.New("password exceeds maximum length"))
	}
	return nil
}

func isAccountNotFound(err error) bool {
	var appErr *errs.Error
	if errors.As(err, &appErr) {
		return appErr.Code == errs.CodeAccountNotFound
	}
	return false
}

func mapAccountSaveError(err error) error {
	var appErr *errs.Error
	if errors.As(err, &appErr) {
		if appErr.Code == errs.CodeEmailTaken {
			return appErr
		}
		return err
	}
	return wrapInfra(err)
}
