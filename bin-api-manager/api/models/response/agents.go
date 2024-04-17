package response

import amagent "monorepo/bin-agent-manager/models/agent"

// BodyAgentsGET is rquest body define for
// GET /v1.0/agents
type BodyAgentsGET struct {
	Result []*amagent.WebhookMessage `json:"result"`
	Pagination
}
