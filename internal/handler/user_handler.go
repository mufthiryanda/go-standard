package handler

import (
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	"go-standard/internal/pkg/httputil"
	appvalidator "go-standard/internal/pkg/validator"
	"go-standard/internal/usecase"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UserHandler handles HTTP requests for the User resource.
type UserHandler struct {
	userUsecase usecase.UserUsecase
	validator   *govalidator.Validate
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(uc usecase.UserUsecase, v *govalidator.Validate) *UserHandler {
	return &UserHandler{userUsecase: uc, validator: v}
}

// Create handles POST /users — registers a new user.
func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req request.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if appErr := appvalidator.ValidateStruct(h.validator, req); appErr != nil {
		return appErr
	}

	res, err := h.userUsecase.Register(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.Success(res))
}

// GetByID handles GET /users/:id — fetches a single user.
func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	res, ucErr := h.userUsecase.GetByID(c.UserContext(), id)
	if ucErr != nil {
		return ucErr
	}

	return c.Status(fiber.StatusOK).JSON(response.Success(res))
}

// List handles GET /users — returns a paginated list of users.
func (h *UserHandler) List(c *fiber.Ctx) error {
	var f request.UserFilter
	if err := c.QueryParser(&f); err != nil {
		return fiber.ErrBadRequest
	}

	users, meta, err := h.userUsecase.GetAll(c.UserContext(), f)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.SuccessWithMeta(users, meta))
}

// Update handles PUT /users/:id — partially updates a user.
func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req request.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if appErr := appvalidator.ValidateStruct(h.validator, req); appErr != nil {
		return appErr
	}

	actorID := actorFromLocals(c)

	res, ucErr := h.userUsecase.Update(c.UserContext(), id, req, actorID)
	if ucErr != nil {
		return ucErr
	}

	return c.Status(fiber.StatusOK).JSON(response.Success(res))
}

// Delete handles DELETE /users/:id — soft-deletes a user.
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := httputil.ParseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	actorID := actorFromLocals(c)

	if ucErr := h.userUsecase.Delete(c.UserContext(), id, actorID); ucErr != nil {
		return ucErr
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// actorFromLocals extracts the authenticated user's UUID from Fiber locals.
// Falls back to uuid.Nil when the auth middleware is not applied.
func actorFromLocals(c *fiber.Ctx) uuid.UUID {
	if raw, ok := c.Locals("user_id").(string); ok {
		if id, err := uuid.Parse(raw); err == nil {
			return id
		}
	}
	return uuid.Nil
}
