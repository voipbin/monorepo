package response

import (
	bmbilling "monorepo/bin-billing-manager/models/billing"
)

// BodyBillingsGET is response body define for
// GET /v1.0/billings
type BodyBillingsGET struct {
	Result []*bmbilling.WebhookMessage `json:"result"`
	Pagination
}
