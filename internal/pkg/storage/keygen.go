package storage

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// GenerateKey produces a unique, collision-resistant object key.
// Format: {entity}/{uuid}.{ext}
// Example: "users/avatars/550e8400-e29b-41d4-a716-446655440000.jpg"
func GenerateKey(entity, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	id := uuid.New().String()
	entity = strings.Trim(entity, "/")
	if ext == "" {
		return fmt.Sprintf("%s/%s", entity, id)
	}
	return fmt.Sprintf("%s/%s%s", entity, id, ext)
}
