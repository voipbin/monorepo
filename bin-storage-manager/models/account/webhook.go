package account

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	TotalFileCount int64 `json:"total_file_count"` // total file count
	TotalFileSize  int64 `json:"total_file_size"`  // total file size in bytes

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Account) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		TotalFileCount: h.TotalFileCount,
		TotalFileSize:  h.TotalFileSize,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Account) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
