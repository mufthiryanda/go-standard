package request

// ProcessVAPaymentRequest is the usecase-layer DTO for initiating a VA payment.
type ProcessVAPaymentRequest struct {
	PartnerServiceID   string  `json:"partnerServiceId"   validate:"required"`
	CustomerNo         string  `json:"customerNo"         validate:"required"`
	VirtualAccountNo   string  `json:"virtualAccountNo"   validate:"required"`
	VirtualAccountName string  `json:"virtualAccountName" validate:"required"`
	TrxID              string  `json:"trxId"              validate:"required"`
	Amount             float64 `json:"amount"             validate:"required,gt=0"`
	Currency           string  `json:"currency"           validate:"required,len=3"`
}
