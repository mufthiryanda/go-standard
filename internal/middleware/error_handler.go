package middleware

import (
	"errors"
	"net/http"

	"go-standard/internal/apperror"
	"go-standard/internal/dto/response"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// NewErrorHandler returns a fiber.ErrorHandler that maps AppError and
// fiber.Error to structured JSON responses. Unknown errors produce 500.
// This function is passed to fiber.Config{ErrorHandler: ...}, not chained.
func NewErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			status := codeToStatus(appErr.Code)

			if appErr.Err != nil {
				zap.L().Error("internal error",
					zap.String("request_id", requestIDFromLocals(c)),
					zap.String("path", c.Path()),
					zap.String("code", string(appErr.Code)),
					zap.Error(appErr.Err),
				)
			}

			if len(appErr.Details) > 0 {
				return c.Status(status).JSON(
					response.ErrorWithDetails(string(appErr.Code), appErr.Message, appErr.Details),
				)
			}
			return c.Status(status).JSON(
				response.Error(string(appErr.Code), appErr.Message),
			)
		}

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return c.Status(fiberErr.Code).JSON(
				response.Error("HTTP_ERROR", fiberErr.Message),
			)
		}

		zap.L().Error("unexpected error",
			zap.String("request_id", requestIDFromLocals(c)),
			zap.String("path", c.Path()),
			zap.Error(err),
		)
		return c.Status(http.StatusInternalServerError).JSON(
			response.Error("INTERNAL_ERROR", "an unexpected error occurred"),
		)
	}
}

// codeToStatus maps AppError codes to HTTP status codes.
func codeToStatus(code apperror.Code) int {
	switch code {
	case apperror.CodeBadRequest:
		return http.StatusBadRequest
	case apperror.CodeUnauthorized:
		return http.StatusUnauthorized
	case apperror.CodeForbidden:
		return http.StatusForbidden
	case apperror.CodeNotFound:
		return http.StatusNotFound
	case apperror.CodeConflict:
		return http.StatusConflict
	case apperror.CodeUnprocessable:
		return http.StatusUnprocessableEntity
	case apperror.CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
