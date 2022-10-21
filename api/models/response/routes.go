package response

import (
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"
)

// BodyRoutesGET is rquest body define for GET /routes
type BodyRoutesGET struct {
	Result []*rmroute.WebhookMessage `json:"result"`
	Pagination
}
