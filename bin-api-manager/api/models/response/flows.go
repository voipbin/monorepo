package response

import fmflow "monorepo/bin-flow-manager/models/flow"

// BodyFlowsGET is rquest body define for
// GET /v1.0/flows
type BodyFlowsGET struct {
	Result []*fmflow.WebhookMessage `json:"result"`
	Pagination
}
