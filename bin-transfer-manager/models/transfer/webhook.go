package transfer

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines conference webhook event
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Type Type `json:"type"`

	// transferer/transferee info
	TransfererCallID    uuid.UUID               `json:"transferer_call_id"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses"`
	TransfereeCallID    uuid.UUID               `json:"transferee_call_id"`

	GroupcallID  uuid.UUID `json:"groupcall_id"` // created groupcall id
	ConfbridgeID uuid.UUID `json:"confbridge_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Transfer) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Type: h.Type,

		TransfererCallID:    h.TransfererCallID,
		TransfereeAddresses: h.TransfereeAddresses,
		TransfereeCallID:    h.TransfereeCallID,

		GroupcallID:  h.GroupcallID,
		ConfbridgeID: h.ConfbridgeID,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}

}

// CreateWebhookEvent generate WebhookEvent
func (h *Transfer) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
