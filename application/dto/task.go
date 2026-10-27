package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
)

// TaskActor is the authenticated caller for task operations (identity from delivery).
type TaskActor struct {
	ID   uuid.UUID
	Role enum.Role
}

type CreateTaskInput struct {
	Title   string
	DueDate *time.Time
}

type RescheduleTaskInput struct {
	ID      uuid.UUID
	DueDate *time.Time
}

type TaskView struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Title       string
	Status      string
	DueDate     *time.Time
	Overdue     bool
	CreatedAt   time.Time
	CompletedAt *time.Time
}
