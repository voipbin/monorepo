package conference

import (
	"encoding/json"

	uuid "github.com/gofrs/uuid"
)

// WebhookEventData defines conference webhook event
type WebhookEventData struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`

	Status string `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

	CallIDs []uuid.UUID `json:"call_ids"` // list of call ids of conference

	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"`

	WebhookURI string `json:"webhook_uri"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// CreateWebhookEvent generate WebhookEvent
func (h *Conference) CreateWebhookEvent(t string) ([]byte, error) {
	e := &WebhookEventData{
		ID:   h.ID,
		Type: string(h.Type),

		Status: string(h.Status),

		Name:    h.Name,
		Detail:  h.Detail,
		Data:    h.Data,
		Timeout: h.Timeout,

		CallIDs: h.CallIDs,

		RecordingID:  h.RecordingID,
		RecordingIDs: h.RecordingIDs,

		WebhookURI: h.WebhookURI,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
