package response

import (
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

// BodyActiveflowsGET is response body define for
// GET /v1.0/activeflows
type BodyActiveflowsGET struct {
	Result []*fmactiveflow.WebhookMessage `json:"result"`
	Pagination
}
