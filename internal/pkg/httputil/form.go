package httputil

import (
	"mime/multipart"

	"go-standard/internal/apperror"

	"github.com/gofiber/fiber/v2"
)

// ParseFormWithFile parses a multipart/form-data request.
// Primitive fields are bound into dst via Fiber's BodyParser.
// The file identified by fileField is returned separately.
// fileField may be empty when no file upload is expected.
func ParseFormWithFile(c *fiber.Ctx, dst interface{}, fileField string) (*multipart.FileHeader, error) {
	if err := c.BodyParser(dst); err != nil {
		return nil, apperror.BadRequest("failed to parse form data: " + err.Error())
	}

	if fileField == "" {
		return nil, nil
	}

	fh, err := c.FormFile(fileField)
	if err != nil {
		// File is optional — callers decide whether nil is acceptable.
		return nil, nil
	}

	return fh, nil
}
