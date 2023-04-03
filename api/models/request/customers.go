package request

import (
	"github.com/gofrs/uuid"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// ParamCustomersGET is request param define for
// GET /v1.0/customers
type ParamCustomersGET struct {
	Pagination
}

// BodyCustomersPOST is rquest body define for
// POST /v1.0/customers
type BodyCustomersPOST struct {
	Username      string                   `json:"username"`
	Password      string                   `json:"password"`
	Name          string                   `json:"name"`
	Detail        string                   `json:"detail"`
	WebhookMethod cscustomer.WebhookMethod `json:"webhook_method"`
	WebhookURI    string                   `json:"webhook_uri"`
	LineSecret    string                   `json:"line_secret"`
	LineToken     string                   `json:"line_token"`
	PermissionIDs []uuid.UUID              `json:"permission_ids"`
}

// BodyCustomersIDPUT is rquest body define for
// PUT /v1.0/customers/<customer-id>
type BodyCustomersIDPUT struct {
	Name          string                   `json:"name"`
	Detail        string                   `json:"detail"`
	WebhookMethod cscustomer.WebhookMethod `json:"webhook_method"`
	WebhookURI    string                   `json:"webhook_uri"`
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

// BodyCustomersIDLineInfoPUT is rquest body define for
// PUT /v1.0/customers/<customer-id>/line_info
type BodyCustomersIDLineInfoPUT struct {
	LineSecret string `json:"line_secret"`
	LineToken  string `json:"line_token"`
}
