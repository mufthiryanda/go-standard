package filter

import "time"

// BaseFilter is embedded into every entity-specific filter struct.
// Pointer fields distinguish "not set" from "zero value".
type BaseFilter struct {
	CreatedAtFrom  *time.Time `query:"created_at_from"`
	CreatedAtTo    *time.Time `query:"created_at_to"`
	UpdatedAtFrom  *time.Time `query:"updated_at_from"`
	UpdatedAtTo    *time.Time `query:"updated_at_to"`
	IncludeDeleted bool       `query:"include_deleted"`
	SortBy         *string    `query:"sort_by"`
	SortOrder      *string    `query:"sort_order"`
	Page           int        `query:"page"`
	PageSize       int        `query:"page_size"`
}
