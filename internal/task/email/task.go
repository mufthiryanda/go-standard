package email

import (
	"encoding/json"

	"go-standard/internal/apperror"

	"github.com/hibiken/asynq"
)

// Task type constants — the only place these strings appear in the codebase.
const (
	TypeSendWelcome   = "email:send_welcome"
	TypeResetPassword = "email:reset_password"
)

// Queue is the asynq queue name for all email domain tasks.
const Queue = "notification"

// SendWelcomePayload is the typed payload for TypeSendWelcome.
// RequestID is always included for cross-boundary tracing.
type SendWelcomePayload struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	RequestID string `json:"request_id"`
}

// ResetPasswordPayload is the typed payload for TypeResetPassword.
type ResetPasswordPayload struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	RequestID string `json:"request_id"`
}

// NewSendWelcomeTask constructs an asynq.Task ready for enqueuing.
// Queue and MaxRetry are set here — not at the call site.
func NewSendWelcomeTask(payload SendWelcomePayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.Internal("email task: marshal payload failed", err)
	}
	return asynq.NewTask(
		TypeSendWelcome,
		data,
		asynq.Queue(Queue),
		asynq.MaxRetry(3),
	), nil
}

// NewResetPasswordTask constructs an asynq.Task ready for enqueuing.
func NewResetPasswordTask(payload ResetPasswordPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.Internal("email task: marshal payload failed", err)
	}
	return asynq.NewTask(
		TypeResetPassword,
		data,
		asynq.Queue(Queue),
		asynq.MaxRetry(3),
	), nil
}
