package middleware

import (
	"time"

	"go-standard/internal/config"
	"go-standard/internal/dto/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberredis "github.com/gofiber/storage/redis/v3"
	"github.com/redis/go-redis/v9"
)

// NewRateLimiter returns a Fiber rate-limit middleware backed by Redis.
// cfg selects the policy (Default or Auth). On limit hit it responds with
// 429 + Retry-After header and a standard error envelope.
func NewRateLimiter(rdb *redis.Client, cfg config.RateLimitPolicy) fiber.Handler {
	window := parseWindow(cfg.Window)

	storage := fiberredis.New(fiberredis.Config{
		URL: redisURL(rdb),
	})

	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: window,
		Storage:    storage,
		LimitReached: func(c *fiber.Ctx) error {
			retryAfter := int(window.Seconds())
			c.Set("Retry-After", time.Duration(retryAfter).String())
			return c.Status(fiber.StatusTooManyRequests).JSON(
				response.Error("TOO_MANY_REQUESTS", "rate limit exceeded"),
			)
		},
	})
}

// parseWindow converts a duration string (e.g. "1m") to time.Duration.
// Falls back to 1 minute on parse failure.
func parseWindow(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return time.Minute
	}
	return d
}

// redisURL builds a redis:// URL from the existing client options.
func redisURL(rdb *redis.Client) string {
	opt := rdb.Options()
	if opt.Password != "" {
		return "redis://:" + opt.Password + "@" + opt.Addr
	}
	return "redis://" + opt.Addr
}
