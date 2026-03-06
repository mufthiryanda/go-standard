package middleware

import (
	"bytes"
	"time"

	"go-standard/internal/domain/ctxkey"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func NewLogger(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Capture request body before Next() consumes it
		reqBody := string(c.Body())

		err := c.Next()

		status := c.Response().StatusCode()
		latencyMs := time.Since(start).Milliseconds()

		// Capture response body
		resBody := string(bytes.TrimSpace(c.Response().Body()))

		fields := []zap.Field{
			zap.String("request_id", requestIDFromLocals(c)),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Int64("latency_ms", latencyMs),
			zap.String("ip", c.IP()),
			zap.String("request_body", reqBody),
			zap.String("response_body", resBody),
		}

		if uid, ok := c.Locals(ctxkey.UserID).(uuid.UUID); ok {
			fields = append(fields, zap.String("user_id", uid.String()))
		}

		switch {
		case status >= 500:
			logger.Error("request completed", fields...)
		case status >= 400:
			logger.Warn("request completed", fields...)
		default:
			logger.Info("request completed", fields...)
		}

		return err
	}
}

func requestIDFromLocals(c *fiber.Ctx) string {
	if id, ok := c.Locals(ctxkey.RequestID).(string); ok {
		return id
	}
	return ""
}
