package call

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	FlowID       uuid.UUID `json:"flow_id,omitempty"` // flow id
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	Type         Type      `json:"type,omitempty"` // call type

	// etc info
	MasterCallID   uuid.UUID   `json:"master_call_id,omitempty"`   // master call id
	ChainedCallIDs []uuid.UUID `json:"chained_call_ids,omitempty"` // chained call ids
	RecordingID    uuid.UUID   `json:"recording_id,omitempty"`     // recording id(current)
	RecordingIDs   []uuid.UUID `json:"recording_ids,omitempty"`    // recording ids
	GroupcallID    uuid.UUID   `json:"groupcall_id,omitempty"`     // groupcall id, if this set, that means this call is part of given groupcall.

	// source/destination
	Source      commonaddress.Address `json:"source,omitempty"`
	Destination commonaddress.Address `json:"destination,omitempty"`

	// info
	Status        Status          `json:"status,omitempty"`
	Action        fmaction.Action `json:"action,omitempty"`
	Direction     Direction       `json:"direction,omitempty"`
	MuteDirection MuteDirection   `json:"mute_direction,omitempty"`

	HangupBy     HangupBy     `json:"hangup_by,omitempty"`
	HangupReason HangupReason `json:"hangup_reason,omitempty"`

	// timestamp
	TMProgressing *time.Time `json:"tm_progressing,omitempty"`
	TMRinging     *time.Time `json:"tm_ringing,omitempty"`
	TMHangup      *time.Time `json:"tm_hangup,omitempty"`

	TMCreate *time.Time `json:"tm_create,omitempty"`
	TMUpdate *time.Time `json:"tm_update,omitempty"`
	TMDelete *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Call) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,
		Owner:    h.Owner,

		FlowID:       h.FlowID,
		ActiveflowID: h.ActiveflowID,
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
