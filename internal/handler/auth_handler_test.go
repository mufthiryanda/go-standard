package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/dto/response"
	appvalidator "go-standard/internal/pkg/validator"
	"go-standard/mocks"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ─── Setup helpers ───────────────────────────────────────────────────────────

func setupAuthHandlerApp(t *testing.T) (*fiber.App, *mocks.MockAuthUsecase) {
	t.Helper()
	mockUC := mocks.NewMockAuthUsecase(t)
	v := appvalidator.New()
	h := NewAuthHandler(mockUC, v)

	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
		if ferr, ok := err.(*fiber.Error); ok {
			return c.Status(ferr.Code).JSON(fiber.Map{"error": ferr.Message})
		}
		if appErr, ok := err.(*apperror.AppError); ok {
			return c.Status(400).JSON(fiber.Map{"error": appErr.Message})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal"})
	}})

	app.Post("/auth/register", h.Register)
	app.Post("/auth/login", h.Login)
	app.Post("/auth/refresh", h.Refresh)

	return app, mockUC
}

func authJSONReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(method, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func sampleTokenResp() *response.AuthTokenResponse {
	r := response.NewAuthTokenResponse("access.token.here", "refresh.token.here", time.Now().Add(15*time.Minute))
	return &r
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestAuthHandler_Register_Success(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Register(mock.Anything, mock.Anything).Return(sampleTokenResp(), nil)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/register", map[string]any{
		"email": "bob@example.com", "password": "pass1234", "name": "Bob", "role": "user",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, res.StatusCode)
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader([]byte("bad-json")))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Register_ValidationFails_MissingName(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/register", map[string]any{
		"email": "bob@example.com", "password": "pass1234", "role": "user",
		// name is missing
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Register_ValidationFails_InvalidRole(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/register", map[string]any{
		"email": "bob@example.com", "password": "pass1234", "name": "Bob", "role": "superadmin",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Register_UsecaseConflict(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Register(mock.Anything, mock.Anything).
		Return(nil, apperror.Conflict("email already registered"))

	res, err := app.Test(authJSONReq(t, "POST", "/auth/register", map[string]any{
		"email": "bob@example.com", "password": "pass1234", "name": "Bob", "role": "user",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_Success(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Login(mock.Anything, mock.Anything).Return(sampleTokenResp(), nil)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/login", map[string]any{
		"email": "bob@example.com", "password": "pass1234",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
}

func TestAuthHandler_Login_InvalidBody(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Login_ValidationFails_MissingEmail(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/login", map[string]any{
		"password": "pass1234",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Login_ValidationFails_MissingPassword(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/login", map[string]any{
		"email": "bob@example.com",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Login_UsecaseUnauthorized(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Login(mock.Anything, mock.Anything).
		Return(nil, apperror.Unauthorized("invalid email or password"))

	res, err := app.Test(authJSONReq(t, "POST", "/auth/login", map[string]any{
		"email": "bob@example.com", "password": "wrongpass",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh_Success(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Refresh(mock.Anything, mock.Anything).Return(sampleTokenResp(), nil)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/refresh", map[string]any{
		"refresh_token": "some.refresh.token",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
}

func TestAuthHandler_Refresh_InvalidBody(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Refresh_ValidationFails_MissingToken(t *testing.T) {
	app, _ := setupAuthHandlerApp(t)

	res, err := app.Test(authJSONReq(t, "POST", "/auth/refresh", map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Refresh_UsecaseUnauthorized(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Refresh(mock.Anything, mock.Anything).
		Return(nil, apperror.Unauthorized("refresh token not found or expired"))

	res, err := app.Test(authJSONReq(t, "POST", "/auth/refresh", map[string]any{
		"refresh_token": "expired.token.here",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestAuthHandler_Refresh_UsecaseInternalError(t *testing.T) {
	app, mockUC := setupAuthHandlerApp(t)
	mockUC.EXPECT().Refresh(mock.Anything, mock.Anything).
		Return(nil, apperror.Internal("failed to generate token", nil))

	res, err := app.Test(authJSONReq(t, "POST", "/auth/refresh", map[string]any{
		"refresh_token": "some.token",
	}))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}
