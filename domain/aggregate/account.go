package aggregate

import (
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

// Account aggregate — behavior filled in GREEN (TASKDEMO-UM-002 T2).
type Account struct {
	id uuid.UUID
}

func NewAccount(id uuid.UUID, email, name, passwordHash string, role enum.Role, now time.Time) (*Account, error) {
	return nil, errAccountNotImplemented
}

func RestoreAccount(
	id uuid.UUID,
	email, name, passwordHash string,
	role enum.Role,
	status enum.AccountStatus,
	deletedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Account {
	return &Account{id: id}
}

func (a *Account) ID() uuid.UUID {
	return a.id
}

func (a *Account) Email() string {
	return ""
}

func (a *Account) Name() string {
	return ""
}

func (a *Account) PasswordHash() string {
	return ""
}

func (a *Account) Role() enum.Role {
	return ""
}

func (a *Account) Status() enum.AccountStatus {
	return ""
}

func (a *Account) DeletedAt() *time.Time {
	return nil
}

func (a *Account) IsSoftDeleted() bool {
	return false
}

func (a *Account) Deactivate(at time.Time) error {
	return errAccountNotImplemented
}

func (a *Account) Activate(at time.Time) error {
	return errAccountNotImplemented
}

func (a *Account) SoftDelete(at time.Time) error {
	return errAccountNotImplemented
}

func (a *Account) UpdateProfile(name, passwordHash string, at time.Time) error {
	return errAccountNotImplemented
}

var errAccountNotImplemented = &notImplementedError{}

type notImplementedError struct{}

func (e *notImplementedError) Error() string {
	return "account aggregate not implemented"
}
