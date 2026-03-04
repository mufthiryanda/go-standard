package middleware

import (
	"go-standard/internal/domain/ctxkey"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// NewRequestID generates a UUID per request, stores it in Fiber Locals,
// and sets the X-Request-ID response header.
func NewRequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := uuid.New().String()
		c.Locals(ctxkey.RequestID, id)
		c.Set("X-Request-ID", id)
		return c.Next()
	}
}
