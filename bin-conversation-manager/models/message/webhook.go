package message

import (
	"encoding/json"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	ConversationID uuid.UUID `json:"conversation_id,omitempty"`
	Direction      Direction `json:"direction,omitempty"`
	Status         Status    `json:"status,omitempty"`

	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	// Absolute endpoints: source = sending party, destination = receiving party.
	// A consumer recovers the remote party by direction: inbound (incoming) =>
	// remote = source; outbound (outgoing) => remote = destination.
	Source      commonaddress.Address `json:"source,omitempty"`
	Destination commonaddress.Address `json:"destination,omitempty"`

	Text   string        `json:"text,omitempty"`
	Medias []media.Media `json:"medias,omitempty"`

	// CaseID decodes the case-linking hint carried on the internal
	// conversation_message_created event (contact-case-management
	// design §4.3). This struct doubles as the decode target for
	// bin-contact-manager's subscribehandler AND the shape
	// ConvertWebhookMessage below builds for the customer-facing
	// webhook -- ConvertWebhookMessage deliberately leaves this unset;
	// see its comment and Test_ConvertWebhookMessage_NeverCopiesCaseID.
	CaseID *uuid.UUID `json:"case_id,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ConversationID: h.ConversationID,
		Direction:      h.Direction,
		Status:         h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Source:      h.Source,
		Destination: h.Destination,

		Text:   h.Text,
		Medias: h.Medias,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Message) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
