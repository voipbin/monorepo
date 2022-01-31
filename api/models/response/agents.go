package response

import amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

// BodyAgentsGET is rquest body define for GET /agents
type BodyAgentsGET struct {
	Result []*amagent.WebhookMessage `json:"result"`
	Pagination
}
