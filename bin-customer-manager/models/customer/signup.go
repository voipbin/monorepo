package customer

import "monorepo/bin-customer-manager/models/accesskey"

// SignupResult contains the result of a signup operation.
type SignupResult struct {
	Customer  *Customer `json:"customer,omitempty"`
	TempToken string    `json:"temp_token,omitempty"`
}

// SignupResultWebhookMessage is the external-facing version of SignupResult.
type SignupResultWebhookMessage struct {
	Customer  *WebhookMessage `json:"customer,omitempty"`
	TempToken string          `json:"temp_token,omitempty"`
}

// ConvertWebhookMessage converts SignupResult to its external-facing representation.
func (h *SignupResult) ConvertWebhookMessage() *SignupResultWebhookMessage {
	res := &SignupResultWebhookMessage{
		TempToken: h.TempToken,
	}
	if h.Customer != nil {
		res.Customer = h.Customer.ConvertWebhookMessage()
	}
	return res
}

// CompleteSignupResult contains the result of a headless signup completion.
type CompleteSignupResult struct {
	CustomerID string             `json:"customer_id"`
	Accesskey  *accesskey.Accesskey `json:"accesskey,omitempty"`
}

// CompleteSignupResultWebhookMessage is the external-facing version of CompleteSignupResult.
type CompleteSignupResultWebhookMessage struct {
	CustomerID string                   `json:"customer_id"`
	Accesskey  *accesskey.WebhookMessage `json:"accesskey,omitempty"`
}

// ConvertWebhookMessage converts CompleteSignupResult to its external-facing representation.
func (h *CompleteSignupResult) ConvertWebhookMessage() *CompleteSignupResultWebhookMessage {
	res := &CompleteSignupResultWebhookMessage{
		CustomerID: h.CustomerID,
	}
	if h.Accesskey != nil {
		res.Accesskey = h.Accesskey.ConvertWebhookMessage()
	}
	return res
}

// EmailVerifyResult contains the result of an email verification.
type EmailVerifyResult struct {
	Customer  *Customer            `json:"customer,omitempty"`
	Accesskey *accesskey.Accesskey `json:"accesskey,omitempty"`
}

// EmailVerifyResultWebhookMessage is the external-facing version of EmailVerifyResult.
type EmailVerifyResultWebhookMessage struct {
	Customer  *WebhookMessage          `json:"customer,omitempty"`
	Accesskey *accesskey.WebhookMessage `json:"accesskey,omitempty"`
}

// ConvertWebhookMessage converts EmailVerifyResult to its external-facing representation.
func (h *EmailVerifyResult) ConvertWebhookMessage() *EmailVerifyResultWebhookMessage {
	res := &EmailVerifyResultWebhookMessage{}
	if h.Customer != nil {
		res.Customer = h.Customer.ConvertWebhookMessage()
	}
	if h.Accesskey != nil {
		res.Accesskey = h.Accesskey.ConvertWebhookMessage()
	}
	return res
}
