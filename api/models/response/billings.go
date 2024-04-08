package response

import (
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
)

// BodyBillingsGET is response body define for
// GET /v1.0/billings
type BodyBillingsGET struct {
	Result []*bmbilling.WebhookMessage `json:"result"`
	Pagination
}
