package usecase

import (
	"context"
	"fmt"

	"go-standard/internal/apperror"
	"go-standard/internal/dto/request"
	"go-standard/internal/dto/response"
	"go-standard/internal/integration/snapbi"

	"go.uber.org/zap"
)

// PaymentUsecase defines payment operations.
type PaymentUsecase interface {
	ProcessVAPayment(ctx context.Context, req request.ProcessVAPaymentRequest) (*response.ProcessVAPaymentResponse, error)
	GetVAInquiry(ctx context.Context, virtualAccountNo string) (*response.VAInquiryResponse, error)
}

type paymentUsecase struct {
	snapBI snapbi.SnapBIClient
	logger *zap.Logger
}

// NewPaymentUsecase creates a PaymentUsecase with the given dependencies.
func NewPaymentUsecase(snapBI snapbi.SnapBIClient, logger *zap.Logger) PaymentUsecase {
	return &paymentUsecase{
		snapBI: snapBI,
		logger: logger,
	}
}

// ProcessVAPayment validates business rules and delegates the VA payment to SNAP BI.
func (u *paymentUsecase) ProcessVAPayment(
	ctx context.Context,
	req request.ProcessVAPaymentRequest,
) (*response.ProcessVAPaymentResponse, error) {
	if err := u.validateVAPayment(req); err != nil {
		return nil, err
	}

	snapReq := snapbi.TransferVARequest{
		PartnerServiceID:   req.PartnerServiceID,
		CustomerNo:         req.CustomerNo,
		VirtualAccountNo:   req.VirtualAccountNo,
		VirtualAccountName: req.VirtualAccountName,
		TrxID:              req.TrxID,
		TotalAmount: snapbi.AmountDetail{
			Value:    fmt.Sprintf("%.2f", req.Amount),
			Currency: req.Currency,
		},
	}

	result, err := u.snapBI.TransferVA(ctx, snapReq)
	if err != nil {
		u.logger.Error("payment_usecase: transfer va failed",
			zap.String("virtual_account_no", req.VirtualAccountNo),
			zap.String("trx_id", req.TrxID),
			zap.Error(err),
		)
		return nil, err
	}

	if result.VirtualAccountData == nil {
		return nil, apperror.Internal("payment_usecase: empty va data in transfer response", nil)
	}

	return &response.ProcessVAPaymentResponse{
		PartnerServiceID:   result.VirtualAccountData.PartnerServiceID,
		CustomerNo:         result.VirtualAccountData.CustomerNo,
		VirtualAccountNo:   result.VirtualAccountData.VirtualAccountNo,
		VirtualAccountName: result.VirtualAccountData.VirtualAccountName,
		Amount:             result.VirtualAccountData.TotalAmount.Value,
		Currency:           result.VirtualAccountData.TotalAmount.Currency,
	}, nil
}

// GetVAInquiry retrieves virtual account details from SNAP BI.
func (u *paymentUsecase) GetVAInquiry(
	ctx context.Context,
	virtualAccountNo string,
) (*response.VAInquiryResponse, error) {
	if virtualAccountNo == "" {
		return nil, apperror.BadRequest("payment_usecase: virtual_account_no is required")
	}

	snapReq := snapbi.InquiryVARequest{
		VirtualAccountNo: virtualAccountNo,
		InquiryRequestID: virtualAccountNo, // unique per inquiry; using VA no as default
	}

	result, err := u.snapBI.InquiryVA(ctx, snapReq)
	if err != nil {
		u.logger.Error("payment_usecase: inquiry va failed",
			zap.String("virtual_account_no", virtualAccountNo),
			zap.Error(err),
		)
		return nil, err
	}

	if result.VirtualAccountData == nil {
		return nil, apperror.NotFound("virtual account", virtualAccountNo)
	}

	return &response.VAInquiryResponse{
		PartnerServiceID:   result.VirtualAccountData.PartnerServiceID,
		CustomerNo:         result.VirtualAccountData.CustomerNo,
		VirtualAccountNo:   result.VirtualAccountData.VirtualAccountNo,
		VirtualAccountName: result.VirtualAccountData.VirtualAccountName,
		Amount:             result.VirtualAccountData.TotalAmount.Value,
		Currency:           result.VirtualAccountData.TotalAmount.Currency,
	}, nil
}

func (u *paymentUsecase) validateVAPayment(req request.ProcessVAPaymentRequest) error {
	if req.VirtualAccountNo == "" {
		return apperror.BadRequest("payment_usecase: virtual_account_no is required")
	}
	if req.Amount <= 0 {
		return apperror.Unprocessable("payment_usecase: amount must be greater than zero")
	}
	if req.TrxID == "" {
		return apperror.BadRequest("payment_usecase: trx_id is required")
	}
	return nil
}
