package httpclient

import (
	"context"
	"errors"
	"fmt"

	"go-standard/internal/apperror"
)

// MapHTTPStatus maps an external HTTP response status code to an AppError.
func MapHTTPStatus(status int, body []byte, provider string) error {
	msg := fmt.Sprintf("%s: upstream error", provider)
	switch {
	case status == 400:
		return apperror.BadRequest(msg)
	case status == 401:
		return apperror.Unauthorized(msg)
	case status == 403:
		return apperror.Forbidden(msg)
	case status == 404:
		return apperror.NotFound(provider, "resource")
	case status == 409:
		return apperror.Conflict(msg)
	case status == 422:
		return apperror.Unprocessable(msg)
	case status == 429:
		return apperror.ServiceUnavailable(msg+" (rate limited)", fmt.Errorf("status 429: %s", string(body)))
	case status >= 500:
		return apperror.ServiceUnavailable(msg, fmt.Errorf("status %d: %s", status, string(body)))
	default:
		return apperror.Internal(msg, fmt.Errorf("unexpected status %d", status))
	}
}

// MapError maps a transport/network error to an AppError.
func MapError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return apperror.ServiceUnavailable("upstream timeout", err)
	}
	return apperror.ServiceUnavailable("upstream unreachable", err)
}
