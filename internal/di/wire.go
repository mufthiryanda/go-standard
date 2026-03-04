//go:build wireinject
// +build wireinject

package di

import (
	"go-standard/internal/config"
	"go-standard/internal/infrastructure"
	"go-standard/internal/pkg/jwt"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App holds all wired dependencies needed to start the server.
// Phase 5+ will add RepoSet, UsecaseSet, HandlerSet fields.
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
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
