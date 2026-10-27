package event

import (
	"time"

	"github.com/google/uuid"
)

type TaskCompleted struct {
	TaskID      uuid.UUID
	CompletedAt time.Time
}
