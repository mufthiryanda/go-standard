package httpclient

import (
	"net/http"
	"time"

	"go-standard/internal/apperror"

	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	Name            string
	MaxFailures     int
	ResetTimeout    time.Duration
	HalfOpenMaxReqs int
}

type circuitBreaker struct {
	name string
	cb   *gobreaker.CircuitBreaker
}

func newCircuitBreaker(cfg CircuitBreakerConfig) *circuitBreaker {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: uint32(cfg.HalfOpenMaxReqs), //nolint:gosec
		Timeout:     cfg.ResetTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return int(counts.ConsecutiveFailures) >= cfg.MaxFailures
		},
	}
	return &circuitBreaker{
		name: cfg.Name,
		cb:   gobreaker.NewCircuitBreaker(settings),
	}
}

// Execute runs fn through the circuit breaker.
func (c *circuitBreaker) Execute(fn func() (*http.Response, error)) (*http.Response, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, apperror.ServiceUnavailable("circuit open: "+c.name, err)
		}
		return nil, err
	}
	return result.(*http.Response), nil
}
