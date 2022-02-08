package request

import (
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// BodyCallsPOST is rquest body define for POST /calls
type BodyCallsPOST struct {
	Source      cmaddress.Address `json:"source" binding:"required"`
	Destination cmaddress.Address `json:"destination" binding:"required"`
	Actions     []fmaction.Action `json:"actions"`
}

// ParamCallsGET is rquest param define for GET /calls
type ParamCallsGET struct {
	Pagination
}
