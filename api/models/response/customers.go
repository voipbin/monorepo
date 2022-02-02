package response

import (
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// BodyCustomersGET is rquest body define for GET /customers
type BodyCustomersGET struct {
	Result []*cscustomer.WebhookMessage `json:"result"`
	Pagination
}
