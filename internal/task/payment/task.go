package payment

import (
	"encoding/json"

	"go-standard/internal/apperror"

	"github.com/hibiken/asynq"
)

// Task type constants — the only place these strings appear in the codebase.
const (
	TypeProcessRefund = "payment:process_refund"
	TypeSendReceipt   = "payment:send_receipt"
)

// Queue is the asynq queue name for all payment domain tasks.
const Queue = "payment"

// ProcessRefundPayload is the typed payload for TypeProcessRefund.
type ProcessRefundPayload struct {
	PaymentID string  `json:"payment_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	RequestID string  `json:"request_id"`
}

// SendReceiptPayload is the typed payload for TypeSendReceipt.
type SendReceiptPayload struct {
	PaymentID string `json:"payment_id"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	RequestID string `json:"request_id"`
}

// NewProcessRefundTask constructs an asynq.Task ready for enqueuing.
// MaxRetry is 5 — refunds are high-value and worth more retries.
func NewProcessRefundTask(payload ProcessRefundPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.Internal("payment task: marshal payload failed", err)
	}
	return asynq.NewTask(
		TypeProcessRefund,
		data,
		asynq.Queue(Queue),
		asynq.MaxRetry(5),
	), nil
}

// NewSendReceiptTask constructs an asynq.Task ready for enqueuing.
func NewSendReceiptTask(payload SendReceiptPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.Internal("payment task: marshal payload failed", err)
	}
	return asynq.NewTask(
		TypeSendReceipt,
		data,
		asynq.Queue(Queue),
		asynq.MaxRetry(3),
	), nil
}
