package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskRow struct {
	ID          uuid.UUID
	Title       string
	Status      string
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}
