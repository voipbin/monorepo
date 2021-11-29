package response

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/agent"
)

// BodyAgentsGET is rquest body define for GET /agents
type BodyAgentsGET struct {
	Result []*agent.Agent `json:"result"`
	Pagination
}
