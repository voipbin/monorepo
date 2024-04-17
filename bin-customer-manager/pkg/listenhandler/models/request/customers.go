package request

import (
	bmbilling "monorepo/bin-billing-manager/models/billing"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

// V1DataCustomersPost is
// v1 data type request struct for
// /v1/customers POST
type V1DataCustomersPost struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Address     string `json:"address"`

	WebhookMethod customer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                 `json:"webhook_uri,omitempty"`
}

// V1DataCustomersIDPut is
// v1 data type request struct for
// /v1/customers/<customer-id> PUT
type V1DataCustomersIDPut struct {
	Name          string                 `json:"name,omitempty"`
	Detail        string                 `json:"detail,omitempty"`
	Email         string                 `json:"email,omitempty"`
	PhoneNumber   string                 `json:"phone_number,omitempty"`
	Address       string                 `json:"address,omitempty"`
	WebhookMethod customer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                 `json:"webhook_uri,omitempty"`
}

// V1DataCustomersIDBillingAccountIDPut is
// v1 data type request struct for
// /v1/customers/<customer-id>/billing_account_id PUT
type V1DataCustomersIDBillingAccountIDPut struct {
	BillingAccountID uuid.UUID `json:"billing_account_id"`
}

// V1DataCustomersIDIsValidBalancePost is rquest param define for POST /customers/<customer-id>/is_valid_balance
type V1DataCustomersIDIsValidBalancePost struct {
	ReferenceType bmbilling.ReferenceType `json:"reference_type"`
	Country       string                  `json:"country"`
	Count         int                     `json:"count"`
}
