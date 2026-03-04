package middleware

import (
	"strings"

	"go-standard/internal/apperror"
	"go-standard/internal/domain/ctxkey"
	"go-standard/internal/pkg/jwt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// NewAuth returns a Fiber middleware that validates JWT Bearer tokens.
// On, success it stores userID and role in Fiber Locals for downstream use.
func NewAuth(jwtMgr *jwt.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return apperror.Unauthorized("missing or invalid token")
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtMgr.ValidateToken(tokenStr)
		if err != nil {
			return err
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return apperror.Unauthorized("invalid token subject")
		}

		c.Locals(ctxkey.UserID, userID)
		c.Locals(ctxkey.Role, claims.Role)

		return c.Next()
	}
}
