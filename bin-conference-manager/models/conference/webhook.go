package conference

import (
	"encoding/json"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"
)

// WebhookMessage defines conference webhook event
type WebhookMessage struct {
	commonidentity.Identity

	Type Type `json:"type"`

	Status Status `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

	PreActions  []fmaction.Action `json:"pre_actions"`  // pre actions
	PostActions []fmaction.Action `json:"post_actions"` // post actions

	ConferencecallIDs []uuid.UUID `json:"conferencecall_ids"`

	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"`

	TranscribeID  uuid.UUID   `json:"transcribe_id"`
	TranscribeIDs []uuid.UUID `json:"transcribe_ids"`

	TMEnd string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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

		PreActions:  h.PreActions,
		PostActions: h.PostActions,

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
