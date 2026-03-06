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

type UserHandler struct {
	userUsecase usecase.UserUsecase
	validator   *govalidator.Validate
}

func NewUserHandler(uc usecase.UserUsecase, v *govalidator.Validate) *UserHandler {
	return &UserHandler{userUsecase: uc, validator: v}
}

// Create handles POST /users — registers a new user.
//
//	@Summary		Create a user
//	@Description	Creates a new user record. Requires Bearer JWT.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.CreateUserRequest							true	"Create user payload"
//	@Success		201		{object}	response.Response{data=response.UserResponse}		"Created"
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}			"Bad request / validation error"
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}			"Unauthorized"
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}			"Email already exists"
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}			"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users [post]
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
//
//	@Summary		Get user by ID
//	@Description	Returns a single user by their UUID.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string										true	"User UUID"
//	@Success		200	{object}	response.Response{data=response.UserResponse}	"OK"
//	@Failure		400	{object}	response.Response{error=response.ErrorBody}		"Invalid UUID"
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}		"User not found"
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}		"Internal server error"
//	@Router			/api/v1/users/{id} [get]
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
//
//	@Summary		List users
//	@Description	Returns a paginated, filterable list of users.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number"				default(1)
//	@Param			page_size	query		int		false	"Items per page"			default(20)
//	@Param			sort_by		query		string	false	"Field to sort by"
//	@Param			sort_order	query		string	false	"Sort direction (asc/desc)"	Enums(asc, desc)
//	@Param			email		query		string	false	"Filter by email"
//	@Param			name		query		string	false	"Filter by name"
//	@Param			role		query		string	false	"Filter by role"
//	@Param			phone		query		string	false	"Filter by phone"
//	@Param			keyword		query		string	false	"Full-text keyword search"
//	@Success		200	{object}	response.Response{data=[]response.UserResponse,meta=response.Meta}	"OK"
//	@Failure		400	{object}	response.Response{error=response.ErrorBody}							"Bad request"
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}							"Internal server error"
//	@Router			/api/v1/users [get]
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
//
//	@Summary		Update a user
//	@Description	Partially updates a user's profile fields. Requires Bearer JWT.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string										true	"User UUID"
//	@Param			request	body		request.UpdateUserRequest					true	"Update user payload"
//	@Success		200		{object}	response.Response{data=response.UserResponse}	"OK"
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}		"Bad request / validation error"
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}		"Unauthorized"
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}		"User not found"
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id} [put]
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
//
//	@Summary		Delete a user
//	@Description	Soft-deletes a user by UUID. Requires Bearer JWT.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"User UUID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.Response{error=response.ErrorBody}	"Invalid UUID"
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}	"Unauthorized"
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}	"User not found"
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id} [delete]
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
