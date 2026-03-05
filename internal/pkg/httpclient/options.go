package httpclient

import "time"

// Option is a functional option for configuring baseClient.
type Option func(*baseClient)

// WithTimeout sets the per-request context timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *baseClient) { c.timeout = d }
}

// WithRetry configures the retry policy.
func WithRetry(cfg RetryConfig) Option {
	return func(c *baseClient) { c.retry = cfg }
}

// WithCircuitBreaker attaches a circuit breaker with the given config.
func WithCircuitBreaker(cfg CircuitBreakerConfig) Option {
	return func(c *baseClient) { c.cb = newCircuitBreaker(cfg) }
}
