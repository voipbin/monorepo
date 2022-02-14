package request

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// BodyCallsPOST is rquest body define for POST /calls
type BodyCallsPOST struct {
	Source       cmaddress.Address   `json:"source" binding:"required"`
	Destinations []cmaddress.Address `json:"destinations" binding:"required"`
	FlowID       uuid.UUID           `json:"flow_id"`
	Actions      []fmaction.Action   `json:"actions"`
}

// ParamCallsGET is rquest param define for GET /calls
type ParamCallsGET struct {
	Pagination
}
