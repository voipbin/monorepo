package customer

import "monorepo/bin-customer-manager/models/accesskey"

// SignupResult contains the result of a signup operation.
type SignupResult struct {
	Customer  *Customer `json:"customer,omitempty"`
	TempToken string    `json:"temp_token,omitempty"`
}

// CompleteSignupResult contains the result of a headless signup completion.
type CompleteSignupResult struct {
	CustomerID string             `json:"customer_id"`
	Accesskey  *accesskey.Accesskey `json:"accesskey,omitempty"`
}

// EmailVerifyResult contains the result of an email verification.
type EmailVerifyResult struct {
	Customer  *Customer            `json:"customer,omitempty"`
	Accesskey *accesskey.Accesskey `json:"accesskey,omitempty"`
}
