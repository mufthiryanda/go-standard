package storage

import (
	"fmt"
	"path/filepath"
	"strings"

	"go-standard/internal/apperror"
	"go-standard/internal/config"
)

// ValidateFile checks size, MIME type, and extension against config rules.
// Called inside Upload/UploadMultipart before any SDK call.
func ValidateFile(filename, contentType string, size int64, cfg config.StorageConfig) error {
	if cfg.MaxFileSizeBytes > 0 && size > cfg.MaxFileSizeBytes {
		return apperror.BadRequest(fmt.Sprintf(
			"file size %d bytes exceeds maximum %d bytes", size, cfg.MaxFileSizeBytes,
		))
	}

	if len(cfg.AllowedMIMETypes) > 0 && !contains(cfg.AllowedMIMETypes, contentType) {
		return apperror.BadRequest(fmt.Sprintf(
			"content type %q is not allowed", contentType,
		))
	}

	if len(cfg.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(filename))
		if ext == "" || !contains(cfg.AllowedExtensions, ext) {
			return apperror.BadRequest(fmt.Sprintf(
				"file extension %q is not allowed", ext,
			))
		}
	}

	return nil
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
