package conversation

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	AccountID uuid.UUID `json:"account_id,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	ReferenceType Type   `json:"reference_type,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`

	Self *commonaddress.Address `json:"self,omitempty"`
	Peer *commonaddress.Address `json:"peer,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Conversation) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,
		Owner:    h.Owner,

		AccountID: h.AccountID,

		Name:   h.Name,
		Detail: h.Detail,

		ReferenceType: h.Type,
		ReferenceID:   h.ReferenceID,

		Self: h.Self,
		Peer: h.Peer,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Conversation) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
