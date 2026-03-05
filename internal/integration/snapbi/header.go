package snapbi

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// mustLoadJakarta loads the Asia/Jakarta timezone. Panics on failure (boot-time invariant).
func mustLoadJakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic("snapbi: failed to load Asia/Jakarta timezone: " + err.Error())
	}
	return loc
}

// BuildTimestamp returns the current time formatted per SNAP BI spec (RFC3339 without sub-seconds).
func BuildTimestamp(loc *time.Location) string {
	return time.Now().In(loc).Format("2006-01-02T15:04:05-07:00")
}

// BuildExternalID returns a UUID v4 with dashes stripped, used as X-EXTERNAL-ID.
func BuildExternalID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
