package transcribe

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty"`

	ReferenceType ReferenceType `json:"reference_type"` // reference's type
	ReferenceID   uuid.UUID     `json:"reference_id"`   // call/conference/recording's id

	Status    Status    `json:"status"`
	Language  string    `json:"language"` // BCP47 type's language code. en-US
	Direction Direction `json:"direction"`

	// timestamp
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Transcribe) ConvertWebhookMessage() *WebhookMessage {

	return &WebhookMessage{
		Identity: h.Identity,

		ActiveflowID: h.ActiveflowID,
		OnEndFlowID:  h.OnEndFlowID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Status:    h.Status,
		Language:  h.Language,
		Direction: h.Direction,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Transcribe) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
