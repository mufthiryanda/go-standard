package rediskey

import (
	"fmt"

	"github.com/google/uuid"
)

// UserDetail returns the Redis key for a cached user detail record.
func UserDetail(id uuid.UUID) string {
	return fmt.Sprintf("user:detail:%s", id.String())
}

// UserSession returns the Redis key for a user session.
func UserSession(userID uuid.UUID) string {
	return fmt.Sprintf("user:session:%s", userID.String())
}

// AuthRefresh returns the Redis key for a stored refresh token.
func AuthRefresh(userID uuid.UUID) string {
	return fmt.Sprintf("auth:refresh:%s", userID.String())
}
