package dto

import (
	"time"

	"github.com/google/uuid"
)

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
	Title       string
	Status      string
	DueDate     *time.Time
	Overdue     bool
	CreatedAt   time.Time
	CompletedAt *time.Time
}
