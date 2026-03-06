//go:build wireinject
// +build wireinject

package di

import (
	"go-standard/internal/config"
	"go-standard/internal/handler"
	"go-standard/internal/infrastructure"
	"go-standard/internal/pkg/jwt"
	"go-standard/internal/worker"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	govalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App holds all wired dependencies needed to start the server.
type App struct {
	Config         *config.Config
	DB             *gorm.DB
	Redis          *redis.Client
	ES             *elasticsearch.Client
	Logger         *zap.Logger
	JWTManager     *jwt.Manager
	ErrorHandler   fiber.ErrorHandler
	RequestID      RequestIDMiddleware
	LoggerMW       LoggerMiddleware
	Recover        RecoverMiddleware
	CORS           CORSMiddleware
	DefaultLimiter DefaultRateLimiter
	AuthLimiter    AuthRateLimiter
	AuthMW         AuthMiddleware
	Validator      *govalidator.Validate
	UserHandler    *handler.UserHandler
	AuthHandler    *handler.AuthHandler
}

// InfraSet wires all infrastructure dependencies.
var InfraSet = wire.NewSet(
	infrastructure.NewLogger,
	infrastructure.NewPostgresDB,
	infrastructure.NewRedisClient,
	infrastructure.NewElasticClient,
)

// MiddlewareSet wires the JWT manager and all middleware handlers.
var MiddlewareSet = wire.NewSet(
	jwt.NewManager,
	ProvideErrorHandler,
	ProvideRequestIDMiddleware,
	ProvideLoggerMiddleware,
	ProvideRecoverMiddleware,
	ProvideCORSMiddleware,
	ProvideDefaultRateLimiter,
	ProvideAuthRateLimiter,
	ProvideAuthMiddleware,
)

// InitializeApp is the Wire injector. Wire generates the body in wire_gen.go.
func InitializeApp(cfg *config.Config) (*App, func(), error) {
	wire.Build(
		InfraSet,
		MiddlewareSet,
		ValidatorSet,
		RepoSet,
		UsecaseSet,
		HandlerSet,
		//EnqueuerSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}

// InitializeWorker builds the worker binary's dependency graph.
// Shares InfraSet with the API binary — no duplication of infra wiring.
func InitializeWorker(cfg *config.Config) (*worker.Server, func(), error) {
	wire.Build(
		InfraSet,
		TaskHandlerSet,
		WorkerSet,
	)
	return nil, nil, nil
}
