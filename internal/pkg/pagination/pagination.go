package pagination

import (
	"go-standard/internal/dto/response"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
	minPage         = 1
	minPageSize     = 1
)

// Params holds validated, clamped pagination request values.
type Params struct {
	Page     int
	PageSize int
}

// NewParams creates a Params from raw request values, applying defaults and
// enforcing bounds per Standard #10:
//   - page defaults to 1, minimum 1
//   - page_size defaults to 20, minimum 1, maximum 100
func NewParams(page, pageSize int) Params {
	if page < minPage {
		page = defaultPage
	}

	if pageSize < minPageSize {
		pageSize = defaultPageSize
	} else if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return Params{Page: page, PageSize: pageSize}
}

// Offset returns the SQL OFFSET value for the current page.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// BuildMeta computes pagination metadata for the response envelope.
// TotalPages uses ceiling division to avoid truncating partial final pages.
func BuildMeta(params Params, totalItems int64) *response.Meta {
	totalPages := 0
	if params.PageSize > 0 && totalItems > 0 {
		totalPages = int((totalItems + int64(params.PageSize) - 1) / int64(params.PageSize))
	}

	return &response.Meta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
