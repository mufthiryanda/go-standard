package response

import "go-standard/internal/apperror"

// Response is the standard API response envelope.
type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data"`
	Error   *ErrorBody `json:"error"`
	Meta    *Meta      `json:"meta"`
}

// ErrorBody carries error code, human message, and optional field details.
type ErrorBody struct {
	Code    string                `json:"code"`
	Message string                `json:"message"`
	Details []apperror.FieldError `json:"details,omitempty"`
}

// Meta carries pagination information.
type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// Success returns a successful envelope with data.
func Success(data any) Response {
	return Response{Success: true, Data: data}
}

// SuccessWithMeta returns a successful envelope with data and pagination meta.
func SuccessWithMeta(data any, meta *Meta) Response {
	return Response{Success: true, Data: data, Meta: meta}
}

// Error returns a failure envelope with code and message.
func Error(code, message string) Response {
	return Response{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message},
	}
}

// ErrorWithDetails returns a failure envelope with per-field validation details.
func ErrorWithDetails(code, message string, details []apperror.FieldError) Response {
	return Response{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message, Details: details},
	}
}
