package filter

import (
	"fmt"

	"go-standard/internal/pkg/pagination"

	"gorm.io/gorm"
)

// defaultSortColumn is used when SortBy is nil or not in the whitelist.
const defaultSortColumn = "created_at"

// ApplyBaseFilter applies common filter conditions to a GORM query.
// Entity repos call this first, then chain their own conditions.
// allowedSortColumns is a per-entity whitelist; pass nil to allow only the default.
func ApplyBaseFilter(db *gorm.DB, f BaseFilter, allowedSortColumns map[string]bool) *gorm.DB {
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

	sortCol := defaultSortColumn
	if f.SortBy != nil && allowedSortColumns[*f.SortBy] {
		sortCol = *f.SortBy
	}

	sortDir := "DESC"
	if f.SortOrder != nil && *f.SortOrder == "asc" {
		sortDir = "ASC"
	}

	db = db.Order(fmt.Sprintf("%s %s", sortCol, sortDir))

	params := pagination.NewParams(f.Page, f.PageSize)
	db = db.Offset(params.Offset()).Limit(params.PageSize)

	return db
}
