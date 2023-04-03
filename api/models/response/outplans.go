package response

import caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"

// BodyOutplansGET is rquest body define for
// GET /v1.0/outplans
type BodyOutplansGET struct {
	Result []*caoutplan.WebhookMessage `json:"result"`
	Pagination
}
