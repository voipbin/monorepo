package conference

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	uuid "github.com/gofrs/uuid"
)

// WebhookMessage defines conference webhook event
type WebhookMessage struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty"`

	Status Status `json:"status,omitempty"`

	Name    string         `json:"name,omitempty"`
	Detail  string         `json:"detail,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Timeout int            `json:"timeout,omitempty"` // timeout. second

	PreFlowID  uuid.UUID `json:"pre_flow_id,omitempty"`  // pre flow id
	PostFlowID uuid.UUID `json:"post_flow_id,omitempty"` // post flow id

	ConferencecallIDs []uuid.UUID `json:"conferencecall_ids,omitempty"`

	RecordingID  uuid.UUID   `json:"recording_id,omitempty"`
	RecordingIDs []uuid.UUID `json:"recording_ids,omitempty"`

	TranscribeID  uuid.UUID   `json:"transcribe_id,omitempty"`
	TranscribeIDs []uuid.UUID `json:"transcribe_ids,omitempty"`

	TMEnd *time.Time `json:"tm_end"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Conference) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Type: h.Type,

		Status: h.Status,

		Name:    h.Name,
		Detail:  h.Detail,
		Data:    h.Data,
		Timeout: h.Timeout,

		PreFlowID:  h.PreFlowID,
		PostFlowID: h.PostFlowID,

		ConferencecallIDs: h.ConferencecallIDs,

		RecordingID:  h.RecordingID,
		RecordingIDs: h.RecordingIDs,

		TranscribeID:  h.TranscribeID,
		TranscribeIDs: h.TranscribeIDs,

		TMEnd: h.TMEnd,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}

}

// CreateWebhookEvent generate WebhookEvent
func (h *Conference) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
