package call

import (
	"fmt"
	"reflect"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	fmaction "monorepo/bin-flow-manager/models/action"

	rmroute "monorepo/bin-route-manager/models/route"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
)

// Call struct represent asterisk's channel information
type Call struct {
	commonidentity.Identity
	commonidentity.Owner

	ChannelID string `json:"channel_id,omitempty" db:"channel_id"`
	BridgeID  string `json:"bridge_id,omitempty" db:"bridge_id"` // call bridge id

	FlowID       uuid.UUID `json:"flow_id,omitempty" db:"flow_id,uuid"`             // flow id
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"` // activeflow id
	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty" db:"confbridge_id,uuid"` // currently joined confbridge id.

	Type Type `json:"type,omitempty" db:"type"` // call type

	// etc info
	MasterCallID    uuid.UUID   `json:"master_call_id,omitempty" db:"master_call_id,uuid"`     // master call id
	ChainedCallIDs  []uuid.UUID `json:"chained_call_ids,omitempty" db:"chained_call_ids,json"` // chained call ids
	RecordingID     uuid.UUID   `json:"recording_id,omitempty" db:"recording_id,uuid"`         // recording id(current)
	RecordingIDs    []uuid.UUID `json:"recording_ids,omitempty" db:"recording_ids,json"`       // recording ids
	ExternalMediaID uuid.UUID   `json:"external_media_id,omitempty" db:"external_media_id,uuid"` // external media id(current)
	GroupcallID     uuid.UUID   `json:"groupcall_id,omitempty" db:"groupcall_id,uuid"`           // groupcall id

	// source/destination
	Source      commonaddress.Address `json:"source,omitempty" db:"source,json"`
	Destination commonaddress.Address `json:"destination,omitempty" db:"destination,json"`

	// info
	Status         Status              `json:"status,omitempty" db:"status"`
	Data           map[DataType]string `json:"data,omitempty" db:"data,json"`
	Action         fmaction.Action     `json:"action,omitempty" db:"action,json"`             // call's current action.
	ActionNextHold bool                `json:"action_next_hold,omitempty" db:"action_next_hold"` // call's next action hold. if true, don't allow to go next action
	Direction      Direction           `json:"direction,omitempty" db:"direction"`               //  direction of call. incoming/outgoing
	MuteDirection  MuteDirection       `json:"mute_direction,omitempty" db:"mute_direction"`     // mute direction

	HangupBy     HangupBy     `json:"hangup_by,omitempty" db:"hangup_by"`
	HangupReason HangupReason `json:"hangup_reason,omitempty" db:"hangup_reason"`

	// dialroute(valid only tel type outgoing call)
	DialrouteID uuid.UUID       `json:"dialroute_id,omitempty" db:"dialroute_id,uuid"` // dialroute id(current use)
	Dialroutes  []rmroute.Route `json:"dialroutes,omitempty" db:"dialroutes,json"`     // list of dialroutes for dialing.

	TMRinging     *time.Time `json:"tm_ringing,omitempty" db:"tm_ringing"`
	TMProgressing *time.Time `json:"tm_progressing,omitempty" db:"tm_progressing"`
	TMHangup      *time.Time `json:"tm_hangup,omitempty" db:"tm_hangup"`

	// timestamp
	TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
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

// MuteDirection represents possible values for channel mute
type MuteDirection string

// List of mute direction types
const (
	MuteDirectionNone MuteDirection = ""     // none
	MuteDirectionBoth MuteDirection = "both" // mute the channel in/out both.
	MuteDirectionOut  MuteDirection = "out"  //
	MuteDirectionIn   MuteDirection = "in"   // mute the channel incoming
)

// DataType define
type DataType string

// list of DataType types.
const (
	// if it sets to true, the call will execute the flow when the ringing started.
	// because it starts the flow with ringing status, it is possible to could not handle the route failover correctly.
	DataTypeEarlyExecution DataType = "early_execution"

	// if it sets to true, the master call will move to the next action when the connect call is failed.
	// this is important if the call is connect call, the call will move to the confbridge(type connect) after call answer.
	// but if the call was failed and the call could not execute the action(which is confbridge join), the master call will wait in the
	// confbridge forever. So, we need to trigger the master call's next action manually if the call was failed.
	DataTypeExecuteNextMasterOnHangup DataType = "execute_next_master_on_hangup"
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
	HangupByNone   HangupBy = ""       // no one hangup yet.
	HangupByRemote HangupBy = "remote" // remote end hangup the call first.
	HangupByLocal  HangupBy = "local"  // local end hangup the call first.
)

// HangupReason type
type HangupReason string

// List of CallHangupReason
const (
	HangupReasonNone     HangupReason = ""
	HangupReasonNormal   HangupReason = "normal"   // the call has ended after answer.
	HangupReasonFailed   HangupReason = "failed"   // the call attempt(signal) was not reached to the phone network.
	HangupReasonBusy     HangupReason = "busy"     // the destination is on the line with another caller.
	HangupReasonCanceled HangupReason = "cancel"   // call was cancelled by the originator before it was answered.
	HangupReasonTimeout  HangupReason = "timeout"  // call reached max call duration after it was answered.
	HangupReasonNoanswer HangupReason = "noanswer" // The call rejected with noanswer status.
	HangupReasonDialout  HangupReason = "dialout"  // The call reached dialing timeout before it was answered. This timeout is fired by our time out(outgoing call).
	HangupReasonAMD      HangupReason = "amd"      // the call's amd action result hung up the call.
)

// Test values
const (
	TestChannelID string = "00000000-0000-0000-0000-000000000000"
)

// Matches return true if the given items are the same
func (h *Call) Matches(x interface{}) bool {
	comp := x.(*Call)
	c := *h

	if c.ChannelID == TestChannelID {
		c.ChannelID = comp.ChannelID
	}
	c.TMCreate = comp.TMCreate
	c.TMUpdate = comp.TMUpdate
	c.TMRinging = comp.TMRinging
	c.TMProgressing = comp.TMProgressing
	c.TMHangup = comp.TMHangup

	return reflect.DeepEqual(c, *comp)
}

func (h *Call) String() string {
	return fmt.Sprintf("%v", *h)
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
func CalculateHangupReason(direction Direction, lastStatus Status, cause ari.ChannelCause) HangupReason {

	if direction == DirectionOutgoing {
		return calculateHangupReasonDirectionOutgoing(lastStatus, cause)
	}

	return calculateHangupReasonDirectionIncoming(lastStatus, cause)
}

// calculateHangupReasonDirectionIncoming calculates incoming direction call's hangup reasone.
func calculateHangupReasonDirectionIncoming(lastStatus Status, cause ari.ChannelCause) HangupReason {

	// Hangup reason calculate table(incoming)
	//
	// +----------------------+---------------------------------+-----------------------+
	// | last status          | cause                           | hangup reason         |
	// |----------------------+---------------------------------+-----------------------+
	// | StatusDialing        | ChannelCauseNoAnswer            | HangupReasonNoanswer  |
	// | StatusRinging        | ChannelCauseUserBusy            | HangupReasonBusy      |
	// |                      | *                               | HangupReasonNormal    |
	// +----------------------+---------------------------------+-----------------------+
	// | StatusProgressing    | ChannelCauseCallProgressTimeout | HangupReasonTimeout   |
	// |                      | *                               | HangupReasonNormal    |
	// +----------------------+---------------------------------+-----------------------+
	// | *                    | *                               | HangupReasonNormal    |
	// +----------------------+---------------------------------+-----------------------+

	switch lastStatus {
	case StatusDialing, StatusRinging:
		switch cause {
		case ari.ChannelCauseNoAnswer:
			return HangupReasonNoanswer
		case ari.ChannelCauseUserBusy:
			return HangupReasonBusy
		default:
			return HangupReasonNormal
		}

	case StatusProgressing:
		if cause == ari.ChannelCauseCallDurationTimeout {
			return HangupReasonTimeout
		}
		return HangupReasonNormal

	default:
		return HangupReasonNormal
	}
}

// calculateHangupReasonDirectionOutgoing calculates outgoing direction call's hangup reasone.
func calculateHangupReasonDirectionOutgoing(lastStatus Status, cause ari.ChannelCause) HangupReason {

	// Hangup reason calculate table(outgoing)
	//
	// +----------------------+---------------------------------+-----------------------+
	// | last status          | cause                           | hangup reason         |
	// |----------------------+---------------------------------+-----------------------+
	// | StatusDialing        | ChannelCauseNoAnswer            | HangupReasonNoanswer  |
	// | StatusRinging        | ChannelCauseCallRejected        | HangupReasonNoanswer  |
	// |                      | ChannelCauseUserBusy            | HangupReasonBusy      |
	// |                      | ChannelCauseNormalClearing      | HangupReasonNormal    |
	// |                      | ChannelCauseAnsweredElsewhere   | HangupReasonNormal    |
	// |                      | ChannelCauseUnknown             | HangupReasonDialout   |
	// |                      | *                               | HangupReasonFailed    |
	// +----------------------+---------------------------------+-----------------------+
	// | StatusProgressing    | ChannelCauseCallDurationTimeout | HangupReasonTimeout   |
	// |                      | *                               | HangupReasonNormal    |
	// +----------------------+---------------------------------+-----------------------+
	// | StatusTerminating    | ChannelCauseCallAMD             | HangupReasonAMD       |
	// |                      | *                               | HangupReasonNormal    |
	// +----------------------+---------------------------------+-----------------------+
	// | StatusHangup         | *                               | HangupReasonNormal  |
	// +----------------------+---------------------------------+-----------------------+
	// | StatusCanceling      | *                               | HangupReasonCanceled  |
	// +----------------------+---------------------------------+-----------------------+
	// | *                    | *                               | HangupReasonNormal    |
	// +----------------------+---------------------------------+-----------------------+

	switch lastStatus {
	case StatusDialing, StatusRinging:
		switch cause {
		case ari.ChannelCauseNoAnswer, ari.ChannelCauseCallRejected:
			return HangupReasonNoanswer
		case ari.ChannelCauseUserBusy:
			return HangupReasonBusy
		case ari.ChannelCauseNormalClearing, ari.ChannelCauseAnsweredElsewhere:
			return HangupReasonNormal
		case ari.ChannelCauseUnknown:
			return HangupReasonDialout
		default:
			return HangupReasonFailed
		}

	case StatusProgressing:
		switch cause {
		case ari.ChannelCauseCallDurationTimeout:
			return HangupReasonTimeout
		default:
			return HangupReasonNormal
		}

	case StatusTerminating:
		switch cause {
		case ari.ChannelCauseCallAMD:
			return HangupReasonAMD
		default:
			return HangupReasonNormal
		}

	case StatusHangup:
		switch cause {
		default:
			return HangupReasonNormal
		}

	case StatusCanceling:
		return HangupReasonCanceled

	default:
		return HangupReasonNormal
	}
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

// ConvertHangupReasonToChannelCause returns ari channel cause code for call hanging up.
func ConvertHangupReasonToChannelCause(reason HangupReason) ari.ChannelCause {

	mapCause := map[HangupReason]ari.ChannelCause{
		HangupReasonNone:     ari.ChannelCauseNormalClearing,
		HangupReasonNormal:   ari.ChannelCauseNormalClearing,
		HangupReasonFailed:   ari.ChannelCauseNoUserResponse, // the call attempt(signal) was not reached to the phone network.
		HangupReasonBusy:     ari.ChannelCauseUserBusy,       // the destination is on the line with another caller.
		HangupReasonCanceled: ari.ChannelCauseNormalClearing, // call was cancelled by the originator before it was answered.
		HangupReasonTimeout:  ari.ChannelCauseNormalClearing, // call reached max call duration after it was answered.
		HangupReasonNoanswer: ari.ChannelCauseNoAnswer,       // The call rejected with noanswer status.
		HangupReasonDialout:  ari.ChannelCauseNoAnswer,       // The call reached dialing timeout before it was answered. This timeout is fired by our time out(outgoing call).
		HangupReasonAMD:      ari.ChannelCauseCallAMD,        // the call's amd action result hung up the call.
	}

	cause, ok := mapCause[reason]
	if !ok {
		return ari.ChannelCauseNormalClearing
	}

	return cause
}
