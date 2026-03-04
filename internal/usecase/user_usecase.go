package usecase

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/audit"
	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	"go-standard/internal/esindex"
	"go-standard/internal/pkg/cache"
	"go-standard/internal/pkg/dbutil"
	"go-standard/internal/pkg/esutil"
	"go-standard/internal/pkg/hash"
	"go-standard/internal/pkg/jwt"
	"go-standard/internal/pkg/pagination"
	"go-standard/internal/pkg/rediskey"
	"go-standard/internal/repository"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	userCacheTTL = 6 * time.Hour
	esUserIndex  = "project_users"
)

// UserUsecase defines the business-logic contract for User operations.
type UserUsecase interface {
	Register(ctx context.Context, req request.CreateUserRequest) (*response.UserResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*response.UserResponse, error)
	GetAll(ctx context.Context, f request.UserFilter) ([]response.UserResponse, *response.Meta, error)
	Update(ctx context.Context, id uuid.UUID, req request.UpdateUserRequest, actorID uuid.UUID) (*response.UserResponse, error)
	Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
}

type userUsecase struct {
	db       *gorm.DB
	userRepo repository.UserRepository
	rdb      *redis.Client
	es       *elasticsearch.Client
	jwtMgr   *jwt.Manager
	logger   *zap.Logger
}

// NewUserUsecase constructs a UserUsecase with all required dependencies.
func NewUserUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	rdb *redis.Client,
	es *elasticsearch.Client,
	jwtMgr *jwt.Manager,
	logger *zap.Logger,
) UserUsecase {
	return &userUsecase{
		db:       db,
		userRepo: userRepo,
		rdb:      rdb,
		es:       es,
		jwtMgr:   jwtMgr,
		logger:   logger,
	}
}

// Register hashes the password, ensures email uniqueness, persists to DB and ES.
func (u *userUsecase) Register(ctx context.Context, req request.CreateUserRequest) (*response.UserResponse, error) {
	exists, err := u.userRepo.Exists(ctx, nil, "email", req.Email)
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

	tx := u.db.WithContext(ctx).Begin()
	defer tx.Rollback() //nolint:errcheck

	if err := u.userRepo.Create(ctx, tx, user); err != nil {
		u.logger.Error("user_usecase: register failed", zap.String("email", req.Email), zap.Error(err))
		return nil, err
	}

	if err := u.indexToES(ctx, user); err != nil {
		u.logger.Error("user_usecase: es index failed on register", zap.String("user_id", user.ID.String()), zap.Error(err))
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("commit failed", err)
	}

	audit.LogAsync(u.logger, user.ID, "CREATE", "user", user.ID, user)

	res := response.NewUserResponse(user)
	return &res, nil
}

