package smoke_test

import (
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

func TestHealthSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	require.NotEmpty(t, rt.Addr())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+rt.Addr()+"/health", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "success", payload["status"])
	_, hasData := payload["data"]
	assert.True(t, hasData)
	_, hasError := payload["error"]
	assert.False(t, hasError)

	shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	require.NoError(t, rt.Stop(shutdownCtx))
}
