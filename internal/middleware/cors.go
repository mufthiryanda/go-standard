package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// NewCORS returns a Fiber CORS middleware with permissive defaults.
// Origin allowlist should be tightened per environment in a future iteration.
func NewCORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:  "*",
		AllowMethods:  "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:  "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		ExposeHeaders: "X-Request-ID",
	})
}
