package response

import (
	rmprovider "monorepo/bin-route-manager/models/provider"
)

// BodyProvidersGET is rquest body define for
// GET /v1.0/providers
type BodyProvidersGET struct {
	Result []*rmprovider.WebhookMessage `json:"result"`
	Pagination
}
