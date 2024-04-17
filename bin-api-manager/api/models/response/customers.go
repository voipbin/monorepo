package response

import (
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// BodyCustomersGET is rquest body define for
// GET /v1.0/customers
type BodyCustomersGET struct {
	Result []*cscustomer.WebhookMessage `json:"result"`
	Pagination
}
