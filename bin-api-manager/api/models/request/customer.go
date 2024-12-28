package request

import (
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
)

// BodyCustomerPUT is rquest body define for
// PUT /v1.0/customer
type BodyCustomerPUT struct {
	Name          string                   `json:"name,omitempty"`
	Detail        string                   `json:"detail,omitempty"`
	Email         string                   `json:"email,omitempty"`
	PhoneNumber   string                   `json:"phone_number,omitempty"`
	Address       string                   `json:"address,omitempty"`
	WebhookMethod cscustomer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                   `json:"webhook_uri,omitempty"`
}

// BodyCustomerBillingAccountIDPUT is rquest body define for
// PUT /v1.0/customer/billing_account_id
type BodyCustomerBillingAccountIDPUT struct {
	BillingAccountID uuid.UUID `json:"billing_account_id"`
}
