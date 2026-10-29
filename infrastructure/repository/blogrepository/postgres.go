package blogrepository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
)

type Postgres struct{ Pool *pgxpool.Pool }

func (r Postgres) Create(context.Context, *aggregate.Blog) error                    { return nil }
func (r Postgres) GetForUpdate(context.Context, uuid.UUID) (*aggregate.Blog, error) { return nil, nil }
