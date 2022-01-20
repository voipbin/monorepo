package call

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Call defines
// used only for the swag.
type Call struct {
	// identity
	ID           uuid.UUID   `json:"id"`
	FlowID       uuid.UUID   `json:"flow_id"`       // flow id
	ConfbridgeID uuid.UUID   `json:"confbridge_id"` // currently joined confbridge id.
	Type         cmcall.Type `json:"type"`          // call type

	// etc info
	MasterCallID   uuid.UUID   `json:"master_call_id"`   // master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // chained call ids
	RecordingID    uuid.UUID   `json:"recording_id"`     // recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids"`    // recording ids

	// source/destination
	Source      cmaddress.Address `json:"source"`
	Destination cmaddress.Address `json:"destination"`

	// info
	Status       cmcall.Status       `json:"status"`
	Action       fmaction.Action     `json:"action"`
	Direction    cmcall.Direction    `json:"direction"`
	HangupBy     cmcall.HangupBy     `json:"hangup_by"`
	HangupReason cmcall.HangupReason `json:"hangup_reason"`
	WebhookURI   string              `json:"webhook_uri"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`

	TMProgressing string `json:"tm_progressing"`
	TMRinging     string `json:"tm_ringing"`
	TMHangup      string `json:"tm_hangup"`
}
