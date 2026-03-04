package repository

import (
	"context"
	"time"

	"go-standard/internal/apperror"
	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"
	"go-standard/internal/pkg/filter"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var userSortableColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"email":      true,
	"name":       true,
}

type userRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewUserRepository constructs a UserRepository backed by GORM.
func NewUserRepository(db *gorm.DB, logger *zap.Logger) UserRepository {
	return &userRepository{db: db, logger: logger}
}

// getDB resolves the transaction or falls back to the struct-level db.
func (r *userRepository) getDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *userRepository) Create(ctx context.Context, tx *gorm.DB, user *model.User) error {
	if err := r.getDB(tx).WithContext(ctx).Create(user).Error; err != nil {
		r.logger.Error("user_repository: create failed",
			zap.String("email", user.Email),
			zap.Error(err),
		)
		return apperror.Internal("failed to create user", err)
	}
	return nil
}

func (r *userRepository) Update(ctx context.Context, tx *gorm.DB, user *model.User) error {
	if err := r.getDB(tx).WithContext(ctx).Save(user).Error; err != nil {
		r.logger.Error("user_repository: update failed",
			zap.String("user_id", user.ID.String()),
			zap.Error(err),
		)
		return apperror.Internal("failed to update user", err)
	}
	return nil
}

func (r *userRepository) SoftDelete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	now := time.Now()
	result := r.getDB(tx).WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)

	if result.Error != nil {
		r.logger.Error("user_repository: soft delete failed",
			zap.String("user_id", id.String()),
			zap.Error(result.Error),
		)
		return apperror.Internal("failed to delete user", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("user", id.String())
	}
	return nil
}

func (r *userRepository) HardDelete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	result := r.getDB(tx).WithContext(ctx).
		Unscoped().
		Delete(&model.User{}, "id = ?", id)

	if result.Error != nil {
		r.logger.Error("user_repository: hard delete failed",
			zap.String("user_id", id.String()),
			zap.Error(result.Error),
		)
		return apperror.Internal("failed to hard-delete user", result.Error)
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.getDB(tx).WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("user", id.String())
		}
		r.logger.Error("user_repository: find by id failed",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return nil, apperror.Internal("failed to find user", err)
	}
	return &user, nil
}

func (r *userRepository) FindAll(ctx context.Context, tx *gorm.DB, f request.UserFilter) ([]model.User, int64, error) {
	// COUNT — conditions only, no ORDER BY / LIMIT.
	countDB := r.applyEntityConditions(r.getDB(tx).WithContext(ctx).Model(&model.User{}), f)
	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		r.logger.Error("user_repository: count failed", zap.Error(err))
		return nil, 0, apperror.Internal("failed to count users", err)
	}

	// DATA — ApplyBaseFilter handles soft-delete, date ranges, sort, pagination;
	// then we chain entity-specific extra conditions.
	dataDB := filter.ApplyBaseFilter(
		r.getDB(tx).WithContext(ctx).Model(&model.User{}),
		f.BaseFilter,
		userSortableColumns,
	)
	dataDB = r.applyEntityExtraConditions(dataDB, f)

	var users []model.User
	if err := dataDB.Find(&users).Error; err != nil {
		r.logger.Error("user_repository: find all failed", zap.Error(err))
		return nil, 0, apperror.Internal("failed to list users", err)
	}
	return users, total, nil
}

func (r *userRepository) Count(ctx context.Context, tx *gorm.DB, f request.UserFilter) (int64, error) {
	db := r.applyEntityConditions(r.getDB(tx).WithContext(ctx).Model(&model.User{}), f)
	var count int64
	if err := db.Count(&count).Error; err != nil {
		r.logger.Error("user_repository: count failed", zap.Error(err))
		return 0, apperror.Internal("failed to count users", err)
	}
	return count, nil
}

func (r *userRepository) Exists(ctx context.Context, tx *gorm.DB, field string, value interface{}) (bool, error) {
	var count int64
	err := r.getDB(tx).WithContext(ctx).
		Model(&model.User{}).
		Where("deleted_at IS NULL").
		Where(field+" = ?", value).
		Count(&count).Error

	if err != nil {
		r.logger.Error("user_repository: exists check failed",
			zap.String("field", field),
			zap.Error(err),
		)
		return false, apperror.Internal("failed to check existence", err)
	}
	return count > 0, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, tx *gorm.DB, email string) (*model.User, error) {
	var user model.User
	err := r.getDB(tx).WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("user", email)
		}
		r.logger.Error("user_repository: find by email failed",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, apperror.Internal("failed to find user by email", err)
	}
	return &user, nil
}

func (r *userRepository) FindByIDs(ctx context.Context, tx *gorm.DB, ids []uuid.UUID) ([]model.User, error) {
	var users []model.User
	if err := r.getDB(tx).WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&users).Error; err != nil {
		r.logger.Error("user_repository: find by ids failed", zap.Error(err))
		return nil, apperror.Internal("failed to find users by ids", err)
	}
	return users, nil
}

// applyEntityConditions applies all WHERE conditions including base soft-delete
// and date ranges — used for count queries (no ORDER BY / pagination).
func (r *userRepository) applyEntityConditions(db *gorm.DB, f request.UserFilter) *gorm.DB {
	if !f.IncludeDeleted {
		db = db.Where("deleted_at IS NULL")
	}
	if f.CreatedAtFrom != nil {
		db = db.Where("created_at >= ?", *f.CreatedAtFrom)
	}
	if f.CreatedAtTo != nil {
		db = db.Where("created_at <= ?", *f.CreatedAtTo)
	}
	if f.UpdatedAtFrom != nil {
		db = db.Where("updated_at >= ?", *f.UpdatedAtFrom)
	}
	if f.UpdatedAtTo != nil {
		db = db.Where("updated_at <= ?", *f.UpdatedAtTo)
	}
	return r.applyEntityExtraConditions(db, f)
}

// applyEntityExtraConditions applies only the user-specific WHERE clauses on top
// of a query that already had ApplyBaseFilter (which handles soft-delete + dates).
func (r *userRepository) applyEntityExtraConditions(db *gorm.DB, f request.UserFilter) *gorm.DB {
	if f.Email != nil {
		db = db.Where("email ILIKE ?", "%"+*f.Email+"%")
	}
	if f.Name != nil {
		db = db.Where("name ILIKE ?", "%"+*f.Name+"%")
	}
	if f.Role != nil {
		db = db.Where("role = ?", *f.Role)
	}
	if f.Phone != nil {
		db = db.Where("phone = ?", *f.Phone)
	}
	return db
}
