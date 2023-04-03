package response

import (
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
)

// BodyProvidersGET is rquest body define for
// GET /v1.0/providers
type BodyProvidersGET struct {
	Result []*rmprovider.WebhookMessage `json:"result"`
	Pagination
}
