package outplan

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	// basic info
	Name   string `json:"name"`
	Detail string `json:"detail"`

	// source settings
	Source *commonaddress.Address `json:"source"` // caller id

	// plan dial settings
	DialTimeout int `json:"dial_timeout"` // milliseconds
	TryInterval int `json:"try_interval"` // milliseconds

	MaxTryCount0 int `json:"max_try_count_0"`
	MaxTryCount1 int `json:"max_try_count_1"`
	MaxTryCount2 int `json:"max_try_count_2"`
	MaxTryCount3 int `json:"max_try_count_3"`
	MaxTryCount4 int `json:"max_try_count_4"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Outplan) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Name:   h.Name,
		Detail: h.Detail,

		Source: h.Source,

		DialTimeout:  h.DialTimeout,
		TryInterval:  h.TryInterval,
		MaxTryCount0: h.MaxTryCount0,
		MaxTryCount1: h.MaxTryCount1,
		MaxTryCount2: h.MaxTryCount2,
		MaxTryCount3: h.MaxTryCount3,
		MaxTryCount4: h.MaxTryCount4,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Outplan) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
