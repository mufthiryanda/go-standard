package model

import (
	"time"

	"github.com/google/uuid"
)

// User is the core domain model. No DB tags — infra concerns live in the repo layer.
type User struct {
	ID        uuid.UUID
	Email     string
	Password  string
	Name      string
	Phone     *string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// TableName tells GORM which table backs this model.
func (User) TableName() string {
	return "users"
}
