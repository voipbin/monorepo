package queuecallreference

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID   uuid.UUID               `json:"id"`   // reference's id
	Type queuecall.ReferenceType `json:"type"` // reference's type

	CurrentQueuecallID uuid.UUID   `json:"current_queuecall_id"`
	QueuecallIDs       []uuid.UUID `json:"queuecall_ids"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage convert to Event
func (h *QueuecallReference) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:   h.ID,
		Type: h.Type,

		CurrentQueuecallID: h.CurrentQueuecallID,
		QueuecallIDs:       h.QueuecallIDs,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}
