package apperror

import "fmt"

// Code represents a machine-readable error classification.
type Code string

const (
	CodeBadRequest         Code = "BAD_REQUEST"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeNotFound           Code = "NOT_FOUND"
	CodeConflict           Code = "CONFLICT"
	CodeUnprocessable      Code = "UNPROCESSABLE_ENTITY"
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
)

// FieldError represents a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is the application-wide error type.
// Err is internal and must never be sent to the client.
type AppError struct {
	Code    Code         `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
	Err     error        `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// BadRequest creates a 400-class AppError.
func BadRequest(msg string) *AppError {
	return &AppError{Code: CodeBadRequest, Message: msg}
}

// BadRequestWithDetails creates a 400-class AppError with per-field detail.
func BadRequestWithDetails(msg string, details []FieldError) *AppError {
	return &AppError{Code: CodeBadRequest, Message: msg, Details: details}
}

// NotFound creates a 404-class AppError with a standard message.
func NotFound(resource, id string) *AppError {
	return &AppError{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s with id %s not found", resource, id),
	}
}

// Unauthorized creates a 401-class AppError.
func Unauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg}
}

// Forbidden creates a 403-class AppError.
func Forbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg}
}

// Conflict creates a 409-class AppError.
func Conflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg}
}

// Unprocessable creates a 422-class AppError.
func Unprocessable(msg string) *AppError {
	return &AppError{Code: CodeUnprocessable, Message: msg}
}

// Internal creates a 500-class AppError wrapping an internal error.
func Internal(msg string, err error) *AppError {
	return &AppError{Code: CodeInternal, Message: msg, Err: err}
}

// ServiceUnavailable creates a 503-class AppError wrapping an internal error.
func ServiceUnavailable(msg string, err error) *AppError {
	return &AppError{Code: CodeServiceUnavailable, Message: msg, Err: err}
}
