package httpclient

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func (c *baseClient) logRequest(req *http.Request) {
	c.logger.Info("httpclient: outbound request",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", sanitizeHeaders(req.Header)),
	)
}

func (c *baseClient) logResponse(req *http.Request, res *http.Response, latency time.Duration, err error) {
	status := 0
	if res != nil {
		status = res.StatusCode
	}
	fields := []zap.Field{
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Int("status", status),
		zap.Int64("latency_ms", latency.Milliseconds()),
	}
	if err != nil {
		c.logger.Error("httpclient: outbound response error", append(fields, zap.Error(err))...)
		return
	}
	c.logger.Info("httpclient: outbound response", fields...)
}
