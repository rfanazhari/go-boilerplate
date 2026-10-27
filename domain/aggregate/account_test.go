package aggregate_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type accountNewCase struct {
	name          string
	email         string
	accountName   string
	passwordHash  string
	role          enum.Role
	now           time.Time
	wantErr       bool
	wantCode      string
	wantEmail     string
	wantName      string
}

func TestNewAccount(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	id := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []accountNewCase{
		{
			name:          "accept admin role with trimmed email",
			email:         "  ada@example.com  ",
			accountName:   "Ada",
			passwordHash:  "hash",
			role:          enum.RoleAdmin,
			now:           now,
			wantEmail:     "ada@example.com",
			wantName:      "Ada",
		},
		{
			name:          "accept user role",
			email:         "budi@example.com",
			accountName:   "Budi",
			passwordHash:  "hash",
			role:          enum.RoleUser,
			now:           now,
			wantEmail:     "budi@example.com",
			wantName:      "Budi",
		},
		{
			name:         "reject empty email",
			email:        "  ",
			accountName:  "Ada",
			passwordHash: "hash",
			role:         enum.RoleAdmin,
			now:          now,
			wantErr:      true,
			wantCode:     errs.CodeValidationFailed,
		},
		{
			name:         "reject empty name",
			email:        "ada@example.com",
			accountName:  " \t ",
			passwordHash: "hash",
			role:         enum.RoleAdmin,
			now:          now,
			wantErr:      true,
			wantCode:     errs.CodeValidationFailed,
		},
		{
			name:         "reject empty password hash",
			email:        "ada@example.com",
			accountName:  "Ada",
			passwordHash: "",
			role:         enum.RoleAdmin,
			now:          now,
			wantErr:      true,
			wantCode:     errs.CodeInvalidPassword,
		},
		{
			name:         "reject invalid role",
			email:        "ada@example.com",
			accountName:  "Ada",
			passwordHash: "hash",
			role:         enum.Role("superuser"),
			now:          now,
			wantErr:      true,
			wantCode:     errs.CodeInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := aggregate.NewAccount(id, tt.email, tt.accountName, tt.passwordHash, tt.role, tt.now)
			if tt.wantErr {
				require.Error(t, err)
				var appErr *errs.Error
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, tt.wantCode, appErr.Code)
				assert.Nil(t, account)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, account)
			assert.Equal(t, id, account.ID())
			assert.Equal(t, enum.AccountActive, account.Status())
			assert.False(t, account.IsSoftDeleted())
			assert.Equal(t, tt.wantEmail, account.Email())
			assert.Equal(t, tt.wantName, account.Name())
			assert.Equal(t, tt.role, account.Role())
		})
	}
}

func TestAccountLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	newActive := func(t *testing.T) *aggregate.Account {
		account, err := aggregate.NewAccount(
			uuid.New(),
			"citra@example.com",
			"Citra",
			"hash",
			enum.RoleUser,
			now,
		)
		require.NoError(t, err)
		return account
	}

	t.Run("deactivate active account", func(t *testing.T) {
		account := newActive(t)
		require.NoError(t, account.Deactivate(now.Add(time.Minute)))
		assert.Equal(t, enum.AccountNonActive, account.Status())
	})

	t.Run("reject deactivate when already non active", func(t *testing.T) {
		account := newActive(t)
		require.NoError(t, account.Deactivate(now))
		err := account.Deactivate(now.Add(time.Minute))
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidStatus, appErr.Code)
	})

	t.Run("activate non active account", func(t *testing.T) {
		account := newActive(t)
		require.NoError(t, account.Deactivate(now))
		require.NoError(t, account.Activate(now.Add(time.Hour)))
		assert.Equal(t, enum.AccountActive, account.Status())
	})

	t.Run("reject activate when already active", func(t *testing.T) {
		account := newActive(t)
		err := account.Activate(now)
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidStatus, appErr.Code)
	})

	t.Run("soft delete sets deleted at", func(t *testing.T) {
		account := newActive(t)
		at := now.Add(2 * time.Hour)
		require.NoError(t, account.SoftDelete(at))
		assert.True(t, account.IsSoftDeleted())
		require.NotNil(t, account.DeletedAt())
		assert.Equal(t, at.UTC().Truncate(time.Millisecond), *account.DeletedAt())
	})

	t.Run("reject lifecycle changes after soft delete", func(t *testing.T) {
		account := newActive(t)
		require.NoError(t, account.SoftDelete(now))
		err := account.Deactivate(now.Add(time.Minute))
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidStatus, appErr.Code)
	})
}

func TestAccountUpdateProfile(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	account, err := aggregate.NewAccount(
		uuid.New(),
		"budi@example.com",
		"Budi",
		"old-hash",
		enum.RoleUser,
		now,
	)
	require.NoError(t, err)

	t.Run("update name and password hash only", func(t *testing.T) {
		at := now.Add(time.Hour)
		require.NoError(t, account.UpdateProfile("Budi Wijaya", "new-hash", at))
		assert.Equal(t, "Budi Wijaya", account.Name())
		assert.Equal(t, "new-hash", account.PasswordHash())
		assert.Equal(t, enum.RoleUser, account.Role())
		assert.Equal(t, "budi@example.com", account.Email())
		assert.Equal(t, enum.AccountActive, account.Status())
	})

	t.Run("reject empty name", func(t *testing.T) {
		err := account.UpdateProfile("  ", "hash", now)
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeValidationFailed, appErr.Code)
	})

	t.Run("reject empty password hash", func(t *testing.T) {
		err := account.UpdateProfile("Budi", "", now)
		require.Error(t, err)
		var appErr *errs.Error
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errs.CodeInvalidPassword, appErr.Code)
	})
}
