package usecase

import (
	"context"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/audit"
	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	"go-standard/internal/esindex"
	"go-standard/internal/pkg/hash"
	"go-standard/internal/pkg/jwt"
	"go-standard/internal/pkg/rediskey"
	"go-standard/internal/repository"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const authESIndex = "project_users"

// AuthUsecase defines the business-logic contract for authentication.
type AuthUsecase interface {
	Register(ctx context.Context, req request.RegisterRequest) (*response.AuthTokenResponse, error)
	Login(ctx context.Context, req request.LoginRequest) (*response.AuthTokenResponse, error)
	Refresh(ctx context.Context, req request.RefreshRequest) (*response.AuthTokenResponse, error)
}

type authUsecase struct {
	db       *gorm.DB
	userRepo repository.UserRepository
	rdb      *redis.Client
	es       *elasticsearch.Client
	jwtMgr   *jwt.Manager
	logger   *zap.Logger
}

// NewAuthUsecase constructs an AuthUsecase with all required dependencies.
func NewAuthUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	rdb *redis.Client,
	es *elasticsearch.Client,
	jwtMgr *jwt.Manager,
	logger *zap.Logger,
) AuthUsecase {
	return &authUsecase{
		db:       db,
		userRepo: userRepo,
		rdb:      rdb,
		es:       es,
		jwtMgr:   jwtMgr,
		logger:   logger,
	}
}

// Register checks email uniqueness, creates user in TX, indexes ES, generates token pair.
func (a *authUsecase) Register(ctx context.Context, req request.RegisterRequest) (*response.AuthTokenResponse, error) {
	exists, err := a.userRepo.Exists(ctx, nil, "email", req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.Conflict("email already registered")
	}

	hashed, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := req.ToModel(hashed)

	tx := a.db.WithContext(ctx).Begin()
	defer tx.Rollback() //nolint:errcheck

	if err := a.userRepo.Create(ctx, tx, user); err != nil {
		a.logger.Error("auth_usecase: register create failed", zap.String("email", req.Email), zap.Error(err))
		return nil, err
	}

	if err := a.indexToES(ctx, user); err != nil {
		a.logger.Error("auth_usecase: es index failed on register", zap.String("user_id", user.ID.String()), zap.Error(err))
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("commit failed", err)
	}

	tokenRes, err := a.generateAndStoreTokens(ctx, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	audit.LogAsync(a.logger, user.ID, "CREATE", "user", user.ID, user)

	return tokenRes, nil
}

// Login authenticates by email/password, then issues a token pair.
func (a *authUsecase) Login(ctx context.Context, req request.LoginRequest) (*response.AuthTokenResponse, error) {
	user, err := a.userRepo.FindByEmail(ctx, nil, req.Email)
	if err != nil {
		return nil, apperror.Unauthorized("invalid email or password")
	}

	if hash.CheckPassword(user.Password, req.Password) != nil {
		return nil, apperror.Unauthorized("invalid email or password")
	}

	return a.generateAndStoreTokens(ctx, user.ID, user.Role)
}

// Refresh validates the old refresh token, rotates tokens, returns a new pair.
func (a *authUsecase) Refresh(ctx context.Context, req request.RefreshRequest) (*response.AuthTokenResponse, error) {
	claims, err := a.jwtMgr.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, apperror.Unauthorized("invalid token subject")
	}

	key := rediskey.AuthRefresh(userID)
	stored, err := a.rdb.Get(ctx, key).Result()
	if err != nil {
		a.logger.Error("auth_usecase: redis get refresh failed", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, apperror.Unauthorized("refresh token not found or expired")
	}

	if stored != req.RefreshToken {
		return nil, apperror.Unauthorized("refresh token mismatch")
	}

	a.rdb.Del(ctx, key) //nolint:errcheck

	user, err := a.userRepo.FindByID(ctx, nil, userID)
	if err != nil {
		return nil, apperror.Unauthorized("user not found")
	}

	return a.generateAndStoreTokens(ctx, user.ID, user.Role)
}

// generateAndStoreTokens creates an access+refresh pair and stores refresh in Redis.
func (a *authUsecase) generateAndStoreTokens(ctx context.Context, userID uuid.UUID, role string) (*response.AuthTokenResponse, error) {
	accessToken, err := a.jwtMgr.GenerateAccessToken(userID, role)
	if err != nil {
		a.logger.Error("auth_usecase: generate access token failed", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, apperror.Internal("failed to generate access token", err)
	}

	refreshToken, err := a.jwtMgr.GenerateRefreshToken(userID)
	if err != nil {
		a.logger.Error("auth_usecase: generate refresh token failed", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, apperror.Internal("failed to generate refresh token", err)
	}

	accessClaims, err := a.jwtMgr.ValidateToken(accessToken)
	if err != nil {
		return nil, apperror.Internal("failed to parse access token claims", err)
	}
	refreshClaims, err := a.jwtMgr.ValidateToken(refreshToken)
	if err != nil {
		return nil, apperror.Internal("failed to parse refresh token claims", err)
	}

	// FIXED: Round to nearest minute to ensure predictable TTL for testing and stable cleanup
	refreshTTL := time.Until(refreshClaims.ExpiresAt.Time).Round(time.Minute)

	key := rediskey.AuthRefresh(userID)
	if err := a.rdb.Set(ctx, key, refreshToken, refreshTTL).Err(); err != nil {
		a.logger.Error("auth_usecase: redis set refresh failed", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, apperror.Internal("failed to store refresh token", err)
	}

	res := response.NewAuthTokenResponse(accessToken, refreshToken, accessClaims.ExpiresAt.Time)
	return &res, nil
}

// indexToES indexes a user document into Elasticsearch.
func (a *authUsecase) indexToES(ctx context.Context, user *model.User) error {
	doc := esindex.UserDocumentFromModel(user)
	reader, err := doc.ToReader()
	if err != nil {
		return apperror.Internal("es doc marshal failed", err)
	}

	res, err := a.es.Index(
		authESIndex,
		reader,
		a.es.Index.WithContext(ctx),
		a.es.Index.WithDocumentID(user.ID.String()),
	)
	if err != nil {
		return apperror.Internal("es index request failed", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return apperror.Internal("es index returned error: "+res.Status(), nil)
	}
	return nil
}
