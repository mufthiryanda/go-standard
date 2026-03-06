package notification

import (
	"encoding/json"

	"go-standard/internal/apperror"

	"github.com/hibiken/asynq"
)

// Task type constants — the only place these strings appear in the codebase.
const (
	TypeSendPush = "notification:send_push"
	TypeSendSMS  = "notification:send_sms"
)

// Queue is the asynq queue name for all notification domain tasks.
const Queue = "notification"

// SendPushPayload is the typed payload for TypeSendPush.
type SendPushPayload struct {
	UserID    string            `json:"user_id"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data,omitempty"`
	RequestID string            `json:"request_id"`
}

// SendSMSPayload is the typed payload for TypeSendSMS.
type SendSMSPayload struct {
	Phone     string `json:"phone"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewSendPushTask constructs an asynq.Task ready for enqueuing.
func NewSendPushTask(payload SendPushPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.Internal("notification task: marshal payload failed", err)
	}
	return asynq.NewTask(
		TypeSendPush,
		data,
		asynq.Queue(Queue),
		asynq.MaxRetry(3),
	), nil
}

// NewSendSMSTask constructs an asynq.Task ready for enqueuing.
func NewSendSMSTask(payload SendSMSPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.Internal("notification task: marshal payload failed", err)
	}
	return asynq.NewTask(
		TypeSendSMS,
		data,
		asynq.Queue(Queue),
		asynq.MaxRetry(3),
	), nil
}
