package response

import (
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// BodyCallsGET is response body define for
// GET /v1.0/calls
type BodyCallsGET struct {
	Result []*cmcall.WebhookMessage `json:"result"`
	Pagination
}
