package httpclient

import (
	"net/http"
	"strings"
)

var sensitiveHeaders = map[string]bool{
	"authorization":   true,
	"x-api-key":       true,
	"x-client-secret": true,
	"x-signature":     true,
	"x-external-id":   true,
}

// sanitizeHeaders returns a copy of headers with sensitive values replaced by [REDACTED].
func sanitizeHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		lower := strings.ToLower(k)
		if sensitiveHeaders[lower] {
			result[k] = "[REDACTED]"
		} else if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}
