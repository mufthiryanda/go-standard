package infrastructure

import (
	"context"
	"fmt"

	"go-standard/internal/apperror"
	"go-standard/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewRedisClient creates a go-redis/v9 client, verifies connectivity via
// PING, and returns a cleanup function. Satisfies the Wire (T, func(), error)
// cleanup pattern.
func NewRedisClient(cfg *config.Config) (*redis.Client, func(), error) {
	opts := buildRedisOptions(cfg.Redis)

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, apperror.ServiceUnavailable("redis: ping failed", err)
	}

	zap.L().Info("redis: connection established",
		zap.String("host", cfg.Redis.Host),
		zap.Int("port", cfg.Redis.Port),
		zap.Int("db", cfg.Redis.DB),
	)

	cleanup := func() {
		if err := client.Close(); err != nil {
			zap.L().Error("redis: error closing client", zap.Error(err))
			return
		}
		zap.L().Info("redis: connection closed")
	}

	return client, cleanup, nil
}

// buildRedisOptions maps config fields to redis.Options, applying safe
// defaults for zero values.
func buildRedisOptions(cfg config.Redis) *redis.Options {
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = 10
	}

	minIdle := cfg.MinIdleConn
	if minIdle <= 0 {
		minIdle = 5
	}

	return &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
	}
}
