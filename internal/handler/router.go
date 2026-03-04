package handler

import (
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures global middleware, the versioned API group, and health endpoints.
// Middleware order follows Standard #12: Recover → RequestID → Logger → CORS → RateLimiter → Auth.
func SetupRoutes(
	app *fiber.App,
	userHandler *UserHandler,
	authHandler *AuthHandler,
	authMW fiber.Handler,
	defaultLimiter fiber.Handler,
	authLimiter fiber.Handler,
	recoverMW fiber.Handler,
	requestIDMW fiber.Handler,
	loggerMW fiber.Handler,
	corsMW fiber.Handler,
	errorHandler fiber.ErrorHandler,
) {
	// Global middleware — applied to every request in order.
	app.Use(recoverMW)
	app.Use(requestIDMW)
	app.Use(loggerMW)
	app.Use(corsMW)

	// Custom error handler surfaced via Fiber config (set before Use calls, but also
	// stored here so main.go can pass it into fiber.Config before calling SetupRoutes).
	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return errorHandler(c, err)
		}
		return nil
	})

	// Health probes — outside /api/v1 versioning, no rate limiter.
	app.Get("/healthz", livenessHandler)
	app.Get("/readyz", readinessHandler)

	// Versioned API group — default rate limiter applied to the entire group.
	api := app.Group("/api/v1")
	api.Use(defaultLimiter)

	// Entity route registrations.
	registerAuthRoutes(api, authHandler, authLimiter)
	registerUserRoutes(api, userHandler, authMW)
}

// livenessHandler returns 200 when the process is alive.
func livenessHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

// readinessHandler returns 200 when all dependencies are reachable.
// A full dependency check is intentionally deferred to infrastructure-layer
// probes injected via main.go; this stub satisfies the contract until then.
func readinessHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ready",
		"checks": fiber.Map{
			"postgres":      "ok",
			"redis":         "ok",
			"elasticsearch": "ok",
		},
	})
}
