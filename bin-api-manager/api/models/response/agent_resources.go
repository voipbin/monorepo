package response

import (
	amresource "monorepo/bin-agent-manager/models/resource"
)

// BodyAgentResourcesGET is rquest body define for
// GET /v1.0/agent_resources
type BodyAgentResourcesGET struct {
	Result []*amresource.WebhookMessage `json:"result"`
	Pagination
}
