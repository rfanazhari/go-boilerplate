package aggregate

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var (
	errEmptyEmail        = errors.New("email is empty")
	errEmptyAccountName  = errors.New("name is empty")
	errEmptyPasswordHash = errors.New("password hash is empty")
	errInvalidRole       = errors.New("invalid account role")
	errInvalidStatus     = errors.New("invalid account status transition")
)

type Account struct {
	id           uuid.UUID
	email        string
	name         string
	passwordHash string
	role         enum.Role
	status       enum.AccountStatus
	deletedAt    *time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

func NewAccount(id uuid.UUID, email, name, passwordHash string, role enum.Role, now time.Time) (*Account, error) {
	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return nil, errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, errEmptyEmail)
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, errEmptyAccountName)
	}
	if passwordHash == "" {
		return nil, errs.New(errs.CodeInvalidPassword, errs.KindInvalidInput, errEmptyPasswordHash)
	}
	if _, ok := enum.ParseRole(string(role)); !ok {
		return nil, errs.New(errs.CodeInvalidRole, errs.KindInvalidInput, errInvalidRole)
	}
	stamp := normalizeInstant(now)
	return &Account{
		id:           id,
		email:        trimmedEmail,
		name:         trimmedName,
		passwordHash: passwordHash,
		role:         role,
		status:       enum.AccountActive,
		createdAt:    stamp,
		updatedAt:    stamp,
	}, nil
}

func RestoreAccount(
	id uuid.UUID,
	email, name, passwordHash string,
	role enum.Role,
	status enum.AccountStatus,
	deletedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Account {
	return &Account{
		id:           id,
		email:        email,
		name:         name,
		passwordHash: passwordHash,
		role:         role,
		status:       status,
		deletedAt:    cloneTime(deletedAt),
		createdAt:    normalizeInstant(createdAt),
		updatedAt:    normalizeInstant(updatedAt),
	}
}

func (a *Account) ID() uuid.UUID {
	return a.id
}

func (a *Account) Email() string {
	return a.email
}

func (a *Account) Name() string {
	return a.name
}

func (a *Account) PasswordHash() string {
	return a.passwordHash
}

func (a *Account) Role() enum.Role {
	return a.role
}

func (a *Account) Status() enum.AccountStatus {
	return a.status
}

func (a *Account) DeletedAt() *time.Time {
	return cloneTime(a.deletedAt)
}

func (a *Account) IsSoftDeleted() bool {
	return a.deletedAt != nil
}

func (a *Account) Deactivate(at time.Time) error {
	if err := a.requireMutable(); err != nil {
		return err
	}
	if a.status != enum.AccountActive {
		return errs.New(errs.CodeInvalidStatus, errs.KindConflict, errInvalidStatus)
	}
	a.status = enum.AccountNonActive
	a.updatedAt = normalizeInstant(at)
	return nil
}

func (a *Account) Activate(at time.Time) error {
	if err := a.requireMutable(); err != nil {
		return err
	}
	if a.status != enum.AccountNonActive {
		return errs.New(errs.CodeInvalidStatus, errs.KindConflict, errInvalidStatus)
	}
	a.status = enum.AccountActive
	a.updatedAt = normalizeInstant(at)
	return nil
}

func (a *Account) SoftDelete(at time.Time) error {
	if a.deletedAt != nil {
		return errs.New(errs.CodeInvalidStatus, errs.KindConflict, errInvalidStatus)
	}
	stamp := normalizeInstant(at)
	a.deletedAt = &stamp
	a.updatedAt = stamp
	return nil
}

func (a *Account) UpdateProfile(name, passwordHash string, at time.Time) error {
	if err := a.requireMutable(); err != nil {
		return err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, errEmptyAccountName)
	}
	if passwordHash == "" {
		return errs.New(errs.CodeInvalidPassword, errs.KindInvalidInput, errEmptyPasswordHash)
	}
	a.name = trimmedName
	a.passwordHash = passwordHash
	a.updatedAt = normalizeInstant(at)
	return nil
}

func (a *Account) requireMutable() error {
	if a.deletedAt != nil {
		return errs.New(errs.CodeInvalidStatus, errs.KindConflict, errInvalidStatus)
	}
	return nil
}
