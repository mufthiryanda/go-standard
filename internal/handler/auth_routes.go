package handler

import "github.com/gofiber/fiber/v2"

// registerAuthRoutes mounts auth endpoints under /auth with the auth-specific rate limiter.
// All routes are public — no auth middleware required.
func registerAuthRoutes(r fiber.Router, h *AuthHandler, authLimiter fiber.Handler) {
	auth := r.Group("/auth")
	auth.Use(authLimiter)

	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
	auth.Post("/refresh", h.Refresh)
}
