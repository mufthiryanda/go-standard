package middleware

import (
	"fmt"
	"runtime/debug"

	"go-standard/internal/dto/response"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// NewRecover returns a Fiber middleware that catches panics, logs them with
// a stack trace, and responds with a generic 500 error envelope.
func NewRecover(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.String("request_id", requestIDFromLocals(c)),
					zap.String("panic", fmt.Sprintf("%v", r)),
					zap.String("stack", string(debug.Stack())),
				)

				err = c.Status(fiber.StatusInternalServerError).JSON(
					response.Error("INTERNAL_ERROR", "something went wrong"),
				)
			}
		}()

		return c.Next()
	}
}