// GetByID retrieves a user by ID, served from Redis cache when available.
func (u *userUsecase) GetByID(ctx context.Context, id uuid.UUID) (*response.UserResponse, error) {
	key := rediskey.UserDetail(id)

	res, err := cache.GetOrLoad(ctx, u.rdb, key, userCacheTTL, func() (response.UserResponse, error) {
		user, repoErr := u.userRepo.FindByID(ctx, nil, id)
		if repoErr != nil {
			return response.UserResponse{}, repoErr
		}
		return response.NewUserResponse(user), nil
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetAll returns a paginated list. When a keyword is present, ES drives the lookup.
func (u *userUsecase) GetAll(ctx context.Context, f request.UserFilter) ([]response.UserResponse, *response.Meta, error) {
	if f.Keyword != nil && *f.Keyword != "" {
		return u.searchViaES(ctx, f)
	}

	users, total, err := u.userRepo.FindAll(ctx, nil, f)
	if err != nil {
		return nil, nil, err
	}

	params := pagination.NewParams(f.Page, f.PageSize)
	meta := pagination.BuildMeta(params, total)
	return response.NewUserListResponse(users), meta, nil
}

// Update applies partial changes inside a TX, reindexes ES, invalidates cache.
func (u *userUsecase) Update(ctx context.Context, id uuid.UUID, req request.UpdateUserRequest, actorID uuid.UUID) (*response.UserResponse, error) {
	tx := u.db.WithContext(ctx).Begin()
	defer tx.Rollback() //nolint:errcheck

	lockedTx := dbutil.ApplyLockForUpdate(tx)
	user, err := u.userRepo.FindByID(ctx, lockedTx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.Role != nil {
		user.Role = *req.Role
	}

	if err := u.userRepo.Update(ctx, tx, user); err != nil {
		u.logger.Error("user_usecase: update failed", zap.String("user_id", id.String()), zap.Error(err))
		return nil, err
	}

	if err := u.indexToES(ctx, user); err != nil {
		u.logger.Error("user_usecase: es reindex failed on update", zap.String("user_id", id.String()), zap.Error(err))
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperror.Internal("commit failed", err)
	}

	u.rdb.Del(ctx, rediskey.UserDetail(id)) //nolint:errcheck
	audit.LogAsync(u.logger, actorID, "UPDATE", "user", id, user)

	res := response.NewUserResponse(user)
	return &res, nil
}

// Delete soft-deletes the user, removes the ES document, and invalidates cache.
func (u *userUsecase) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	tx := u.db.WithContext(ctx).Begin()
	defer tx.Rollback() //nolint:errcheck

	if err := u.userRepo.SoftDelete(ctx, tx, id); err != nil {
		u.logger.Error("user_usecase: soft delete failed", zap.String("user_id", id.String()), zap.Error(err))
		return err
	}

	if err := u.deleteFromES(ctx, id); err != nil {
		u.logger.Error("user_usecase: es delete failed", zap.String("user_id", id.String()), zap.Error(err))
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return apperror.Internal("commit failed", err)
	}

	u.rdb.Del(ctx, rediskey.UserDetail(id)) //nolint:errcheck
	audit.LogAsync(u.logger, actorID, "DELETE", "user", id, map[string]string{"id": id.String()})

	return nil
}

// searchViaES queries Elasticsearch then fetches authoritative data from Postgres.
func (u *userUsecase) searchViaES(ctx context.Context, f request.UserFilter) ([]response.UserResponse, *response.Meta, error) {
	params := pagination.NewParams(f.Page, f.PageSize)

	query := esutil.BoolQuery(
		[]map[string]interface{}{
			esutil.MatchQuery("name", *f.Keyword),
		},
		nil, nil, nil,
	)

	body, err := esutil.SearchRequest(esUserIndex, query, params.Offset(), params.PageSize)
	if err != nil {
		return nil, nil, apperror.Internal("es search request build failed", err)
	}

	res, err := u.es.Search(
		u.es.Search.WithContext(ctx),
		u.es.Search.WithIndex(esUserIndex),
		u.es.Search.WithBody(body),
	)
	if err != nil {
		return nil, nil, apperror.Internal("es search failed", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, nil, apperror.Internal("es search returned non-200", nil)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return nil, nil, apperror.Internal("es read body failed", err)
	}

	sr, err := esutil.ParseSearchResult(buf.Bytes())
	if err != nil {
		return nil, nil, apperror.Internal("es parse result failed", err)
	}

	ids := esutil.ExtractIDs(sr)
	if len(ids) == 0 {
		meta := pagination.BuildMeta(params, 0)
		return []response.UserResponse{}, meta, nil
	}

	uuids := make([]uuid.UUID, 0, len(ids))
	for _, raw := range ids {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr == nil {
			uuids = append(uuids, parsed)
		}
	}

	users, err := u.userRepo.FindByIDs(ctx, nil, uuids)
	if err != nil {
		return nil, nil, err
	}

	meta := pagination.BuildMeta(params, sr.Hits.Total.Value)
	return response.NewUserListResponse(users), meta, nil
}

// indexToES indexes or updates a user document in Elasticsearch.
func (u *userUsecase) indexToES(ctx context.Context, user *model.User) error {
	doc := esindex.UserDocumentFromModel(user)
	reader, err := doc.ToReader()
	if err != nil {
		return apperror.Internal("es doc marshal failed", err)
	}

	res, err := u.es.Index(
		esUserIndex,
		reader,
		u.es.Index.WithContext(ctx),
		u.es.Index.WithDocumentID(user.ID.String()),
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

// deleteFromES removes a user document from Elasticsearch.
func (u *userUsecase) deleteFromES(ctx context.Context, id uuid.UUID) error {
	res, err := u.es.Delete(
		esUserIndex,
		id.String(),
		u.es.Delete.WithContext(ctx),
	)
	if err != nil {
		return apperror.Internal("es delete request failed", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != http.StatusNotFound {
		return apperror.Internal("es delete returned error: "+res.Status(), nil)
	}
	return nil
}
