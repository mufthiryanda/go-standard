package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	appvalidator "go-standard/internal/pkg/validator"
	"go-standard/mocks"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ─── Setup helpers ───────────────────────────────────────────────────────────

func setupUserHandlerApp(t *testing.T) (*fiber.App, *mocks.MockUserUsecase) {
	t.Helper()
	mockUC := mocks.NewMockUserUsecase(t)
	v := appvalidator.New()
	h := NewUserHandler(mockUC, v)

	testActorID := uuid.New().String()

	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
		var appErr *apperror.AppError
		if aerr, ok := err.(*apperror.AppError); ok {
			appErr = aerr
		}
		if appErr != nil {
			return c.Status(400).JSON(fiber.Map{"error": appErr.Message})
		}
		if ferr, ok := err.(*fiber.Error); ok {
			return c.Status(ferr.Code).JSON(fiber.Map{"error": ferr.Message})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal"})
	}})

	app.Post("/users", h.Create)
	app.Get("/users/:id", h.GetByID)
	app.Get("/users", h.List)
	app.Put("/users/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", testActorID)
		return c.Next()
	}, h.Update)
	app.Delete("/users/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", testActorID)
		return c.Next()
	}, h.Delete)

	return app, mockUC
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func makeReq(method, url string, body *bytes.Reader) *http.Request {
	if body == nil {
		req := httptest.NewRequest(method, url, nil)
		return req
	}
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func sampleUserResp() *response.UserResponse {
	id := uuid.New()
	return &response.UserResponse{
		ID: id.String(), Email: "alice@example.com",
		Name: "Alice", Role: "user",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestUserHandler_Create_Success(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	resp := sampleUserResp()
	mockUC.EXPECT().Register(mock.Anything, mock.Anything).Return(resp, nil)

	req := makeReq("POST", "/users", jsonBody(t, map[string]any{
		"email": "alice@example.com", "password": "pass1234", "name": "Alice", "role": "user",
	}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, res.StatusCode)
}

func TestUserHandler_Create_InvalidBody(t *testing.T) {
	app, _ := setupUserHandlerApp(t)

	req := httptest.NewRequest("POST", "/users", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_Create_ValidationFails_MissingEmail(t *testing.T) {
	app, _ := setupUserHandlerApp(t)

	req := makeReq("POST", "/users", jsonBody(t, map[string]any{
		"password": "pass1234", "name": "Alice", "role": "user",
	}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_Create_UsecaseConflict(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	mockUC.EXPECT().Register(mock.Anything, mock.Anything).
		Return(nil, apperror.Conflict("email already registered"))

	req := makeReq("POST", "/users", jsonBody(t, map[string]any{
		"email": "alice@example.com", "password": "pass1234", "name": "Alice", "role": "user",
	}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode) // ErrorHandler maps to 400 in test
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestUserHandler_GetByID_Success(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	id := uuid.New()
	resp := sampleUserResp()
	resp.ID = id.String()

	mockUC.EXPECT().GetByID(mock.Anything, id).Return(resp, nil)

	res, err := app.Test(makeReq("GET", "/users/"+id.String(), nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
}

func TestUserHandler_GetByID_InvalidUUID(t *testing.T) {
	app, _ := setupUserHandlerApp(t)

	res, err := app.Test(makeReq("GET", "/users/not-a-uuid", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	id := uuid.New()

	mockUC.EXPECT().GetByID(mock.Anything, id).Return(nil, apperror.NotFound("user", id.String()))

	res, err := app.Test(makeReq("GET", "/users/"+id.String(), nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

// ─── List ────────────────────────────────────────────────────────────────────

func TestUserHandler_List_Success(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	users := []response.UserResponse{*sampleUserResp()}
	meta := &response.Meta{Page: 1, PageSize: 20, TotalItems: 1, TotalPages: 1}

	mockUC.EXPECT().GetAll(mock.Anything, mock.Anything).Return(users, meta, nil)

	res, err := app.Test(makeReq("GET", "/users?page=1&page_size=20", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
}

func TestUserHandler_List_UsecaseError(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	mockUC.EXPECT().GetAll(mock.Anything, mock.Anything).
		Return(nil, nil, apperror.Internal("db error", nil))

	res, err := app.Test(makeReq("GET", "/users", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestUserHandler_Update_Success(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	id := uuid.New()
	resp := sampleUserResp()
	resp.ID = id.String()
	resp.Name = "Updated"

	mockUC.EXPECT().Update(mock.Anything, id, mock.Anything, mock.Anything).Return(resp, nil)

	req := makeReq("PUT", "/users/"+id.String(), jsonBody(t, map[string]any{"name": "Updated"}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
}

func TestUserHandler_Update_InvalidUUID(t *testing.T) {
	app, _ := setupUserHandlerApp(t)

	req := makeReq("PUT", "/users/bad-uuid", jsonBody(t, map[string]any{"name": "Updated"}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_Update_InvalidBody(t *testing.T) {
	app, _ := setupUserHandlerApp(t)
	id := uuid.New()

	req := httptest.NewRequest("PUT", "/users/"+id.String(), bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_Update_ValidationFails(t *testing.T) {
	app, _ := setupUserHandlerApp(t)
	id := uuid.New()

	// "role" with invalid value triggers validation failure
	req := makeReq("PUT", "/users/"+id.String(), jsonBody(t, request.UpdateUserRequest{
		Role: strPtr("superadmin"),
	}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_Update_UsecaseError(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	id := uuid.New()

	mockUC.EXPECT().Update(mock.Anything, id, mock.Anything, mock.Anything).
		Return(nil, apperror.NotFound("user", id.String()))

	req := makeReq("PUT", "/users/"+id.String(), jsonBody(t, map[string]any{"name": "X"}))
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

// ─── Delete ──────────────────────────────────────────────────────────────────

func TestUserHandler_Delete_Success(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	id := uuid.New()

	mockUC.EXPECT().Delete(mock.Anything, id, mock.Anything).Return(nil)

	res, err := app.Test(makeReq("DELETE", "/users/"+id.String(), nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, res.StatusCode)
}

func TestUserHandler_Delete_InvalidUUID(t *testing.T) {
	app, _ := setupUserHandlerApp(t)

	res, err := app.Test(makeReq("DELETE", "/users/not-a-uuid", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestUserHandler_Delete_UsecaseError(t *testing.T) {
	app, mockUC := setupUserHandlerApp(t)
	id := uuid.New()

	mockUC.EXPECT().Delete(mock.Anything, id, mock.Anything).
		Return(apperror.NotFound("user", id.String()))

	res, err := app.Test(makeReq("DELETE", "/users/"+id.String(), nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

// ─── helper ──────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
