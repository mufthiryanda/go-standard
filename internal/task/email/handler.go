package email

import (
	"context"
	"encoding/json"
	"fmt"

	"go-standard/internal/domain/ctxkey"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// EmailHandler handles all email domain tasks.
type EmailHandler struct {
	logger *zap.Logger
}

// NewEmailHandler constructs an EmailHandler with injected dependencies.
func NewEmailHandler(logger *zap.Logger) *EmailHandler {
	return &EmailHandler{logger: logger}
}

// HandleSendWelcome processes TypeSendWelcome tasks.
func (h *EmailHandler) HandleSendWelcome(ctx context.Context, t *asynq.Task) error {
	var payload SendWelcomePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		// Malformed payload will never succeed — skip all retries.
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	// Re-inject request_id for downstream tracing.
	ctx = context.WithValue(ctx, ctxkey.RequestID, payload.RequestID) //nolint:staticcheck
	_ = ctx

	// Real implementation would call a mailer here.
	h.logger.Info("email: welcome sent",
		zap.String("user_id", payload.UserID),
		zap.String("email", payload.Email),
		zap.String("request_id", payload.RequestID),
	)

	return nil
}

// HandleResetPassword processes TypeResetPassword tasks.
func (h *EmailHandler) HandleResetPassword(ctx context.Context, t *asynq.Task) error {
	var payload ResetPasswordPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	ctx = context.WithValue(ctx, ctxkey.RequestID, payload.RequestID) //nolint:staticcheck
	_ = ctx

	// Real implementation would call a mailer with the reset token here.
	h.logger.Info("email: reset password sent",
		zap.String("user_id", payload.UserID),
		zap.String("email", payload.Email),
		zap.String("request_id", payload.RequestID),
	)

	return nil
}
