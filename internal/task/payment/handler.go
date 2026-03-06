package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"go-standard/internal/domain/ctxkey"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// PaymentHandler handles all payment domain tasks.
type PaymentHandler struct {
	logger *zap.Logger
}

// NewPaymentHandler constructs a PaymentHandler with injected dependencies.
func NewPaymentHandler(logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{logger: logger}
}

// HandleProcessRefund processes TypeProcessRefund tasks.
func (h *PaymentHandler) HandleProcessRefund(ctx context.Context, t *asynq.Task) error {
	var payload ProcessRefundPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	ctx = context.WithValue(ctx, ctxkey.RequestID, payload.RequestID) //nolint:staticcheck
	_ = ctx

	// Real implementation would call the payment gateway refund API here.
	h.logger.Info("payment: refund processed",
		zap.String("payment_id", payload.PaymentID),
		zap.Float64("amount", payload.Amount),
		zap.String("currency", payload.Currency),
		zap.String("request_id", payload.RequestID),
	)

	return nil
}

// HandleSendReceipt processes TypeSendReceipt tasks.
func (h *PaymentHandler) HandleSendReceipt(ctx context.Context, t *asynq.Task) error {
	var payload SendReceiptPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	ctx = context.WithValue(ctx, ctxkey.RequestID, payload.RequestID) //nolint:staticcheck
	_ = ctx

	// Real implementation would call a mailer with the receipt here.
	h.logger.Info("payment: receipt sent",
		zap.String("payment_id", payload.PaymentID),
		zap.String("email", payload.Email),
		zap.String("request_id", payload.RequestID),
	)

	return nil
}
