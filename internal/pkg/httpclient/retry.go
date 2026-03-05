package httpclient

import (
	"math/rand"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// RetryConfig holds retry policy configuration.
type RetryConfig struct {
	MaxAttempts   int
	InitialWait   time.Duration
	MaxWait       time.Duration
	Multiplier    float64
	RetryOnStatus []int
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialWait:   200 * time.Millisecond,
		MaxWait:       2 * time.Second,
		Multiplier:    2.0,
		RetryOnStatus: []int{429, 502, 503, 504},
	}
}

func (c *baseClient) shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return false
	}
	for _, s := range c.retry.RetryOnStatus {
		if resp.StatusCode == s {
			return true
		}
	}
	return false
}

func (c *baseClient) computeWait(attempt int) time.Duration {
	wait := float64(c.retry.InitialWait)
	for i := 0; i < attempt; i++ {
		wait *= c.retry.Multiplier
	}
	if d := time.Duration(wait); d > c.retry.MaxWait {
		wait = float64(c.retry.MaxWait)
	}
	// ±10% jitter
	jitter := wait * 0.1
	wait += (rand.Float64()*2 - 1) * jitter //nolint:gosec
	return time.Duration(wait)
}

// executeWithResilience runs the request with retry logic and optional circuit breaker.
func (c *baseClient) executeWithResilience(req *http.Request) (*http.Response, error) {
	maxAttempts := c.retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	doRequest := func() (*http.Response, error) {
		if c.cb != nil {
			return c.cb.Execute(func() (*http.Response, error) {
				return c.httpClient.Do(req)
			})
		}
		return c.httpClient.Do(req)
	}

	var (
		resp *http.Response
		err  error
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err = doRequest()
		if attempt == maxAttempts-1 {
			break
		}
		if !c.shouldRetry(resp, err) {
			break
		}
		wait := c.computeWait(attempt)
		reason := "network error"
		if err == nil && resp != nil {
			reason = http.StatusText(resp.StatusCode)
		}
		c.logger.Warn("httpclient: retrying request",
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
			zap.String("reason", reason),
		)
		time.Sleep(wait)
	}
	return resp, err
}
