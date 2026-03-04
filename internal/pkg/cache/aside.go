package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetOrLoad implements the cache-aside pattern for any JSON-serializable type.
// On a cache hit the value is unmarshalled and returned immediately.
// On a miss, loader is called, the result is stored with ttl, then returned.
func GetOrLoad[T any](
	ctx context.Context,
	rdb *redis.Client,
	key string,
	ttl time.Duration,
	loader func() (T, error),
) (T, error) {
	var zero T

	raw, err := rdb.Get(ctx, key).Bytes()
	if err != nil && !errors.Is(err, redis.Nil) {
		return zero, err
	}

	if err == nil {
		var val T
		if jsonErr := json.Unmarshal(raw, &val); jsonErr != nil {
			return zero, jsonErr
		}
		return val, nil
	}

	// Cache miss — call loader.
	val, err := loader()
	if err != nil {
		return zero, err
	}

	b, err := json.Marshal(val)
	if err != nil {
		return zero, err
	}

	// Best-effort SET — ignore error so a Redis hiccup doesn't block the response.
	_ = rdb.Set(ctx, key, b, ttl).Err()

	return val, nil
}
