package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// NewLoggerMiddleware logs the lifecycle of every task: started → completed/failed.
// This is the only place task-level start/end is logged.
// Payload is never logged — it may contain PII.
func NewLoggerMiddleware(logger *zap.Logger) asynq.MiddlewareFunc {
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			start := time.Now()
			taskID, _ := asynq.GetTaskID(ctx)
			retryCount, _ := asynq.GetRetryCount(ctx)

			logger.Info("worker: task started",
				zap.String("task_type", t.Type()),
				zap.String("task_id", taskID),
				zap.Int("retry_count", retryCount),
			)

			err := next.ProcessTask(ctx, t)

			latency := time.Since(start)
			if err != nil {
				logger.Error("worker: task failed",
					zap.String("task_type", t.Type()),
					zap.String("task_id", taskID),
					zap.Int("retry_count", retryCount),
					zap.Int64("latency_ms", latency.Milliseconds()),
					zap.Error(err),
				)
				return err
			}

			logger.Info("worker: task completed",
				zap.String("task_type", t.Type()),
				zap.String("task_id", taskID),
				zap.Int64("latency_ms", latency.Milliseconds()),
			)

			return nil
		})
	}
}
