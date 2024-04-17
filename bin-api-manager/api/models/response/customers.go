package response

import (
	cscustomer "monorepo/bin-customer-manager/models/customer"
)

// BodyCustomersGET is rquest body define for
// GET /v1.0/customers
type BodyCustomersGET struct {
	Result []*cscustomer.WebhookMessage `json:"result"`
	Pagination
}
