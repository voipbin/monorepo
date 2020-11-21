package call

import (
	"fmt"
	"reflect"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
)

// Call struct represent asterisk's channel information
type Call struct {
	// identity
	ID         uuid.UUID `json:"id"`
	UserID     uint64    `json:"user_id"`
	AsteriskID string    `json:"asterisk_id"`
	ChannelID  string    `json:"channel_id"`
	FlowID     uuid.UUID `json:"flow_id"` // flow id
	ConfID     uuid.UUID `json:"conf_id"` // currently joined conference id
	Type       Type      `json:"type"`    // call type

	// etc info
	MasterCallID   uuid.UUID   `json:"master_call_id"`   // master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // chained call ids
	RecordID       string      `json:"record_id"`        // record id(current)
	RecordIDs      []string    `json:"record_ids"`       // record ids

	// source/destination
	Source      Address `json:"source"`
	Destination Address `json:"destination"`

	// info
	Status       Status                 `json:"status"`
	Data         map[string]interface{} `json:"data"`
	Action       action.Action          `json:"action"`
	Direction    Direction              `json:"direction"`
	HangupBy     HangupBy               `json:"hangup_by"`
	HangupReason HangupReason           `json:"hangup_reason"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`

	TMProgressing string `json:"tm_progressing"`
	TMRinging     string `json:"tm_ringing"`
	TMHangup      string `json:"tm_hangup"`
}

// fixed user id
const (
	UserIDAdmin uint64 = 1 // admin user id
)

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

// Test values
const (
	TestChannelID string = "00000000-0000-0000-0000-000000000000"
)

// Matches return true if the given items are the same
func (a *Call) Matches(x interface{}) bool {
	comp := x.(*Call)
	c := *a

	if c.ChannelID == TestChannelID {
		c.ChannelID = comp.ChannelID
	}
	c.TMCreate = comp.TMCreate

	return reflect.DeepEqual(c, *comp)
}

func (a *Call) String() string {
	return fmt.Sprintf("%v", *a)
}

// NewCall creates a call struct and return it.
func NewCall(
	id uuid.UUID,
	userID uint64,
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
		UserID:     userID,
		AsteriskID: asteriskID,
		ChannelID:  channelID,
		FlowID:     flowID,
		Type:       cType,

		ChainedCallIDs: []uuid.UUID{},
		RecordIDs:      []string{},

		Source:      *source,
		Destination: *destination,

		Status:    status,
		Data:      data,
		Direction: direction,

		TMCreate: tmCreate,
	}

	return c
}

// NewCallByChannel creates a Call and return it.
func NewCallByChannel(cn *channel.Channel, userID uint64, cType Type, direction Direction, data map[string]interface{}) *Call {
	// create a call
	source := CreateAddressByChannelSource(cn)
	destination := CreateAddressByChannelDestination(cn)
	status := GetStatusByChannelState(cn.State)

	c := &Call{
		ID:         uuid.Must(uuid.NewV4()),
		UserID:     userID,
		AsteriskID: cn.AsteriskID,
		ChannelID:  cn.ID,
		FlowID:     uuid.Nil,
		Type:       cType,

		ChainedCallIDs: []uuid.UUID{},
		RecordIDs:      []string{},

		Source:      *source,
		Destination: *destination,

		Status:    status,
		Data:      data,
		Direction: direction,

		TMCreate: string(cn.TMCreate),
	}

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
	// | last status          | cause | hangup reason         |
	// |----------------------+-------+-----------------------+
	// | StatusDialing        | ?     | HangupReasonFailed    |
	// | StatusRinging        | ?     | HangupReasonBusy      |
	// |                      | ?     | HangupReasonTimeout   |
	// |                      | ?     | HangupReasonUnanswer  |
	// |                      | ?     | HanupgReasonDialout   |
	// +----------------------+-------+-----------------------+
	// | StatusProgressing    | *     | HangupReasonNormal    |
	// +----------------------+-------+-----------------------+
	// | StatusTerminating    | *     | HangupReasonNormal    |
	// +----------------------+-------+-----------------------+
	// | StatusCanceling      | *     | HangupReasonCanceled  |
	// +----------------------+-------+-----------------------+
	// | StatusHangup         | *     | HangupReasonNormal    |
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
	// | last status          | Hangup by       |
	// |----------------------+-----------------+
	// | StatusDialing        | HangupByRemote  |
	// | StatusRinging        |                 |
	// | StatusProgressing    |                 |
	// | StatusHangup         |                 |
	// +----------------------+-----------------+
	// | StatusTerminating    | HangupByLocal   |
	// | StatusCanceling      |                 |
	// +----------------------+-----------------+

	switch lastStatus {
	case StatusDialing, StatusRinging, StatusProgressing, StatusHangup:
		return HangupByRemote
	case StatusTerminating, StatusCanceling:
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
	// StatusRinging -> StatusTerminating
	// StatusRinging -> StatusHangup
	// StatusProgressing -> StatusTerminating
	// StatusProgressing -> StatusHangup
	// StatusCanceling -> StatusHangup
	// StatusTerminating -> StatusHangup

	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | old \ new          | StatusDialing	| StatusRinging	| StatusProgressing	| StatusTerminating	| StatusCanceling | StatusHangup  |
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusDialing      |      x        |       o       |         o         |         o         |        o        |       o       |
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusRinging      |      x        |       x       |         o         |         o         |        o        |       o       |
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusProgressing  |      x        |       x       |         x         |         o         |        x        |       o       |
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusTerminating  |      x        |       x       |         x         |         x         |        x        |       o       |
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusCanceling    |      x        |       x       |         x         |         x         |        x        |       o       |
	// |--------------------+---------------+---------------+-------------------+-------------------+-----------------+---------------+
	// | StatusHangup       |      x        |       x       |         x         |         x         |        x        |       x       |
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
