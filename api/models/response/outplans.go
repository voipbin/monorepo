package response

import caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"

// BodyOutplansGET is rquest body define for GET /outplans
type BodyOutplansGET struct {
	Result []*caoutplan.WebhookMessage `json:"result"`
	Pagination
}
