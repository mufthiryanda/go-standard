package middleware

import (
	"time"

	"go-standard/internal/domain/ctxkey"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NewLogger returns a Fiber middleware that logs each request with structured
// fields. Log level is chosen by HTTP status: info (2xx/3xx), warn (4xx),
// error (5xx).
func NewLogger(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		status := c.Response().StatusCode()
		latencyMs := time.Since(start).Milliseconds()

		fields := []zap.Field{
			zap.String("request_id", requestIDFromLocals(c)),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Int64("latency_ms", latencyMs),
			zap.String("ip", c.IP()),
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
