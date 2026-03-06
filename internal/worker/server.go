package worker

import (
	"context"
	"time"

	"go-standard/internal/config"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Server wraps asynq.Server with lifecycle management.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	logger *zap.Logger
}

// NewServer constructs an asynq.Server from the shared *redis.Client and config.
// Returns a cleanup function that performs a graceful shutdown.
func NewServer(cfg *config.Config, rdb *redis.Client, mux *asynq.ServeMux, logger *zap.Logger) (*Server, func(), error) {
	opt := rdb.Options()
	redisOpt := asynq.RedisClientOpt{
		Addr:     opt.Addr,
		Password: opt.Password,
		DB:       opt.DB,
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: cfg.Worker.Concurrency,
		Queues: map[string]int{
			cfg.Worker.Queues.User:         1,
			cfg.Worker.Queues.Payment:      1,
			cfg.Worker.Queues.Notification: 1,
		},
		ErrorHandler:    asynq.ErrorHandlerFunc(newFinalErrorHandler(logger)),
		ShutdownTimeout: 30 * time.Second,
	})

	cleanup := func() {
		srv.Shutdown()
		logger.Info("worker: server shutdown complete")
	}

	return &Server{server: srv, mux: mux, logger: logger}, cleanup, nil
}

// Start blocks until the server is stopped. Call from a goroutine.
func (s *Server) Start() error {
	s.logger.Info("worker: server starting")
	return s.server.Run(s.mux)
}

// newFinalErrorHandler is called when a task exhausts all retries and is archived.
func newFinalErrorHandler(logger *zap.Logger) func(ctx context.Context, task *asynq.Task, err error) {
	return func(ctx context.Context, task *asynq.Task, err error) {
		taskID, _ := asynq.GetTaskID(ctx)
		logger.Error("worker: task archived after exhausting retries",
			zap.String("task_type", task.Type()),
			zap.String("task_id", taskID),
			zap.Error(err),
		)
	}
}
