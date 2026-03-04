package handler

import "github.com/gofiber/fiber/v2"

// registerUserRoutes mounts the User resource under the given router.
// GET list and GET :id are public; POST, PUT :id, DELETE :id require auth.
func registerUserRoutes(r fiber.Router, h *UserHandler, authMW fiber.Handler) {
	users := r.Group("/users")

	users.Get("/", h.List)
	users.Get("/:id", h.GetByID)

	users.Post("/", authMW, h.Create)
	users.Put("/:id", authMW, h.Update)
	users.Delete("/:id", authMW, h.Delete)
}
