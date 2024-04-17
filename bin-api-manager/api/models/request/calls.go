package request

import (
	cmcall "monorepo/bin-call-manager/models/call"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// BodyCallsPOST is rquest body define for
// POST /v1.0/calls
type BodyCallsPOST struct {
	Source       commonaddress.Address   `json:"source" binding:"required"`
	Destinations []commonaddress.Address `json:"destinations" binding:"required"`
	FlowID       uuid.UUID               `json:"flow_id"`
	Actions      []fmaction.Action       `json:"actions"`
}

// ParamCallsGET is rquest param define for
// GET /v1.0/calls
type ParamCallsGET struct {
	Pagination
}

// BodyCallsIDTranscribePOST is rquest body define for
// POST /v1.0/calls/<call-id>/transcribe
type BodyCallsIDTranscribePOST struct {
	Source       commonaddress.Address   `json:"source" binding:"required"`
	Destinations []commonaddress.Address `json:"destinations" binding:"required"`
	FlowID       uuid.UUID               `json:"flow_id"`
	Actions      []fmaction.Action       `json:"actions"`
}

// BodyCallsIDTalkPOST is rquest body define for
// POST /v1.0/calls/<call-id>/talk
type BodyCallsIDTalkPOST struct {
	Text     string `json:"text"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
}

// BodyCallsIDMutePost is data type for
// POST /v1.0/calls/<call-id>/mute
type BodyCallsIDMutePost struct {
	Direction cmcall.MuteDirection `json:"direction"`
}

// BodyCallsIDMuteDelete is data type for
// DELETE /v1.0/calls/<call-id>/mute
type BodyCallsIDMuteDelete struct {
	Direction cmcall.MuteDirection `json:"direction"`
}

// ParamCallsIDMediaStreamGET is rquest param define for
// GET /v1.0/calls/<call-id>/media_stream
type ParamCallsIDMediaStreamGET struct {
	Encapsulation string `form:"encapsulation"`
}
