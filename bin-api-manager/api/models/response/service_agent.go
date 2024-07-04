package response

import (
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
)

// BodyServiceAgentCallsGET is response body define for
// GET /v1.0/service_agent/calls
type BodyServiceAgentCallsGET struct {
	Result []*cmcall.WebhookMessage `json:"result"`
	Pagination
}

// BodyServiceAgentCallsPOST is response body define for
// POST /v1.0/service_agent/calls
type BodyServiceAgentCallsPOST struct {
	Calls      []*cmcall.WebhookMessage      `json:"calls"`
	Groupcalls []*cmgroupcall.WebhookMessage `json:"groupcalls"`
}
