package taskqueue

import (
	"context"

	"go-standard/internal/apperror"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Enqueuer is the interface usecases depend on for task enqueuing.
// Wraps asynq.Client so usecases are not coupled to the asynq package.
type Enqueuer interface {
	Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// AsynqEnqueuer is the concrete asynq-backed Enqueuer.
// Exported so Wire can bind it to the Enqueuer interface.
type AsynqEnqueuer struct {
	client *asynq.Client
	logger *zap.Logger
}

// NewEnqueuer constructs an Enqueuer backed by the existing *redis.Client.
// Returns a cleanup function that closes the asynq client.
func NewEnqueuer(rdb *redis.Client, logger *zap.Logger) (Enqueuer, func(), error) {
	opt := rdb.Options()
	redisOpt := asynq.RedisClientOpt{
		Addr:     opt.Addr,
		Password: opt.Password,
		DB:       opt.DB,
	}

	client := asynq.NewClient(redisOpt)

	cleanup := func() {
		_ = client.Close()
		logger.Info("taskqueue: enqueuer closed")
	}

	return &AsynqEnqueuer{client: client, logger: logger}, cleanup, nil
}

// Enqueue submits a task to its target queue. It is a fast Redis LPUSH —
// not blocking beyond the Redis round-trip.
func (e *AsynqEnqueuer) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	info, err := e.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		e.logger.Error("taskqueue: enqueue failed",
			zap.String("task_type", task.Type()),
			zap.Error(err),
		)
		return nil, apperror.Internal("taskqueue: enqueue failed", err)
	}

	e.logger.Info("taskqueue: task enqueued",
		zap.String("task_type", task.Type()),
		zap.String("task_id", info.ID),
		zap.String("queue", info.Queue),
	)

	return info, nil
}
