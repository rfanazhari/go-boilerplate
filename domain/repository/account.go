package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
)

type AccountRepository interface {
	Save(ctx context.Context, account *aggregate.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*aggregate.Account, error)
	GetByEmail(ctx context.Context, email string) (*aggregate.Account, error)
	CountLiveAdmins(ctx context.Context) (int, error)
}
