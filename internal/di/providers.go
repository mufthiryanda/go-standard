// internal/di/providers.go
package di

import (
	"go-standard/internal/config"
	"go-standard/internal/handler"
	"go-standard/internal/integration/snapbi"
	"go-standard/internal/middleware"
	"go-standard/internal/pkg/httpclient"
	"go-standard/internal/pkg/jwt"
	"go-standard/internal/pkg/storage"
	"go-standard/internal/repository"
	"go-standard/internal/usecase"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	govalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Named middleware types allow Wire to distinguish multiple fiber.Handler
// and fiber.ErrorHandler values during dependency injection.

// RecoverMiddleware wraps the panic-recovery fiber.Handler.
type RecoverMiddleware fiber.Handler

// RequestIDMiddleware wraps the request-ID fiber.Handler.
type RequestIDMiddleware fiber.Handler

// LoggerMiddleware wraps the structured-logger fiber.Handler.
type LoggerMiddleware fiber.Handler

// CORSMiddleware wraps the CORS fiber.Handler.
type CORSMiddleware fiber.Handler

// DefaultRateLimiter wraps the default-policy rate-limit fiber.Handler.
type DefaultRateLimiter fiber.Handler

// AuthRateLimiter wraps the auth-policy rate-limit fiber.Handler.
type AuthRateLimiter fiber.Handler

// AuthMiddleware wraps the JWT-auth fiber.Handler.
type AuthMiddleware fiber.Handler

// ── Middleware Providers ─────────────────────────────────────────────────────

// ProvideErrorHandler constructs the fiber.ErrorHandler used in fiber.Config.
func ProvideErrorHandler() fiber.ErrorHandler {
	return middleware.NewErrorHandler()
}

// ProvideRequestIDMiddleware constructs and casts RequestIDMiddleware.
func ProvideRequestIDMiddleware() RequestIDMiddleware {
	return RequestIDMiddleware(middleware.NewRequestID())
}

// ProvideLoggerMiddleware constructs and casts LoggerMiddleware.
func ProvideLoggerMiddleware(logger *zap.Logger) LoggerMiddleware {
	return LoggerMiddleware(middleware.NewLogger(logger))
}

// ProvideRecoverMiddleware constructs and casts RecoverMiddleware.
func ProvideRecoverMiddleware(logger *zap.Logger) RecoverMiddleware {
	return RecoverMiddleware(middleware.NewRecover(logger))
}

// ProvideCORSMiddleware constructs and casts CORSMiddleware.
func ProvideCORSMiddleware() CORSMiddleware {
	return CORSMiddleware(middleware.NewCORS())
}

// ProvideDefaultRateLimiter constructs the default-policy limiter.
func ProvideDefaultRateLimiter(rdb *redis.Client, cfg *config.Config) DefaultRateLimiter {
	return DefaultRateLimiter(middleware.NewRateLimiter(rdb, cfg.RateLimit.Default))
}

// ProvideAuthRateLimiter constructs the auth-policy limiter.
func ProvideAuthRateLimiter(rdb *redis.Client, cfg *config.Config) AuthRateLimiter {
	return AuthRateLimiter(middleware.NewRateLimiter(rdb, cfg.RateLimit.Auth))
}

// ProvideAuthMiddleware constructs and casts AuthMiddleware.
func ProvideAuthMiddleware(jwtMgr *jwt.Manager) AuthMiddleware {
	return AuthMiddleware(middleware.NewAuth(jwtMgr))
}

// ProvideBaseHTTPClient provides a production-ready HTTP client with retry and circuit breaker.
func ProvideBaseHTTPClient(logger *zap.Logger) httpclient.Client {
	return httpclient.NewBaseClient(logger,
		httpclient.WithTimeout(30*time.Second),
		httpclient.WithRetry(httpclient.RetryConfig{
			MaxAttempts:   3,
			InitialWait:   200 * time.Millisecond,
			MaxWait:       2 * time.Second,
			Multiplier:    2.0,
			RetryOnStatus: []int{429, 502, 503, 504},
		}),
		httpclient.WithCircuitBreaker(httpclient.CircuitBreakerConfig{
			Name:            "default",
			MaxFailures:     5,
			ResetTimeout:    30 * time.Second,
			HalfOpenMaxReqs: 1,
		}),
	)
}

// ProvideSnapBIClient provides the SNAP BI integration client via Wire.
func ProvideSnapBIClient(
	base httpclient.Client,
	cfg *config.Config,
	rdb *redis.Client,
	logger *zap.Logger,
) (snapbi.SnapBIClient, func(), error) {
	return snapbi.NewSnapBIClient(base, cfg.Integrations.SnapBI, rdb, logger)
}

// ProvideStorageManager initialises all storage providers and returns the manager.
func ProvideStorageManager(cfg *config.Config, logger *zap.Logger) (storage.StorageManager, func(), error) {
	return storage.NewStorageManager(cfg.Storage, logger)
}

// ── Validator Provider ───────────────────────────────────────────────────────

// ProvideValidator constructs a shared go-playground/validator instance with
// custom domain rules registered.
func ProvideValidator() *govalidator.Validate {
	return govalidator.New()
}

// ── Repository Providers ─────────────────────────────────────────────────────

// ProvideUserRepository constructs a UserRepository backed by GORM.
func ProvideUserRepository(db *gorm.DB, logger *zap.Logger) repository.UserRepository {
	return repository.NewUserRepository(db, logger)
}

// ── Usecase Providers ────────────────────────────────────────────────────────

// ProvideUserUsecase constructs a UserUsecase with all required dependencies.
func ProvideUserUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	rdb *redis.Client,
	es *elasticsearch.Client,
	jwtMgr *jwt.Manager,
	logger *zap.Logger,
) usecase.UserUsecase {
	return usecase.NewUserUsecase(db, userRepo, rdb, es, jwtMgr, logger)
}

// ProvideAuthUsecase constructs an AuthUsecase with all required dependencies.
func ProvideAuthUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	rdb *redis.Client,
	es *elasticsearch.Client,
	jwtMgr *jwt.Manager,
	logger *zap.Logger,
) usecase.AuthUsecase {
	return usecase.NewAuthUsecase(db, userRepo, rdb, es, jwtMgr, logger)
}

// ── Handler Providers ────────────────────────────────────────────────────────

// ProvideUserHandler constructs a UserHandler with the shared validator.
func ProvideUserHandler(uc usecase.UserUsecase, v *govalidator.Validate) *handler.UserHandler {
	return handler.NewUserHandler(uc, v)
}

// ProvideAuthHandler constructs an AuthHandler with the shared validator.
func ProvideAuthHandler(uc usecase.AuthUsecase, v *govalidator.Validate) *handler.AuthHandler {
	return handler.NewAuthHandler(uc, v)
}

var IntegrationSet = wire.NewSet(
	ProvideBaseHTTPClient,
	ProvideSnapBIClient,
)

// ── Provider Sets ────────────────────────────────────────────────────────────

// ValidatorSet provides the shared validator instance.
var ValidatorSet = wire.NewSet(ProvideValidator)

// RepoSet provides all repository implementations.
var RepoSet = wire.NewSet(ProvideUserRepository)

// UsecaseSet provides all usecase implementations.
var UsecaseSet = wire.NewSet(ProvideUserUsecase, ProvideAuthUsecase)

// HandlerSet provides all HTTP handler instances.
var HandlerSet = wire.NewSet(ProvideUserHandler, ProvideAuthHandler)

// StorageSet is the Wire provider set for object storage.
var StorageSet = wire.NewSet(ProvideStorageManager)
