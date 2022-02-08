package request

import (
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// BodyFlowsPOST is rquest body define for POST /flows
type BodyFlowsPOST struct {
	Name    string            `json:"name"`
	Detail  string            `json:"detail"`
	Actions []fmaction.Action `json:"actions"`
}

// ParamFlowsGET is rquest param define for GET /flows
type ParamFlowsGET struct {
	Pagination
}

// BodyFlowsIDPUT is rquest body define for PUT /flows/{id}
type BodyFlowsIDPUT struct {
	Name    string            `json:"name"`
	Detail  string            `json:"detail"`
	Actions []fmaction.Action `json:"actions"`
}
