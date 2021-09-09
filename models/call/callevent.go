package call

import (
	"encoding/json"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// WebhookEventData defines
type WebhookEventData struct {
	// identity
	ID     uuid.UUID `json:"id"`
	FlowID uuid.UUID `json:"flow_id"` // flow id
	ConfID uuid.UUID `json:"conf_id"` // currently joined conference id
	Type   string    `json:"type"`    // call type

	// etc info
	MasterCallID   uuid.UUID   `json:"master_call_id"`   // master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // chained call ids
	RecordingID    uuid.UUID   `json:"recording_id"`     // recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids"`    // recording ids

	// source/destination
	Source      address.Address `json:"source"`
	Destination address.Address `json:"destination"`

	// info
	Status       string        `json:"status"`
	Action       action.Action `json:"action"`
	Direction    string        `json:"direction"`
	HangupBy     string        `json:"hangup_by"`
	HangupReason string        `json:"hangup_reason"`
	WebhookURI   string        `json:"webhook_uri"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`

	TMProgressing string `json:"tm_progressing"`
	TMRinging     string `json:"tm_ringing"`
	TMHangup      string `json:"tm_hangup"`
}

// CreateWebhookEvent generate WebhookEvent
func (h *Call) CreateWebhookEvent(t string) ([]byte, error) {
	e := &WebhookEventData{
		ID:     h.ID,
		FlowID: h.FlowID,
		ConfID: h.ConfID,
		Type:   string(h.Type),

		MasterCallID:   h.MasterCallID,
		ChainedCallIDs: h.ChainedCallIDs,
		RecordingID:    h.RecordingID,
		RecordingIDs:   h.RecordingIDs,

		Source:      h.Source,
		Destination: h.Destination,
		Status:      string(h.Status),
		Action:      h.Action,
		Direction:   string(h.Direction),

		HangupBy:      string(h.HangupBy),
		HangupReason:  string(h.HangupReason),
		WebhookURI:    h.WebhookURI,
		TMCreate:      h.TMCreate,
		TMUpdate:      h.TMUpdate,
		TMProgressing: h.TMProgressing,
		TMRinging:     h.TMRinging,
		TMHangup:      h.TMHangup,
	}

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
