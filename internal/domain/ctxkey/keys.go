package ctxkey

// ctxKey is a private type to prevent key collisions in context/Fiber locals.
type ctxKey string

const (
	// UserID stores uuid.UUID of the authenticated user.
	UserID ctxKey = "user_id"
	// Role stores the string role of the authenticated user.
	Role ctxKey = "role"
	// RequestID stores the string UUID assigned per request.
	RequestID ctxKey = "request_id"
)
