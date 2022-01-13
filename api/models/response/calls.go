package response

import (
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// BodyCallsGET is response body define for GET /calls
type BodyCallsGET struct {
	Result []*cmcall.WebhookMessage `json:"result"`
	Pagination
}
