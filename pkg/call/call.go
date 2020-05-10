package call

import (
	"strings"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// Call struct represent asterisk's channel information
type Call struct {
	// identity
	ID         uuid.UUID
	AsteriskID string
	ChannelID  string
	FlowID     uuid.UUID // flow id
	ConfID     uuid.UUID // currently joined conference id
	Type       Type

	// source/destination
	Source      *Address
	Destination *Address

	// info
	Status       Status
	Data         map[string]interface{}
	Action       action.Action
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

// NewCall creates a call struct and return it.
func NewCall(
	id uuid.UUID,
	asteriskID string,
	channelID string,
	flowID uuid.UUID,
	cType Type,

	source *Address,
	destination *Address,

	status Status,
	data map[string]interface{},
	direction Direction,

	tmCreate string,
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

		TMCreate: tmCreate,
	}

	return c
}

// NewCallByChannel creates a Call and return it.
func NewCallByChannel(cn *channel.Channel, cType Type, direction Direction) *Call {
	// create a call
	source := CreateAddressByChannelSource(cn)
	destination := CreateAddressByChannelDestination(cn)
	status := GetStatusByChannelState(cn.State)
	data := map[string]interface{}{}

	for k, v := range cn.Data {
		data[strings.ToLower(k)] = v
	}

	c := NewCall(
		uuid.Must(uuid.NewV4()),
		cn.AsteriskID,
		cn.ID,
		uuid.Nil,
		cType,

		source,
		destination,

		status,
		data,
		direction,

		string(cn.TMCreate),
	)

	return c
}

// CreateAddressByChannelSource creates and return the Address using channel's source.
func CreateAddressByChannelSource(cn *channel.Channel) *Address {
	r := &Address{
		Type:   AddressTypeTel,
		Target: cn.SourceNumber,
		Name:   cn.SourceName,
	}
	return r
}

// CreateAddressByChannelDestination creates and return the Address using channel's destination.
func CreateAddressByChannelDestination(cn *channel.Channel) *Address {
	r := &Address{
		Type:   AddressTypeTel,
		Target: cn.DestinationNumber,
		Name:   cn.DestinationName,
	}
	return r
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

// NewAddressByDialplan parsing the ari's CallerID and returns Address
func NewAddressByDialplan(e *ari.DialplanCEP) *Address {
	r := &Address{
		Type:   AddressTypeTel,
		Target: e.Exten,
	}

	return r
}

// GetStatusByChannelState return Status by the ChannelState.
func GetStatusByChannelState(state ari.ChannelState) Status {

	mapParse := map[ari.ChannelState]Status{
		ari.ChannelStateDown:           StatusDialing,
		ari.ChannelStateRsrvd:          StatusDialing,
		ari.ChannelStateOffHook:        StatusDialing,
		ari.ChannelStateDialing:        StatusDialing,
		ari.ChannelStateBusy:           StatusDialing,
		ari.ChannelStateDialingOffHook: StatusDialing,
		ari.ChannelStatePreRing:        StatusDialing,
		ari.ChannelStateUnknown:        StatusDialing,

		ari.ChannelStateRinging: StatusRinging,
		ari.ChannelStateRing:    StatusRinging,

		ari.ChannelStateUp:   StatusProgressing,
		ari.ChannelStateMute: StatusProgressing,
	}

	res, ok := mapParse[state]
	if !ok {
		return StatusHangup
	}
	return res
}

// CalculateHangupReason calculates call hangup reason based on current status and hangup cause
func CalculateHangupReason(lastStatus Status, cause ari.ChannelCause) HangupReason {
	// Hangup reason calculate table
	//
	// +----------------------+-------+-----------------------+
	// | last status					| cuase	| hangup reason					|
	// |----------------------+-------+-----------------------+
	// | StatusDialing				| ?			| HangupReasonFailed		|
	// | StatusRinging				| ?			| HangupReasonBusy			|
	// |											| ?			| HangupReasonTimeout		|
	// |											| ?			| HangupReasonUnanswer	|
	// |											| ?			| HanupgReasonDialout		|
	// +----------------------+-------+-----------------------+
	// | StatusProgressing		| *			| HangupReasonNormal		|
	// +----------------------+-------+-----------------------+
	// | StatusTerminating		| * 		| HangupReasonNormal		|
	// +----------------------+-------+-----------------------+
	// | StatusCanceling			| * 		| HangupReasonCanceled	|
	// +----------------------+-------+-----------------------+
	// | StatusHangup					| * 		| HangupReasonNormal		|
	// +----------------------+-------+-----------------------+

	switch lastStatus {
	case StatusProgressing, StatusTerminating, StatusHangup:
		return HangupReasonNormal
	case StatusCanceling:
		return HangupReasonCanceled
	}

	// TODO: Need to be fixed as above chart.
	return HangupReasonFailed
}

// CalculateHangupBy calculates call hangupBy based on current status and hangup cause
func CalculateHangupBy(lastStatus Status) HangupBy {
	// Hangup by calculate table
	//
	// +----------------------+-----------------+
	// | last status					| Hangup by				|
	// |----------------------+-----------------+
	// | StatusDialing				| HangupByRemote	|
	// | StatusRinging				| 								|
	// | StatusProgressing		|									|
	// +----------------------+-----------------+
	// | StatusTerminating		| HangupByLocal		|
	// | StatusCanceling			|									|
	// | StatusHangup					|									|
	// +----------------------+-----------------+

	switch lastStatus {
	case StatusDialing, StatusRinging, StatusProgressing:
		return HangupByRemote
	case StatusTerminating, StatusCanceling, StatusHangup:
		return HangupByLocal
	default:
		return HangupByRemote
	}
}

// IsUpdatableStatus returns true if the new status is updatable.
func IsUpdatableStatus(oldStatus, newStatus Status) bool {

	// Possible scenarios

	// StatusDialing -> StatusRinging
	// StatusDialing -> StatusProgressing
	// StatusDialing -> StatusCanceling
	// StatusDialing -> StatusTerminating
	// StatusDialing -> StatusHangup
	// StatusRinging -> StatusProgressing
	// StatusRinging -> StatusCanceling
	// StatusDialing -> StatusTerminating
	// StatusDialing -> StatusHangup
	// StatusProgressing -> StatusTerminating
	// StatusProgressing -> StatusHangup
	// StatusCanceling -> StatusHangup
	// StatusTerminating -> StatusHangup

	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | old \ new					| StatusDialing	| StatusRinging	| StatusProgressing	| StatusTerminating	| StatusCanceling	| StatusHangup	|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusDialing			| 			x				| 			o				|					o					|					o					|					o				|				o				|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusRinging			|				x				| 			x				|					o					|					o					|					o				|				o				|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusProgressing	|				x				| 			x				|					x					|					o					|					x				|				o				|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusTerminating	|				x				| 			x				|					x					|					x					|					x				|				o				|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusCanceling		|				x				| 			x				|					x					|					x					|					x				|				o				|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusHangup				|				x				| 			x				|					x					|					x					|					x				|				x				|
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+

	mapOldStatusDialing := map[Status]bool{
		StatusDialing:     false,
		StatusRinging:     true,
		StatusProgressing: true,
		StatusTerminating: true,
		StatusCanceling:   true,
		StatusHangup:      true,
	}
	mapOldStatusRinging := map[Status]bool{
		StatusDialing:     false,
		StatusRinging:     false,
		StatusProgressing: true,
		StatusTerminating: true,
		StatusCanceling:   true,
		StatusHangup:      true,
	}
	mapOldStatusProgressing := map[Status]bool{
		StatusDialing:     false,
		StatusRinging:     false,
		StatusProgressing: false,
		StatusTerminating: true,
		StatusCanceling:   false,
		StatusHangup:      true,
	}
	mapOldStatusTerminating := map[Status]bool{
		StatusDialing:     false,
		StatusRinging:     false,
		StatusProgressing: false,
		StatusTerminating: false,
		StatusCanceling:   false,
		StatusHangup:      true,
	}
	mapOldStatusCanceling := map[Status]bool{
		StatusDialing:     false,
		StatusRinging:     false,
		StatusProgressing: false,
		StatusTerminating: false,
		StatusCanceling:   false,
		StatusHangup:      true,
	}

	// return false if change is not valid
	if oldStatus == newStatus || oldStatus == StatusHangup {
		return false
	}

	switch oldStatus {
	case StatusDialing:
		return mapOldStatusDialing[newStatus]
	case StatusRinging:
		return mapOldStatusRinging[newStatus]
	case StatusProgressing:
		return mapOldStatusProgressing[newStatus]
	case StatusTerminating:
		return mapOldStatusTerminating[newStatus]
	case StatusCanceling:
		return mapOldStatusCanceling[newStatus]
	}

	// should not reach to here
	return false
}
