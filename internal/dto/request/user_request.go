package request

import (
	"go-standard/internal/domain/model"
	"go-standard/internal/pkg/filter"

	"github.com/google/uuid"
)

// CreateUserRequest is the payload for user registration.
type CreateUserRequest struct {
	Email    string  `json:"email"    validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
	Name     string  `json:"name"     validate:"required,min=1,max=100"`
	Phone    *string `json:"phone"    validate:"omitempty,indonesian_phone"`
	Role     string  `json:"role"     validate:"required,oneof=admin user"`
}

// ToModel maps the request DTO to a domain model.
// Password should already be hashed before calling ToModel.
func (r *CreateUserRequest) ToModel(hashedPassword string) *model.User {
	return &model.User{
		ID:       uuid.New(),
		Email:    r.Email,
		Password: hashedPassword,
		Name:     r.Name,
		Phone:    r.Phone,
		Role:     r.Role,
	}
}

// UpdateUserRequest supports partial updates — only non-nil fields are applied.
type UpdateUserRequest struct {
	Name  *string `json:"name"  validate:"omitempty,min=1,max=100"`
	Phone *string `json:"phone" validate:"omitempty,indonesian_phone"`
	Role  *string `json:"role"  validate:"omitempty,oneof=admin user"`
}

// UserFilter is the query filter for listing users.
type UserFilter struct {
	filter.BaseFilter
	Email   *string `query:"email"`
	Name    *string `query:"name"`
	Role    *string `query:"role"`
	Phone   *string `query:"phone"`
	Keyword *string `query:"keyword"`
}
