package response

import (
	"time"

	"go-standard/internal/domain/model"
)

// jakartaLoc is loaded once at package init via a var declaration (no init() used).
var jakartaLoc = mustLoadLocation("Asia/Jakarta")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}

// UserResponse is the API-facing shape for a user. Never expose raw DB models.
type UserResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Phone     *string `json:"phone,omitempty"`
	Role      string  `json:"role"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

const iso8601 = "2006-01-02T15:04:05Z07:00"

// NewUserResponse converts a domain User model into a UserResponse.
func NewUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Phone:     u.Phone,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.In(jakartaLoc).Format(iso8601),
		UpdatedAt: u.UpdatedAt.In(jakartaLoc).Format(iso8601),
	}
}

// NewUserListResponse converts a slice of domain users to response DTOs.
func NewUserListResponse(users []model.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, NewUserResponse(&users[i]))
	}
	return out
}
