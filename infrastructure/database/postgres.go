package database

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	Inner *pgxpool.Pool
}
