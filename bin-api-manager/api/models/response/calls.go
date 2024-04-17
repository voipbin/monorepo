package response

import (
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// BodyCallsPOST is response body define for
// POST /v1.0/calls
type BodyCallsPOST struct {
	Calls      []*cmcall.WebhookMessage      `json:"calls"`
	Groupcalls []*cmgroupcall.WebhookMessage `json:"groupcalls"`
}

// BodyCallsGET is response body define for
// GET /v1.0/calls
type BodyCallsGET struct {
	Result []*cmcall.WebhookMessage `json:"result"`
	Pagination
}
