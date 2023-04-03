package response

import fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

// BodyFlowsGET is rquest body define for
// GET /v1.0/flows
type BodyFlowsGET struct {
	Result []*fmflow.WebhookMessage `json:"result"`
	Pagination
}
