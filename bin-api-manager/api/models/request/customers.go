package request

import (
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
)

// ParamCustomersGET is request param define for
// GET /v1.0/customers
type ParamCustomersGET struct {
	Pagination
}

// BodyCustomersPOST is rquest body define for
// POST /v1.0/customers
type BodyCustomersPOST struct {
	Name          string                   `json:"name,omitempty"`
	Detail        string                   `json:"detail,omitempty"`
	Email         string                   `json:"email,omitempty"`
	PhoneNumber   string                   `json:"phone_number,omitempty"`
	Address       string                   `json:"address,omitempty"`
	WebhookMethod cscustomer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                   `json:"webhook_uri,omitempty"`
}

// BodyCustomersIDPUT is rquest body define for
// PUT /v1.0/customers/<customer-id>
type BodyCustomersIDPUT struct {
	Name          string                   `json:"name,omitempty"`
	Detail        string                   `json:"detail,omitempty"`
	Email         string                   `json:"email,omitempty"`
	PhoneNumber   string                   `json:"phone_number,omitempty"`
	Address       string                   `json:"address,omitempty"`
	WebhookMethod cscustomer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                   `json:"webhook_uri,omitempty"`
}

// BodyCustomersIDPasswordPUT is rquest body define for
// PUT /v1.0/customers/<customer-id>/password
type BodyCustomersIDPasswordPUT struct {
	Password string `json:"password"`
}

// BodyCustomersIDPermissionIDsPUT is rquest body define for
// PUT /v1.0/customers/<customer-id>/permission_ids
type BodyCustomersIDPermissionIDsPUT struct {
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// BodyCustomersIDBillingAccountIDPUT is rquest body define for
// PUT /v1.0/customers/<customer-id>/billing_account_id
type BodyCustomersIDBillingAccountIDPUT struct {
	BillingAccountID uuid.UUID `json:"billing_account_id"`
}
