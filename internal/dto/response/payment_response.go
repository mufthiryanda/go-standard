package response

// ProcessVAPaymentResponse is returned by PaymentUsecase.ProcessVAPayment.
type ProcessVAPaymentResponse struct {
	PartnerServiceID   string `json:"partnerServiceId"`
	CustomerNo         string `json:"customerNo"`
	VirtualAccountNo   string `json:"virtualAccountNo"`
	VirtualAccountName string `json:"virtualAccountName"`
	Amount             string `json:"amount"`
	Currency           string `json:"currency"`
}

// VAInquiryResponse is returned by PaymentUsecase.GetVAInquiry.
type VAInquiryResponse struct {
	PartnerServiceID   string `json:"partnerServiceId"`
	CustomerNo         string `json:"customerNo"`
	VirtualAccountNo   string `json:"virtualAccountNo"`
	VirtualAccountName string `json:"virtualAccountName"`
	Amount             string `json:"amount"`
	Currency           string `json:"currency"`
}
