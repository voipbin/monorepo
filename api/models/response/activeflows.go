package response

import (
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

// BodyActiveflowsGET is response body define for GET /activeflows
type BodyActiveflowsGET struct {
	Result []*fmactiveflow.WebhookMessage `json:"result"`
	Pagination
}
