package httputil

import (
	"go-standard/internal/apperror"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ParseUUIDParam extracts and parses a UUID path parameter by name.
// Returns apperror.BadRequest when the value is missing or not a valid UUID.
func ParseUUIDParam(c *fiber.Ctx, param string) (uuid.UUID, error) {
	raw := c.Params(param)
	if raw == "" {
		return uuid.Nil, apperror.BadRequest("missing path parameter: " + param)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperror.BadRequest("invalid UUID for parameter: " + param)
	}

	return id, nil
}
