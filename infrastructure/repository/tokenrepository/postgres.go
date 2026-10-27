package tokenrepository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/pgxtx"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Postgres struct {
	Pool *pgxpool.Pool
}

func (r Postgres) querier(ctx context.Context) querier {
	if tx, ok := pgxtx.From(ctx); ok {
		return tx
	}
	return r.Pool
}

func (r Postgres) Save(ctx context.Context, accountID uuid.UUID, tokenHash string) error {
	const q = `
INSERT INTO credential_tokens (id, account_id, token_hash, created_at)
VALUES ($1, $2, $3, $4)`
	_, err := r.querier(ctx).Exec(ctx, q, uuid.New(), accountID, tokenHash, time.Now().UTC())
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}

func (r Postgres) GetByHash(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	const q = `
SELECT account_id
FROM credential_tokens
WHERE token_hash = $1`
	var accountID uuid.UUID
	if err := r.querier(ctx).QueryRow(ctx, q, tokenHash).Scan(&accountID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, errs.New(errs.CodeUnauthenticated, errs.KindUnauthenticated, errors.New("token not found"))
		}
		return uuid.Nil, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return accountID, nil
}

func (r Postgres) DeleteByAccountID(ctx context.Context, accountID uuid.UUID) error {
	const q = `DELETE FROM credential_tokens WHERE account_id = $1`
	_, err := r.querier(ctx).Exec(ctx, q, accountID)
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}
