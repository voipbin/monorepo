package call

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	// identity
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  OwnerType `json:"owner_type"`
	OwnerID    uuid.UUID `json:"owner_id"`

	FlowID       uuid.UUID `json:"flow_id"` // flow id
	ActiveflowID uuid.UUID `json:"activeflow_id"`
	Type         Type      `json:"type"` // call type

	// etc info
	MasterCallID   uuid.UUID   `json:"master_call_id"`   // master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids"` // chained call ids
	RecordingID    uuid.UUID   `json:"recording_id"`     // recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids"`    // recording ids
	GroupcallID    uuid.UUID   `json:"groupcall_id"`     // groupcall id, if this set, that means this call is part of given groupcall.

	// source/destination
	Source      commonaddress.Address `json:"source"`
	Destination commonaddress.Address `json:"destination"`

	// info
	Status        Status          `json:"status"`
	Action        fmaction.Action `json:"action"`
	Direction     Direction       `json:"direction"`
	MuteDirection MuteDirection   `json:"mute_direction"`

	HangupBy     HangupBy     `json:"hangup_by"`
	HangupReason HangupReason `json:"hangup_reason"`

	// timestamp
	TMProgressing string `json:"tm_progressing"`
	TMRinging     string `json:"tm_ringing"`
	TMHangup      string `json:"tm_hangup"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Call) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,
		OwnerType:  h.OwnerType,
		OwnerID:    h.OwnerID,

		FlowID:       h.FlowID,
		ActiveflowID: h.ActiveFlowID,
		Type:         h.Type,

		MasterCallID:   h.MasterCallID,
		ChainedCallIDs: h.ChainedCallIDs,
		RecordingID:    h.RecordingID,
		RecordingIDs:   h.RecordingIDs,

		Source:      h.Source,
		Destination: h.Destination,

		Status:        h.Status,
		Action:        h.Action,
		Direction:     h.Direction,
		MuteDirection: h.MuteDirection,

		HangupBy:     h.HangupBy,
		HangupReason: h.HangupReason,

		TMRinging:     h.TMRinging,
		TMProgressing: h.TMProgressing,
		TMHangup:      h.TMHangup,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
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
