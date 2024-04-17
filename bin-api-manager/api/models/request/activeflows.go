package request

import (
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// BodyActiveflowsPOST is rquest body define for
// POST /v1.0/activeflows
type BodyActiveflowsPOST struct {
	ID      uuid.UUID         `json:"id"` // activeflow id
	FlowID  uuid.UUID         `json:"flow_id"`
	Actions []fmaction.Action `json:"actions"`
}

// ParamActiveflowsGET is rquest param define for
// GET /v1.0/activeflows
type ParamActiveflowsGET struct {
	Pagination
}
