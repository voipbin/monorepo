package call

import (
	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

// Call struct represent asterisk's channel information
type Call struct {
	// identity
	ID         uuid.UUID
	AsteriskID string
	ChannelID  string
	FlowID     uuid.UUID
	Type       Type

	// source/destination
	Source      Address
	Destination Address

	// info
	Status       Status
	Data         map[string]string
	Direction    Direction
	HangupBy     HangupBy
	HangupReason HangupReason

	// timestamp
	TMCreate string
	TMUpdate string

	TMProgressing string
	TMRinging     string
	TMHangup      string
}

// Type type
type Type string

// List of CallType
const (
	TypeFlow Type = "flow"
	TypeEcho Type = "echo"
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
	Type   AddressType // type of address
	Target string      // parsed destination
	Name   string      // parsed name
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
	StatusHangup      Status = "hangup"      // The call has been terminated.
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

// NewCall creates a call struct and return it.
func NewCall(
	id uuid.UUID,
	asteriskID string,
	channelID string,
	flowID uuid.UUID,
	cType Type,

	source Address,
	destination Address,

	status Status,
	data map[string]string,
	direction Direction,

	tmCreate ari.Timestamp,
) *Call {

	c := &Call{
		ID:         id,
		AsteriskID: asteriskID,
		ChannelID:  channelID,
		FlowID:     flowID,
		Type:       cType,

		Source:      source,
		Destination: destination,

		Status:    status,
		Data:      data,
		Direction: direction,

		TMCreate: string(tmCreate),
	}

	return c
}

// ParseAddressByCallerID parsing the ari's CallerID and returns Address
func ParseAddressByCallerID(e *ari.CallerID) *Address {
	r := &Address{
		Type:   AddressTypeTel,
		Target: e.Number,
		Name:   e.Name,
	}

	return r
}

// ParseAddressByDialplan parsing the ari's CallerID and returns Address
func ParseAddressByDialplan(e *ari.DialplanCEP) *Address {
	r := &Address{
		Type:   AddressTypeTel,
		Target: e.Exten,
	}

	return r
}

// ParseStatusByChannelState return Status by the ChannelState
func ParseStatusByChannelState(state ari.ChannelState) Status {
	mapParse := map[ari.ChannelState]Status{
		ari.ChannelStateDown:           StatusHangup,
		ari.ChannelStateRsrvd:          StatusHangup,
		ari.ChannelStateOffHook:        StatusHangup,
		ari.ChannelStateDialing:        StatusDialing,
		ari.ChannelStateRing:           StatusRinging,
		ari.ChannelStateRinging:        StatusRinging,
		ari.ChannelStateUp:             StatusProgressing,
		ari.ChannelStateBusy:           StatusHangup,
		ari.ChannelStateDialingOffHook: StatusHangup,
		ari.ChannelStatePreRing:        StatusDialing,
		ari.ChannelStateMute:           StatusProgressing,
		ari.ChannelStateUnknown:        StatusHangup,
	}

	res, ok := mapParse[state]
	if !ok {
		return StatusHangup
	}

	return res
}
