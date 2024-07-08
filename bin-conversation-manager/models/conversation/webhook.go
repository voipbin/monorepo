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

	AccountID uuid.UUID `json:"account_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   string        `json:"reference_id"`

	Source       *commonaddress.Address  `json:"source"`
	Participants []commonaddress.Address `json:"participants"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Conversation) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,
		Owner:    h.Owner,

		AccountID: h.AccountID,

		Name:   h.Name,
		Detail: h.Detail,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Source:       h.Source,
		Participants: h.Participants,

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
