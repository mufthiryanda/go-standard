package worker

import (
	"go-standard/internal/task/email"
	"go-standard/internal/task/notification"
	"go-standard/internal/task/payment"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// NewServeMux builds the central task router and applies global middleware.
// It is the single registration point for all task types.
func NewServeMux(
	logger *zap.Logger,
	emailHandler *email.EmailHandler,
	paymentHandler *payment.PaymentHandler,
	notificationHandler *notification.NotificationHandler,
) *asynq.ServeMux {
	mux := asynq.NewServeMux()

	// Global middleware — applied to every task.
	mux.Use(NewLoggerMiddleware(logger))

	registerEmailTasks(mux, emailHandler)
	registerPaymentTasks(mux, paymentHandler)
	registerNotificationTasks(mux, notificationHandler)

	return mux
}

func registerEmailTasks(mux *asynq.ServeMux, h *email.EmailHandler) {
	mux.HandleFunc(email.TypeSendWelcome, h.HandleSendWelcome)
	mux.HandleFunc(email.TypeResetPassword, h.HandleResetPassword)
}

func registerPaymentTasks(mux *asynq.ServeMux, h *payment.PaymentHandler) {
	mux.HandleFunc(payment.TypeProcessRefund, h.HandleProcessRefund)
	mux.HandleFunc(payment.TypeSendReceipt, h.HandleSendReceipt)
}

func registerNotificationTasks(mux *asynq.ServeMux, h *notification.NotificationHandler) {
	mux.HandleFunc(notification.TypeSendPush, h.HandleSendPush)
	mux.HandleFunc(notification.TypeSendSMS, h.HandleSendSMS)
}
