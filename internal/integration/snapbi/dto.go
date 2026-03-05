package snapbi

// AccessTokenRequest is sent to obtain a B2B access token.
type AccessTokenRequest struct {
	GrantType string `json:"grantType"`
}

// AccessTokenResponse is the response from the B2B access token endpoint.
type AccessTokenResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AccessToken     string `json:"accessToken"`
	TokenType       string `json:"tokenType"`
	ExpiresIn       string `json:"expiresIn"`
}

// TransferVARequest is sent to initiate a VA payment.
type TransferVARequest struct {
	PartnerServiceID   string         `json:"partnerServiceId"`
	CustomerNo         string         `json:"customerNo"`
	VirtualAccountNo   string         `json:"virtualAccountNo"`
	VirtualAccountName string         `json:"virtualAccountName"`
	TrxID              string         `json:"trxId"`
	TotalAmount        AmountDetail   `json:"totalAmount"`
	AdditionalInfo     map[string]any `json:"additionalInfo,omitempty"`
}

// TransferVAResponse is the response from the VA payment endpoint.
type TransferVAResponse struct {
	ResponseCode       string  `json:"responseCode"`
	ResponseMessage    string  `json:"responseMessage"`
	VirtualAccountData *VAData `json:"virtualAccountData,omitempty"`
}

// InquiryVARequest is sent to inquire about a VA.
type InquiryVARequest struct {
	PartnerServiceID string         `json:"partnerServiceId"`
	CustomerNo       string         `json:"customerNo"`
	VirtualAccountNo string         `json:"virtualAccountNo"`
	InquiryRequestID string         `json:"inquiryRequestId"`
	AdditionalInfo   map[string]any `json:"additionalInfo,omitempty"`
}

// InquiryVAResponse is the response from the VA inquiry endpoint.
type InquiryVAResponse struct {
	ResponseCode       string  `json:"responseCode"`
	ResponseMessage    string  `json:"responseMessage"`
	VirtualAccountData *VAData `json:"virtualAccountData,omitempty"`
}

// AmountDetail holds a monetary value and its currency.
type AmountDetail struct {
	Value    string `json:"value"`    // e.g. "10000.00"
	Currency string `json:"currency"` // e.g. "IDR"
}

// VAData holds virtual account detail data returned by SNAP BI.
type VAData struct {
	PartnerServiceID   string       `json:"partnerServiceId"`
	CustomerNo         string       `json:"customerNo"`
	VirtualAccountNo   string       `json:"virtualAccountNo"`
	VirtualAccountName string       `json:"virtualAccountName"`
	TotalAmount        AmountDetail `json:"totalAmount"`
}

// ErrorResponse represents a SNAP BI error payload.
type ErrorResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}
