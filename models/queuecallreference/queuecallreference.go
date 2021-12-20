package queuecallreference

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// QueuecallReference defines
type QueuecallReference struct {
	ID     uuid.UUID               `json:"id"` // reference's id
	UserID uint64                  `json:"user_id"`
	Type   queuecall.ReferenceType `json:"type"` // reference's type

	CurrentQueuecallID uuid.UUID   `json:"current_queuecall_id"`
	QueuecallIDs       []uuid.UUID `json:"queuecall_ids"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}
