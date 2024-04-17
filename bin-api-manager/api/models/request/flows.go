package request

import (
	fmaction "monorepo/bin-flow-manager/models/action"
)

// BodyFlowsPOST is rquest body define for
// POST /v1.0/flows
type BodyFlowsPOST struct {
	Name    string            `json:"name"`
	Detail  string            `json:"detail"`
	Actions []fmaction.Action `json:"actions"`
}

// ParamFlowsGET is rquest param define for
// GET /v1.0/flows
type ParamFlowsGET struct {
	Pagination
}

// BodyFlowsIDPUT is rquest body define for
// PUT /v1.0/flows/<flow-id>
type BodyFlowsIDPUT struct {
	Name    string            `json:"name"`
	Detail  string            `json:"detail"`
	Actions []fmaction.Action `json:"actions"`
}
