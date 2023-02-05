package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// BodyCallsPOST is rquest body define for POST /calls
type BodyCallsPOST struct {
	Source       commonaddress.Address   `json:"source" binding:"required"`
	Destinations []commonaddress.Address `json:"destinations" binding:"required"`
	FlowID       uuid.UUID               `json:"flow_id"`
	Actions      []fmaction.Action       `json:"actions"`
}

// ParamCallsGET is rquest param define for GET /calls
type ParamCallsGET struct {
	Pagination
}

// BodyCallsIDTranscribePOST is rquest body define for POST /calls/<call-id>/transcribe
type BodyCallsIDTranscribePOST struct {
	Source       commonaddress.Address   `json:"source" binding:"required"`
	Destinations []commonaddress.Address `json:"destinations" binding:"required"`
	FlowID       uuid.UUID               `json:"flow_id"`
	Actions      []fmaction.Action       `json:"actions"`
}

// BodyCallsIDTalkPOST is rquest body define for POST /calls/<call-id>/talk
type BodyCallsIDTalkPOST struct {
	Text     string `json:"text"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
}
