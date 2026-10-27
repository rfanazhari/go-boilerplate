package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/rfanazhari/go-boilerplate/infrastructure/config"
)

// BootstrapAdminID is the fixed account id inserted by the accounts migration for task owner backfill.
var BootstrapAdminID = uuid.MustParse("e0000000-0000-4000-8000-000000000001")

const bcryptCost = 12

// SeedAdmin reconciles the bootstrap admin from env or inserts the first live admin when none exist.
func SeedAdmin(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) error {
	if pool == nil {
		return nil
	}
	email := strings.TrimSpace(cfg.SeedAdminEmail)
	name := strings.TrimSpace(cfg.SeedAdminName)
	if email == "" || name == "" || cfg.SeedAdminPassword == "" {
		return fmt.Errorf("SEED_ADMIN_EMAIL, SEED_ADMIN_NAME, and SEED_ADMIN_PASSWORD are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedAdminPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("seed admin password hash: %w", err)
	}
	now := time.Now().UTC()

	tag, err := pool.Exec(ctx, `
		UPDATE accounts
		SET email = $2, name = $3, password_hash = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`, BootstrapAdminID, email, name, string(hash), now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	var liveAdmins int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounts WHERE deleted_at IS NULL AND role = 'admin'
	`).Scan(&liveAdmins); err != nil {
		return err
	}
	if liveAdmins > 0 {
		return nil
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO accounts (
			id, email, name, password_hash, role, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'admin', 'active', $5, $5)
	`, uuid.New(), email, name, string(hash), now)
	if err != nil {
		return err
	}
	return nil
}
