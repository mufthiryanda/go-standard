package request

import (
	"go-standard/internal/domain/model"
	"go-standard/internal/pkg/filter"

	"github.com/google/uuid"
)

// CreateUserRequest is the payload for user registration.
type CreateUserRequest struct {
	Email    string  `json:"email"    validate:"required,email"             example:"john.doe@example.com"`
	Password string  `json:"password" validate:"required,min=8,max=72"      example:"P@ssw0rd123"`
	Name     string  `json:"name"     validate:"required,min=1,max=100"     example:"John Doe"`
	Phone    *string `json:"phone"    validate:"omitempty,indonesian_phone" example:"+6281234567890"`
	Role     string  `json:"role"     validate:"required,oneof=admin user"  example:"user"`
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
	Name  *string `json:"name"  validate:"omitempty,min=1,max=100"     example:"Jane Doe"`
	Phone *string `json:"phone" validate:"omitempty,indonesian_phone"  example:"+6289876543210"`
	Role  *string `json:"role"  validate:"omitempty,oneof=admin user"  example:"admin"`
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
