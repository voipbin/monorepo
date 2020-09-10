package call

import (
	"github.com/gofrs/uuid"
)

// Call struct represent asterisk's channel information for client show
type Call struct {
	// identity
	ID uuid.UUID `json:"id"`
	// AsteriskID string    `json:"asterisk_id"`
	// ChannelID  string    `json:"channel_id"`
	UserID uint64    `json:"user_id"`
	FlowID uuid.UUID `json:"flow_id"` // flow id
	ConfID uuid.UUID `json:"conf_id"` // currently joined conference id
	Type   Type      `json:"type"`

	// source/destination
	Source      Address `json:"source"`
	Destination Address `json:"destination"`

	// info
	Status Status `json:"status"`
	// Data         map[string]interface{} `json:"data"`
	// Action       action.Action `json:"action"`
	Direction    Direction    `json:"direction"`
	HangupBy     HangupBy     `json:"hangup_by"`
	HangupReason HangupReason `json:"hangup_reason"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`

	TMProgressing string `json:"tm_progressing"`
	TMRinging     string `json:"tm_ringing"`
	TMHangup      string `json:"tm_hangup"`
}

// Type type
type Type string

// List of CallType
const (
	TypeNone       Type = ""
	TypeFlow       Type = "flow"        // executing the call-flow
	TypeConference Type = "conference"  // conference call.
	TypeSipService Type = "sip-service" // sip-service call. Will execute the corresponding the pre-defined sip-service by the destination.
)

// AddressType type
type AddressType string

// List of CallAddressType
const (
	AddressTypeSIP AddressType = "sip"
	AddressTypeTel AddressType = "tel"
)

// Address contains source/destination detail info.
type Address struct {
	Type   AddressType `json:"type"`   // type of address
	Target string      `json:"target"` // parsed destination
	Name   string      `json:"name"`   // parsed name
}

// Status type
type Status string

// List of CallStatus
const (
	StatusDialing     Status = "dialing"     // The call is created. We are dialing to the destination.
	StatusRinging     Status = "ringing"     // The destination has confirmed that the call is ringng.
	StatusProgressing Status = "progressing" // The call has answered. The both endpoints are talking to each other.
	StatusTerminating Status = "terminating" // The call is terminating.
	StatusCanceling   Status = "canceling"   // The call originator is canceling the call.
	StatusHangup      Status = "hangup"      // The call has been completed.
)

// Direction type
type Direction string

// List of CallDirection
const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

// HangupBy type
type HangupBy string

// List of CallHangupBy
const (
	HangupByRemote HangupBy = "remote" // remote end hangup the call first.
	HangupByLocal  HangupBy = "local"  // local end hangup the call first.
)

// HangupReason type
type HangupReason string

// List of CallHangupReason
const (
	HangupReasonNormal   HangupReason = "normal"   // the call has ended after answer.
	HangupReasonFailed   HangupReason = "failed"   // the call attempt(signal) was not reached to the phone network.
	HangupReasonBusy     HangupReason = "busy"     // the destination is on the line with another caller.
	HangupReasonCanceled HangupReason = "cancel"   // call was cancelled by the originator before it was answered.
	HangupReasonTimeout  HangupReason = "timeout"  // call reached max call duration after it was answered.
	HangupReasonUnanswer HangupReason = "unanswer" // destination didn't answer until destination's timeout.
	HanupgReasonDialout  HangupReason = "dialout"  // The call reached dialing timeout before it was answered. This timeout is fired by our time out(outgoing call).
)
