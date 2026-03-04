package repository

import (
	"context"

	"go-standard/internal/domain/model"
	"go-standard/internal/dto/request"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines the data-access contract for the User entity.
// All methods accept an optional tx — if nil the implementation falls back to its own db.
type UserRepository interface {
	Create(ctx context.Context, tx *gorm.DB, user *model.User) error
	Update(ctx context.Context, tx *gorm.DB, user *model.User) error
	SoftDelete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
	HardDelete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
	FindByID(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.User, error)
	FindAll(ctx context.Context, tx *gorm.DB, f request.UserFilter) ([]model.User, int64, error)
	Count(ctx context.Context, tx *gorm.DB, f request.UserFilter) (int64, error)
	Exists(ctx context.Context, tx *gorm.DB, field string, value interface{}) (bool, error)
	FindByEmail(ctx context.Context, tx *gorm.DB, email string) (*model.User, error)
	FindByIDs(ctx context.Context, tx *gorm.DB, ids []uuid.UUID) ([]model.User, error)
}
