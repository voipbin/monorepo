package call

import (
	"encoding/json"

	uuid "github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// WebhookMessage defines
type WebhookMessage struct {
	// identity
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	FlowID     uuid.UUID `json:"flow_id"` // flow id
	Type       Type      `json:"type"`    // call type

	// etc info
	MasterCallID   uuid.UUID   `json:"master_call_id"`   // master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // chained call ids
	RecordingID    uuid.UUID   `json:"recording_id"`     // recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids"`    // recording ids

	// source/destination
	Source      commonaddress.Address `json:"source"`
	Destination commonaddress.Address `json:"destination"`

	// info
	Status       Status          `json:"status"`
	Action       fmaction.Action `json:"action"`
	Direction    Direction       `json:"direction"`
	HangupBy     HangupBy        `json:"hangup_by"`
	HangupReason HangupReason    `json:"hangup_reason"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`

	TMProgressing string `json:"tm_progressing"`
	TMRinging     string `json:"tm_ringing"`
	TMHangup      string `json:"tm_hangup"`
}

// ConvertWebhookMessage converts to the event
func (h *Call) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,
		FlowID:     h.FlowID,
		Type:       h.Type,

		MasterCallID:   h.MasterCallID,
		ChainedCallIDs: h.ChainedCallIDs,
		RecordingID:    h.RecordingID,
		RecordingIDs:   h.RecordingIDs,

		Source:      h.Source,
		Destination: h.Destination,
		Status:      h.Status,
		Action:      h.Action,
		Direction:   h.Direction,

		HangupBy:     h.HangupBy,
		HangupReason: h.HangupReason,

		TMCreate:      h.TMCreate,
		TMUpdate:      h.TMUpdate,
		TMProgressing: h.TMProgressing,
		TMRinging:     h.TMRinging,
		TMHangup:      h.TMHangup,
	}

}

// CreateWebhookEvent generate WebhookEvent
func (h *Call) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
