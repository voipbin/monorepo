package response

import amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

// BodyAgentsGET is rquest body define for
// GET /v1.0/agents
type BodyAgentsGET struct {
	Result []*amagent.WebhookMessage `json:"result"`
	Pagination
}
