package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
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
	infsvc "github.com/rfanazhari/go-boilerplate/infrastructure/service"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
	"github.com/rfanazhari/go-boilerplate/translations"
)

type envelope struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
	Error  map[string]any `json:"error"`
}

type smokeHasher struct{}

func (smokeHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

func (smokeHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return io.EOF
	}
	return nil
}

func restoreSmokeAccount(id uuid.UUID, email, name, hash string, role enum.Role) *aggregate.Account {
	now := testdata.Instant()
	return aggregate.RestoreAccount(id, email, name, hash, role, enum.AccountActive, nil, now, now)
}

func startSmokeAPI(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	bundle, err := i18n.Load(translations.Files)
	require.NoError(t, err)

	adminID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	accounts := &accountrepository.Memory{}
	tokens := &tokenrepository.Memory{}
	require.NoError(t, accounts.Save(ctx, restoreSmokeAccount(adminID, "admin@example.com", "Admin", "hash:admin-pass", enum.RoleAdmin)))
	require.NoError(t, accounts.Save(ctx, restoreSmokeAccount(userID, "user@example.com", "User", "hash:user-pass", enum.RoleUser)))

	hasher := smokeHasher{}
	loginUC := usecase.LoginUseCase{Accounts: accounts, Tokens: tokens, Hasher: hasher}
	accountUC := usecase.AccountUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   hasher,
		Clock:    testdata.NewClock(testdata.Instant()),
		IDs:      uuid.New,
	}
	taskUC := usecase.TaskUseCase{
		Tasks:     &taskrepository.Memory{},
		Publisher: &infsvc.RecordingPublisher{},
		Clock:     service.SystemClock{},
	}
	auth := httpapi.Auth{Accounts: accounts, Tokens: tokens}
	handler := httpapi.Server{
		Handler: httpapi.TaskHandler{UseCase: taskUC, Bundle: bundle},
		Login:   httpapi.LoginHandler{UseCase: loginUC, Bundle: bundle},
		Account: httpapi.AccountHandler{UseCase: accountUC, Bundle: bundle},
		Profile: httpapi.ProfileHandler{UseCase: accountUC, Bundle: bundle},
		Auth:    auth,
		Bundle:  bundle,
	}.Router()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	return "http://" + ln.Addr().String()
}

func loginToken(t *testing.T, ctx context.Context, base, email, password string) string {
	t.Helper()
	env := postJSON(t, ctx, base+"/login", `{"email":"`+email+`","password":"`+password+`"}`, http.StatusOK)
	token, _ := env.Data["token"].(string)
	require.NotEmpty(t, token)
	return token
}

func TestExampleAPIFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	base := startSmokeAPI(t)

	t.Run("tasks without token returns 401", func(t *testing.T) {
		postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusUnauthorized)
	})

	userToken := loginToken(t, ctx, base, "user@example.com", "user-pass")

	created := postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusCreated, userToken)
	id, _ := created.Data["id"].(string)
	require.NotEmpty(t, id)
	assert.Equal(t, "todo", created.Data["status"])
	ownerID, _ := created.Data["ownerId"].(string)
	assert.Equal(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", ownerID)

	started := postJSON(t, ctx, base+"/tasks/"+id+"/start", `{}`, http.StatusOK, userToken)
	assert.Equal(t, "in_progress", started.Data["status"])

	completed := postJSON(t, ctx, base+"/tasks/"+id+"/complete", `{}`, http.StatusOK, userToken)
	assert.Equal(t, "done", completed.Data["status"])

	got := getJSON(t, ctx, base+"/tasks/"+id, http.StatusOK, userToken)
	assert.Equal(t, "done", got.Data["status"])
	assert.Equal(t, ownerID, got.Data["ownerId"])

	rejected := postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusCreated, userToken)
	todoID, _ := rejected.Data["id"].(string)
	fail := postJSON(t, ctx, base+"/tasks/"+todoID+"/complete", `{}`, http.StatusConflict, userToken)
	assert.Equal(t, "error", fail.Status)
	assert.Equal(t, "INVALID_TRANSITION", fail.Error["code"])

	adminToken := loginToken(t, ctx, base, "admin@example.com", "admin-pass")
	postJSON(t, ctx, base+"/tasks", `{"title":"Admin task"}`, http.StatusForbidden, adminToken)
}

func postJSON(t *testing.T, ctx context.Context, url, body string, want int, bearer ...string) envelope {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if len(bearer) > 0 && bearer[0] != "" {
		req.Header.Set("Authorization", "Bearer "+bearer[0])
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, want, resp.StatusCode)
	return decodeEnvelope(t, resp.Body)
}

func getJSON(t *testing.T, ctx context.Context, url string, want int, bearer ...string) envelope {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	if len(bearer) > 0 && bearer[0] != "" {
		req.Header.Set("Authorization", "Bearer "+bearer[0])
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, want, resp.StatusCode)
	return decodeEnvelope(t, resp.Body)
}

func decodeEnvelope(t *testing.T, body io.Reader) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.NewDecoder(body).Decode(&env))
	return env
}
