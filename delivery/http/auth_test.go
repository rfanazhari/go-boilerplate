package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/application/usecase"
	httpapi "github.com/rfanazhari/go-boilerplate/delivery/http"
	"github.com/rfanazhari/go-boilerplate/domain/aggregate"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/infrastructure/i18n"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/accountrepository"
	taskrepository "github.com/rfanazhari/go-boilerplate/infrastructure/repository/taskrepository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/repository/tokenrepository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
	"github.com/rfanazhari/go-boilerplate/translations"
)

type testHasher struct{}

func (testHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

func (testHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return errPasswordMismatch
	}
	return nil
}

var errPasswordMismatch = io.EOF

func restoreAccount(id uuid.UUID, email, name, hash string, role enum.Role, status enum.AccountStatus, now time.Time) *aggregate.Account {
	return aggregate.RestoreAccount(id, email, name, hash, role, status, nil, now, now)
}

func testRouter(t *testing.T) (http.Handler, *accountrepository.Memory) {
	t.Helper()
	ctx := context.Background()
	bundle, err := i18n.Load(translations.Files)
	require.NoError(t, err)

	now := testdata.Instant()
	adminID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	accounts := &accountrepository.Memory{}
	tokens := &tokenrepository.Memory{}
	require.NoError(t, accounts.Save(ctx, restoreAccount(adminID, "admin@example.com", "Admin", "hash:admin-pass", enum.RoleAdmin, enum.AccountActive, now)))
	require.NoError(t, accounts.Save(ctx, restoreAccount(userID, "user@example.com", "User", "hash:user-pass", enum.RoleUser, enum.AccountActive, now)))

	hasher := testHasher{}
	loginUC := usecase.LoginUseCase{Accounts: accounts, Tokens: tokens, Hasher: hasher}
	accountUC := usecase.AccountUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   hasher,
		Clock:    testdata.NewClock(now),
		IDs:      uuid.New,
	}
	taskUC := usecase.TaskUseCase{
		Tasks: &taskrepository.Memory{},
		Clock: service.SystemClock{},
	}

	auth := httpapi.Auth{Accounts: accounts, Tokens: tokens}
	srv := httpapi.Server{
		Handler: httpapi.TaskHandler{UseCase: taskUC, Bundle: bundle},
		Login:   httpapi.LoginHandler{UseCase: loginUC, Bundle: bundle},
		Account: httpapi.AccountHandler{UseCase: accountUC, Bundle: bundle},
		Profile: httpapi.ProfileHandler{UseCase: accountUC, Bundle: bundle},
		Auth:    auth,
		Bundle:  bundle,
	}
	return srv.Router(), accounts
}

type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func doJSON(t *testing.T, handler http.Handler, method, path string, body any, bearer string) (int, envelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return rec.Code, env
}

func TestHTTPAuthRBAC(t *testing.T) {
	handler, _ := testRouter(t)

	t.Run("health without authorization", func(t *testing.T) {
		status, env := doJSON(t, handler, http.MethodGet, "/health", nil, "")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "success", env.Status)
	})

	t.Run("profile without token returns 401", func(t *testing.T) {
		status, env := doJSON(t, handler, http.MethodPatch, "/profile", map[string]string{"name": "New"}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		require.NotNil(t, env.Error)
		assert.Equal(t, errs.CodeUnauthenticated, env.Error.Code)
	})

	t.Run("user cannot create account", func(t *testing.T) {
		_, loginEnv := doJSON(t, handler, http.MethodPost, "/login", map[string]string{
			"email":    "user@example.com",
			"password": "user-pass",
		}, "")
		var loginData struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal(loginEnv.Data, &loginData))
		require.NotEmpty(t, loginData.Token)

		status, env := doJSON(t, handler, http.MethodPost, "/accounts", map[string]string{
			"email":    "new@example.com",
			"name":     "New",
			"password": "secret",
			"role":     "user",
		}, loginData.Token)
		assert.Equal(t, http.StatusForbidden, status)
		require.NotNil(t, env.Error)
		assert.Equal(t, errs.CodeForbidden, env.Error.Code)
	})

	t.Run("admin can create account", func(t *testing.T) {
		_, loginEnv := doJSON(t, handler, http.MethodPost, "/login", map[string]string{
			"email":    "admin@example.com",
			"password": "admin-pass",
		}, "")
		var loginData struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal(loginEnv.Data, &loginData))

		status, env := doJSON(t, handler, http.MethodPost, "/accounts", map[string]string{
			"email":    "created@example.com",
			"name":     "Created",
			"password": "secret",
			"role":     "user",
		}, loginData.Token)
		assert.Equal(t, http.StatusCreated, status)
		assert.Equal(t, "success", env.Status)
	})

	t.Run("tasks without token returns 401", func(t *testing.T) {
		status, env := doJSON(t, handler, http.MethodPost, "/tasks", map[string]string{"title": "x"}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		require.NotNil(t, env.Error)
		assert.Equal(t, errs.CodeUnauthenticated, env.Error.Code)
	})

	t.Run("admin cannot create task", func(t *testing.T) {
		_, loginEnv := doJSON(t, handler, http.MethodPost, "/login", map[string]string{
			"email":    "admin@example.com",
			"password": "admin-pass",
		}, "")
		var loginData struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal(loginEnv.Data, &loginData))
		status, env := doJSON(t, handler, http.MethodPost, "/tasks", map[string]string{"title": "x"}, loginData.Token)
		assert.Equal(t, http.StatusForbidden, status)
		require.NotNil(t, env.Error)
		assert.Equal(t, errs.CodeForbidden, env.Error.Code)
	})

	t.Run("login then bearer on profile", func(t *testing.T) {
		_, loginEnv := doJSON(t, handler, http.MethodPost, "/login", map[string]string{
			"email":    "user@example.com",
			"password": "user-pass",
		}, "")
		var loginData struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal(loginEnv.Data, &loginData))

		status, env := doJSON(t, handler, http.MethodPatch, "/profile", map[string]string{"name": "Renamed"}, loginData.Token)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "success", env.Status)
		var profile struct {
			Name string `json:"name"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &profile))
		assert.Equal(t, "Renamed", profile.Name)
	})
}
