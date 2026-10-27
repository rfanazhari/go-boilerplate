package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/infrastructure/config"
	"github.com/rfanazhari/go-boilerplate/infrastructure/runtime"
)

type envelope struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
	Error  map[string]any `json:"error"`
}

func TestExampleAPIFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg := config.Config{
		ServiceName:     "taskdemo",
		AppEnv:          "test",
		HTTPAddr:        "127.0.0.1:0",
		InfraOptional:   true,
		ShutdownTimeout: 5 * time.Second,
	}
	logApp := logmanager.NewApplication(logmanager.WithAppName("taskdemo"), logmanager.WithService("taskdemo"))
	rt, err := runtime.Start(ctx, cfg, logApp)
	require.NoError(t, err)
	defer func() {
		shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = rt.Stop(shutdownCtx)
	}()
	base := "http://" + rt.Addr()

	created := postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusCreated)
	id, _ := created.Data["id"].(string)
	require.NotEmpty(t, id)
	assert.Equal(t, "todo", created.Data["status"])

	started := postJSON(t, ctx, base+"/tasks/"+id+"/start", `{}`, http.StatusOK)
	assert.Equal(t, "in_progress", started.Data["status"])

	completed := postJSON(t, ctx, base+"/tasks/"+id+"/complete", `{}`, http.StatusOK)
	assert.Equal(t, "done", completed.Data["status"])

	got := getJSON(t, ctx, base+"/tasks/"+id, http.StatusOK)
	assert.Equal(t, "done", got.Data["status"])

	rejected := postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusCreated)
	todoID, _ := rejected.Data["id"].(string)
	fail := postJSON(t, ctx, base+"/tasks/"+todoID+"/complete", `{}`, http.StatusConflict)
	assert.Equal(t, "error", fail.Status)
	assert.Equal(t, "INVALID_TRANSITION", fail.Error["code"])
}

func postJSON(t *testing.T, ctx context.Context, url, body string, want int) envelope {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, want, resp.StatusCode)
	return decodeEnvelope(t, resp.Body)
}

func getJSON(t *testing.T, ctx context.Context, url string, want int) envelope {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
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
