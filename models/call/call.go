package call

import (
	"github.com/gofrs/uuid"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
)

// Call struct represent asterisk's channel information for client show
type Call struct {
	ID     uuid.UUID `json:"id"`      // Call's ID.
	UserID uint64    `json:"user_id"` // Call owner's ID.
	FlowID uuid.UUID `json:"flow_id"` // Attached flow id
	ConfID uuid.UUID `json:"conf_id"` // Currently joined conference id.
	Type   Type      `json:"type"`    // Call's type.

	MasterCallID   uuid.UUID   `json:"master_call_id"`   // Master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // Chained call ids
	RecordingID    uuid.UUID   `json:"recording_id"`     // Recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids"`    // Recording ids

	Source      address.Address `json:"source"`      // Source info
	Destination address.Address `json:"destination"` // Destination info

	Status       Status       `json:"status"`        // Call's status.
	Direction    Direction    `json:"direction"`     // Call's direction.
	HangupBy     HangupBy     `json:"hangup_by"`     // Describe which endpoint sent the hangup request first.
	HangupReason HangupReason `json:"hangup_reason"` // Desribe detail of hangup reason.
	WebhookURI   string       `json:"webhook_uri"`   // Webhook destination uri

	TMCreate string `json:"tm_create"` // Timestamp. Created time.
	TMUpdate string `json:"tm_update"` // Timestamp. Updated time.

	TMProgressing string `json:"tm_progressing"` // Timestamp. Progressing time.
	TMRinging     string `json:"tm_ringing"`     // Timestamp. Ringing time.
	TMHangup      string `json:"tm_hangup"`      // Timestamp. Hangup time.
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

// Direction type
type Direction string

// List of Direction
const (
	DirectionIncoming Direction = "incoming" // Call comming from outside of the voipbin.
	DirectionOutgoing Direction = "outgoing" // Call is generating from the voipbin.
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

// ConvertCall returns call.Call from cmall.Call
func ConvertCall(h *cmcall.Call) *Call {
	c := &Call{
		ID:     h.ID,
		UserID: h.UserID,
		FlowID: h.FlowID,
		ConfID: h.ConfID,
		Type:   Type(h.Type),

		MasterCallID:   h.MasterCallID,
		ChainedCallIDs: h.ChainedCallIDs,
		RecordingID:    h.RecordingID,
		RecordingIDs:   h.RecordingIDs,

		Source: address.Address{
			Type:   address.Type(h.Source.Type),
			Name:   h.Source.Name,
			Target: h.Source.Target,
		},
		Destination: address.Address{
			Type:   address.Type(h.Destination.Type),
			Name:   h.Destination.Name,
			Target: h.Destination.Target,
		},

		Status: Status(h.Status),

		Direction:    Direction(h.Direction),
		HangupBy:     HangupBy(h.HangupBy),
		HangupReason: HangupReason(h.HangupReason),
		WebhookURI:   h.WebhookURI,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,

		TMProgressing: h.TMProgressing,
		TMRinging:     h.TMRinging,
		TMHangup:      h.TMHangup,
	}

	return c
}
