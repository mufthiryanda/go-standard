package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LogAsync fires a goroutine that writes a structured audit entry via zap.
// It is fire-and-forget; a panic inside the goroutine is recovered and logged.
func LogAsync(
	logger *zap.Logger,
	userID uuid.UUID,
	action, entityType string,
	entityID uuid.UUID,
	payload interface{},
) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("audit: panic in async log goroutine", zap.Any("recover", r))
			}
		}()

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			logger.Error("audit: failed to marshal payload",
				zap.String("entity_type", entityType),
				zap.String("entity_id", entityID.String()),
				zap.Error(err),
			)
			return
		}

		logger.Info("audit",
			zap.String("user_id", userID.String()),
			zap.String("action", action),
			zap.String("entity_type", entityType),
			zap.String("entity_id", entityID.String()),
			zap.Time("timestamp", time.Now()),
			zap.ByteString("payload", payloadBytes),
		)
	}()
}
