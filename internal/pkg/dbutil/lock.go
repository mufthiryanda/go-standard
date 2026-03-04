package dbutil

import "gorm.io/gorm"

// ApplyLockForUpdate appends a SELECT FOR UPDATE clause to the query.
// Called by usecases before passing tx to repos that need pessimistic locking.
func ApplyLockForUpdate(tx *gorm.DB) *gorm.DB {
	return tx.Set("gorm:query_option", "FOR UPDATE")
}
