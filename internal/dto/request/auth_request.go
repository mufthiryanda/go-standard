package request

import (
	"go-standard/internal/domain/model"

	"github.com/google/uuid"
)

// RegisterRequest is the payload for auth registration (same fields as CreateUserRequest).
type RegisterRequest struct {
	Email    string  `json:"email"    validate:"required,email"                    example:"john.doe@example.com"`
	Password string  `json:"password" validate:"required,min=8,max=72"             example:"P@ssw0rd123"`
	Name     string  `json:"name"     validate:"required,min=1,max=100"            example:"John Doe"`
	Phone    *string `json:"phone"    validate:"omitempty,indonesian_phone"        example:"+6281234567890"`
	Role     string  `json:"role"     validate:"required,oneof=admin user"         example:"user"`
}

// ToModel maps the request DTO to a domain model.
// Password should already be hashed before calling ToModel.
func (r *RegisterRequest) ToModel(hashedPassword string) *model.User {
	return &model.User{
		ID:       uuid.New(),
		Email:    r.Email,
		Password: hashedPassword,
		Name:     r.Name,
		Phone:    r.Phone,
		Role:     r.Role,
	}
}

// LoginRequest is the payload for authentication.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email" example:"john.doe@example.com"`
	Password string `json:"password" validate:"required"       example:"P@ssw0rd123"`
}

// RefreshRequest is the payload for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
