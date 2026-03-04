package handler

import (
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	appvalidator "go-standard/internal/pkg/validator"
	"go-standard/internal/usecase"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	authUsecase usecase.AuthUsecase
	validator   *govalidator.Validate
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(uc usecase.AuthUsecase, v *govalidator.Validate) *AuthHandler {
	return &AuthHandler{authUsecase: uc, validator: v}
}

// Register handles POST /auth/register — creates a user and returns tokens.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req request.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if appErr := appvalidator.ValidateStruct(h.validator, req); appErr != nil {
		return appErr
	}

	res, err := h.authUsecase.Register(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.Success(res))
}

// Login handles POST /auth/login — authenticates and returns tokens.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req request.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if appErr := appvalidator.ValidateStruct(h.validator, req); appErr != nil {
		return appErr
	}

	res, err := h.authUsecase.Login(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.Success(res))
}

// Refresh handles POST /auth/refresh — rotates token pair.
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req request.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if appErr := appvalidator.ValidateStruct(h.validator, req); appErr != nil {
		return appErr
	}

	res, err := h.authUsecase.Refresh(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.Success(res))
}
