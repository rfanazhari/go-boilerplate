package service

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainevent "github.com/rfanazhari/go-boilerplate/domain/event"
)

func TestTaskCompletedMessageJSONKeys(t *testing.T) {
	completed := domainevent.TaskCompleted{
		TaskID:      uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		CompletedAt: time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
	}
	body, err := json.Marshal(taskCompletedMessage{
		TaskID:      completed.TaskID.String(),
		CompletedAt: completed.CompletedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	})
	require.NoError(t, err)

	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &keys))
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	sort.Strings(names)
	assert.Equal(t, []string{"completedAt", "taskId"}, names)
}
