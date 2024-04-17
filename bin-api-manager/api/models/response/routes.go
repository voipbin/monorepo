package response

import (
	rmroute "monorepo/bin-route-manager/models/route"
)

// BodyRoutesGET is rquest body define for
// GET /v1.0/routes
type BodyRoutesGET struct {
	Result []*rmroute.WebhookMessage `json:"result"`
	Pagination
}
