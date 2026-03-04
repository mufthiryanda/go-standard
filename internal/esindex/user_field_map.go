package esindex

// UserFieldMapES maps API/filter field names to their Elasticsearch field paths.
// Used by the ES query builder for dynamic sort and filter resolution.
var UserFieldMapES = map[string]string{
	"id":         "id",
	"email":      "email.keyword",
	"name":       "name.keyword",
	"role":       "role",
	"phone":      "phone",
	"created_at": "created_at",
	"updated_at": "updated_at",
	"deleted_at": "deleted_at",
}
