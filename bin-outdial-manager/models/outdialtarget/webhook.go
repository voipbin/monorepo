package outdialtarget

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID        uuid.UUID `json:"id"`
	OutdialID uuid.UUID `json:"outdial_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Data   string `json:"data"`
	Status Status `json:"status"`

	// destinations
	Destination0 *commonaddress.Address `json:"destination_0"` // destination address 0
	Destination1 *commonaddress.Address `json:"destination_1"` // destination address 1
	Destination2 *commonaddress.Address `json:"destination_2"` // destination address 2
	Destination3 *commonaddress.Address `json:"destination_3"` // destination address 3
	Destination4 *commonaddress.Address `json:"destination_4"` // destination address 4

	// try counts
	TryCount0 int `json:"try_count_0"` // try count for destination 0
	TryCount1 int `json:"try_count_1"` // try count for destination 1
	TryCount2 int `json:"try_count_2"` // try count for destination 2
	TryCount3 int `json:"try_count_3"` // try count for destination 3
	TryCount4 int `json:"try_count_4"` // try count for destination 4

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *OutdialTarget) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:        h.ID,
		OutdialID: h.OutdialID,

		Name:   h.Name,
		Detail: h.Detail,

		Data:   h.Data,
		Status: h.Status,

		Destination0: h.Destination0,
		Destination1: h.Destination1,
		Destination2: h.Destination2,
		Destination3: h.Destination3,
		Destination4: h.Destination4,

		TryCount0: h.TryCount0,
		TryCount1: h.TryCount1,
		TryCount2: h.TryCount2,
		TryCount3: h.TryCount3,
		TryCount4: h.TryCount4,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *OutdialTarget) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
