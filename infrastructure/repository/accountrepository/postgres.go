package accountrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
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

func (r Postgres) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	txCtx := pgxtx.With(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}

func (r Postgres) Save(ctx context.Context, account *aggregate.Account) error {
	now := time.Now().UTC()
	const q = `
INSERT INTO accounts (id, email, name, password_hash, role, status, deleted_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
	email = EXCLUDED.email,
	name = EXCLUDED.name,
	password_hash = EXCLUDED.password_hash,
	role = EXCLUDED.role,
	status = EXCLUDED.status,
	deleted_at = EXCLUDED.deleted_at,
	updated_at = EXCLUDED.updated_at`
	_, err := r.querier(ctx).Exec(
		ctx,
		q,
		account.ID(),
		account.Email(),
		account.Name(),
		account.PasswordHash(),
		string(account.Role()),
		string(account.Status()),
		account.DeletedAt(),
		account.CreatedAt(),
		now,
	)
	if err != nil {
		return mapPostgresSaveError(err)
	}
	return nil
}

func (r Postgres) GetByID(ctx context.Context, id uuid.UUID) (*aggregate.Account, error) {
	const q = `
SELECT id, email, name, password_hash, role, status, deleted_at, created_at, updated_at
FROM accounts
WHERE id = $1`
	return r.scanAccount(r.querier(ctx).QueryRow(ctx, q, id))
}

func (r Postgres) GetByEmail(ctx context.Context, email string) (*aggregate.Account, error) {
	const q = `
SELECT id, email, name, password_hash, role, status, deleted_at, created_at, updated_at
FROM accounts
WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`
	return r.scanAccount(r.querier(ctx).QueryRow(ctx, q, strings.TrimSpace(email)))
}

func (r Postgres) CountLiveAdmins(ctx context.Context) (int, error) {
	const q = `
SELECT COUNT(*)
FROM accounts
WHERE deleted_at IS NULL AND role = 'admin'`
	var count int
	if err := r.querier(ctx).QueryRow(ctx, q).Scan(&count); err != nil {
		return 0, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return count, nil
}

func (r Postgres) scanAccount(row pgx.Row) (*aggregate.Account, error) {
	var (
		id           uuid.UUID
		email        string
		name         string
		passwordHash string
		role         string
		status       string
		deletedAt    *time.Time
		createdAt    time.Time
		updatedAt    time.Time
	)
	if err := row.Scan(&id, &email, &name, &passwordHash, &role, &status, &deletedAt, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.New(errs.CodeAccountNotFound, errs.KindNotFound, fmt.Errorf("account not found"))
		}
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	parsedRole, ok := enum.ParseRole(role)
	if !ok {
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, fmt.Errorf("invalid stored role %q", role))
	}
	parsedStatus, ok := enum.ParseAccountStatus(status)
	if !ok {
		return nil, errs.New(errs.CodeInternalError, errs.KindInternal, fmt.Errorf("invalid stored status %q", status))
	}
	return aggregate.RestoreAccount(id, email, name, passwordHash, parsedRole, parsedStatus, deletedAt, createdAt, updatedAt), nil
}

func mapPostgresSaveError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_live_uidx" {
		return errs.New(errs.CodeEmailTaken, errs.KindConflict, errors.New("email already taken"))
	}
	return errs.New(errs.CodeInternalError, errs.KindInternal, err)
}
