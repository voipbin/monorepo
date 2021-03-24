package request

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/action"

// BodyFlowsPOST is rquest body define for POST /flows
type BodyFlowsPOST struct {
	Name       string          `json:"name"`
	Detail     string          `json:"detail"`
	WebhookURI string          `json:"webhook_uri"`
	Actions    []action.Action `json:"actions"`
}

// ParamFlowsGET is rquest param define for GET /flows
type ParamFlowsGET struct {
	Pagination
}

// BodyFlowsIDPUT is rquest body define for PUT /flows/{id}
type BodyFlowsIDPUT struct {
	Name       string          `json:"name"`
	Detail     string          `json:"detail"`
	WebhookURI string          `json:"webhook_uri"`
	Actions    []action.Action `json:"actions"`
}
