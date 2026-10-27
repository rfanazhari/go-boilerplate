package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateAccountInput struct {
	Email    string
	Name     string
	Password string
	Role     string
}

type UpdateProfileInput struct {
	Name     *string
	Password *string
}

type AccountView struct {
	ID        uuid.UUID
	Email     string
	Name      string
	Role      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
