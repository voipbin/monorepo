package models

import (
	"github.com/gofrs/uuid"
)

// Call struct represent asterisk's channel information for client show
type Call struct {
	ID     uuid.UUID `json:"id"`      // Call's ID.
	UserID uint64    `json:"user_id"` // Call owner's ID.
	FlowID uuid.UUID `json:"flow_id"` // Attached flow id
	ConfID uuid.UUID `json:"conf_id"` // Currently joined conference id.
	Type   CallType  `json:"type"`    // Call's type.

	MasterCallID   uuid.UUID   `json:"master_call_id"`   // Master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // Chained call ids
	RecordingID    uuid.UUID   `json:"recording_id"`     // Recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids"`    // Recording ids

	Source      CallAddress `json:"source"`      // Source info
	Destination CallAddress `json:"destination"` // Destination info

	Status       CallStatus       `json:"status"`        // Call's status.
	Direction    CallDirection    `json:"direction"`     // Call's direction.
	HangupBy     CallHangupBy     `json:"hangup_by"`     // Describe which endpoint sent the hangup request first.
	HangupReason CallHangupReason `json:"hangup_reason"` // Desribe detail of hangup reason.
	WebhookURI   string           `json:"webhook_uri"`   // Webhook destination uri

	TMCreate string `json:"tm_create"` // Timestamp. Created time.
	TMUpdate string `json:"tm_update"` // Timestamp. Updated time.

	TMProgressing string `json:"tm_progressing"` // Timestamp. Progressing time.
	TMRinging     string `json:"tm_ringing"`     // Timestamp. Ringing time.
	TMHangup      string `json:"tm_hangup"`      // Timestamp. Hangup time.
}

// CallType type
type CallType string

// List of CallType
const (
	CallTypeNone       CallType = ""
	CallTypeFlow       CallType = "flow"        // executing the call-flow
	CallTypeConference CallType = "conference"  // conference call.
	CallTypeSipService CallType = "sip-service" // sip-service call. Will execute the corresponding the pre-defined sip-service by the destination.
)

// CallAddressType type
type CallAddressType string

// List of CallAddressType
const (
	CallAddressTypeSIP CallAddressType = "sip"
	CallAddressTypeTel CallAddressType = "tel"
)

// CallAddress contains source/destination detail info.
type CallAddress struct {
	Type   CallAddressType `json:"type"`   // Type of address. must be one of ["sip", "tel"].
	Target string          `json:"target"` // Destination. If the type is 'tel' type, the terget must follow the E.164 format(https://www.itu.int/rec/T-REC-E.164/en).
	Name   string          `json:"name"`   // Name.
}

// CallStatus type
type CallStatus string

// List of CallStatus
const (
	CallStatusDialing     CallStatus = "dialing"     // The call is created. We are dialing to the destination.
	CallStatusRinging     CallStatus = "ringing"     // The destination has confirmed that the call is ringng.
	CallStatusProgressing CallStatus = "progressing" // The call has answered. The both endpoints are talking to each other.
	CallStatusTerminating CallStatus = "terminating" // The call is terminating.
	CallStatusCanceling   CallStatus = "canceling"   // The call originator is canceling the call.
	CallStatusHangup      CallStatus = "hangup"      // The call has been completed.
)

// CallDirection type
type CallDirection string

// List of CallDirection
const (
	CallDirectionIncoming CallDirection = "incoming" // Call comming from outside of the voipbin.
	CallDirectionOutgoing CallDirection = "outgoing" // Call is generating from the voipbin.
)

// CallHangupBy type
type CallHangupBy string

// List of CallHangupBy
const (
	CallHangupByRemote CallHangupBy = "remote" // remote end hangup the call first.
	CallHangupByLocal  CallHangupBy = "local"  // local end hangup the call first.
)

// CallHangupReason type
type CallHangupReason string

// List of CallHangupReason
const (
	CallHangupReasonNormal   CallHangupReason = "normal"   // the call has ended after answer.
	CallHangupReasonFailed   CallHangupReason = "failed"   // the call attempt(signal) was not reached to the phone network.
	CallHangupReasonBusy     CallHangupReason = "busy"     // the destination is on the line with another caller.
	CallHangupReasonCanceled CallHangupReason = "cancel"   // call was cancelled by the originator before it was answered.
	CallHangupReasonTimeout  CallHangupReason = "timeout"  // call reached max call duration after it was answered.
	CallHangupReasonUnanswer CallHangupReason = "unanswer" // destination didn't answer until destination's timeout.
	CallHanupgReasonDialout  CallHangupReason = "dialout"  // The call reached dialing timeout before it was answered. This timeout is fired by our time out(outgoing call).
)
