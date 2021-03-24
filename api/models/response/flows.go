package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"

// BodyFlowsGET is rquest body define for GET /flows
type BodyFlowsGET struct {
	Result []*flow.Flow `json:"result"`
	Pagination
}
