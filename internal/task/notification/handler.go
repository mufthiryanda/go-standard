package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"go-standard/internal/domain/ctxkey"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// NotificationHandler handles all notification domain tasks.
type NotificationHandler struct {
	logger *zap.Logger
}

// NewNotificationHandler constructs a NotificationHandler with injected dependencies.
func NewNotificationHandler(logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{logger: logger}
}

// HandleSendPush processes TypeSendPush tasks.
func (h *NotificationHandler) HandleSendPush(ctx context.Context, t *asynq.Task) error {
	var payload SendPushPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	ctx = context.WithValue(ctx, ctxkey.RequestID, payload.RequestID) //nolint:staticcheck
	_ = ctx

	// Real implementation would call FCM / APNs here.
	h.logger.Info("notification: push sent",
		zap.String("user_id", payload.UserID),
		zap.String("title", payload.Title),
		zap.String("request_id", payload.RequestID),
	)

	return nil
}

// HandleSendSMS processes TypeSendSMS tasks.
func (h *NotificationHandler) HandleSendSMS(ctx context.Context, t *asynq.Task) error {
	var payload SendSMSPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	ctx = context.WithValue(ctx, ctxkey.RequestID, payload.RequestID) //nolint:staticcheck
	_ = ctx

	// Real implementation would call an SMS gateway here.
	h.logger.Info("notification: sms sent",
		zap.String("phone", payload.Phone),
		zap.String("request_id", payload.RequestID),
	)

	return nil
}
