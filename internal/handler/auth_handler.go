package handler

import (
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	appvalidator "go-standard/internal/pkg/validator"
	"go-standard/internal/usecase"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
	validator   *govalidator.Validate
}

func NewAuthHandler(uc usecase.AuthUsecase, v *govalidator.Validate) *AuthHandler {
	return &AuthHandler{authUsecase: uc, validator: v}
}

// Register handles POST /auth/register — creates a user and returns tokens.
//
//	@Summary		Register a new user
//	@Description	Creates a new user account and returns a JWT access/refresh token pair.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.RegisterRequest								true	"Register payload"
//	@Success		201		{object}	response.Response{data=response.AuthTokenResponse}	"Created"
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}			"Bad request / validation error"
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}			"Email already registered"
//	@Failure		429		{object}	response.Response{error=response.ErrorBody}			"Rate limit exceeded"
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}			"Internal server error"
//	@Router			/api/v1/auth/register [post]
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
//
//	@Summary		Login
//	@Description	Authenticates a user with email and password and returns a JWT token pair.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.LoginRequest								true	"Login payload"
//	@Success		200		{object}	response.Response{data=response.AuthTokenResponse}	"OK"
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}			"Bad request / validation error"
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}			"Invalid credentials"
//	@Failure		429		{object}	response.Response{error=response.ErrorBody}			"Rate limit exceeded"
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}			"Internal server error"
//	@Router			/api/v1/auth/login [post]
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
//
//	@Summary		Refresh tokens
//	@Description	Accepts a valid refresh token and returns a new JWT access/refresh token pair.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.RefreshRequest								true	"Refresh payload"
//	@Success		200		{object}	response.Response{data=response.AuthTokenResponse}	"OK"
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}			"Bad request / validation error"
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}			"Invalid or expired refresh token"
//	@Failure		429		{object}	response.Response{error=response.ErrorBody}			"Rate limit exceeded"
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}			"Internal server error"
//	@Router			/api/v1/auth/refresh [post]
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
