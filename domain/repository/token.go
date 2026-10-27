package repository

import (
	"context"

	"github.com/google/uuid"
)

type TokenRepository interface {
	Save(ctx context.Context, accountID uuid.UUID, tokenHash string) error
	GetByHash(ctx context.Context, tokenHash string) (uuid.UUID, error)
	DeleteByAccountID(ctx context.Context, accountID uuid.UUID) error
}
