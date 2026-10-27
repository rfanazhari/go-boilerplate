package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
)

type TaskRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*aggregate.Task, error)
	Save(ctx context.Context, task *aggregate.Task) error
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
