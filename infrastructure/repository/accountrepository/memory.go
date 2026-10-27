package accountrepository

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Memory struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]*aggregate.Account
}

func (m *Memory) ensure() {
	if m.accounts == nil {
		m.accounts = map[uuid.UUID]*aggregate.Account{}
	}
}

func (m *Memory) clone(account *aggregate.Account) *aggregate.Account {
	return aggregate.RestoreAccount(
		account.ID(),
		account.Email(),
		account.Name(),
		account.PasswordHash(),
		account.Role(),
		account.Status(),
		account.DeletedAt(),
		account.CreatedAt(),
		account.UpdatedAt(),
	)
}

func (m *Memory) GetByID(_ context.Context, id uuid.UUID) (*aggregate.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	account, ok := m.accounts[id]
	if !ok {
		return nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, errors.New("account not found"))
	}
	return m.clone(account), nil
}

func (m *Memory) GetByEmail(_ context.Context, email string) (*aggregate.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	needle := strings.ToLower(strings.TrimSpace(email))
	for _, account := range m.accounts {
		if account.IsSoftDeleted() {
			continue
		}
		if strings.EqualFold(account.Email(), needle) {
			return m.clone(account), nil
		}
	}
	return nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, errors.New("account not found"))
}

func (m *Memory) CountLiveAdmins(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	count := 0
	for _, account := range m.accounts {
		if account.IsSoftDeleted() {
			continue
		}
		if account.Role() == enum.RoleAdmin {
			count++
		}
	}
	return count, nil
}

func (m *Memory) Save(_ context.Context, account *aggregate.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	if !account.IsSoftDeleted() {
		for _, existing := range m.accounts {
			if existing.IsSoftDeleted() {
				continue
			}
			if existing.ID() != account.ID() && strings.EqualFold(existing.Email(), account.Email()) {
				return errs.New(errs.CodeEmailTaken, errs.KindConflict, errors.New("email already taken"))
			}
		}
	}
	m.accounts[account.ID()] = m.clone(account)
	return nil
}

func (m *Memory) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
