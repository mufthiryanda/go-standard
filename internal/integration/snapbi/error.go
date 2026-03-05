package snapbi

import (
	"encoding/json"
	"strconv"

	"go-standard/internal/apperror"
	"go-standard/internal/pkg/httpclient"
)

// ParseSnapBIError maps a SNAP BI error response body to an AppError.
// SNAP BI uses 7-digit response codes: first 3 digits = HTTP status.
func ParseSnapBIError(body []byte) error {
	var e ErrorResponse
	if err := json.Unmarshal(body, &e); err != nil {
		return apperror.Internal("snapbi: unreadable error response", err)
	}
	if len(e.ResponseCode) < 3 {
		return apperror.Internal("snapbi: malformed response code", nil)
	}
	httpStatus, err := strconv.Atoi(e.ResponseCode[:3])
	if err != nil {
		return apperror.Internal("snapbi: non-numeric response code", err)
	}
	return httpclient.MapHTTPStatus(httpStatus, body, "snapbi")
}
