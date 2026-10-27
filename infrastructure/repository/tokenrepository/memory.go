package tokenrepository

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Memory struct {
	mu     sync.Mutex
	byHash map[string]uuid.UUID
}

func (m *Memory) ensure() {
	if m.byHash == nil {
		m.byHash = map[string]uuid.UUID{}
	}
}

func (m *Memory) Save(_ context.Context, accountID uuid.UUID, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	m.byHash[tokenHash] = accountID
	return nil
}

func (m *Memory) GetByHash(_ context.Context, tokenHash string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	accountID, ok := m.byHash[tokenHash]
	if !ok {
		return uuid.Nil, errs.New(errs.CodeUnauthenticated, errs.KindUnauthenticated, errors.New("token not found"))
	}
	return accountID, nil
}

func (m *Memory) DeleteByAccountID(_ context.Context, accountID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure()
	for hash, id := range m.byHash {
		if id == accountID {
			delete(m.byHash, hash)
		}
	}
	return nil
}
