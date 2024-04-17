package response

import caoutplan "monorepo/bin-campaign-manager/models/outplan"

// BodyOutplansGET is rquest body define for
// GET /v1.0/outplans
type BodyOutplansGET struct {
	Result []*caoutplan.WebhookMessage `json:"result"`
	Pagination
}
