package transfer

import (
	"encoding/json"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines conference webhook event
type WebhookMessage struct {
	commonidentity.Identity

	Type Type `json:"type"`

	// transferer/transferee info
	TransfererCallID    uuid.UUID               `json:"transferer_call_id"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses"`
	TransfereeCallID    uuid.UUID               `json:"transferee_call_id"`

	GroupcallID  uuid.UUID `json:"groupcall_id"` // created groupcall id
	ConfbridgeID uuid.UUID `json:"confbridge_id"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Transfer) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

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
