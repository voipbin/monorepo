package providercall

import (
	"encoding/json"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a ProviderCall.
// All fields are safe to expose — ProviderCall has no infrastructure-only
// or internal-only fields today.
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	// Requested
	CustomerID   uuid.UUID               `json:"customer_id"`
	ProviderID   uuid.UUID               `json:"provider_id"`
	FlowID       uuid.UUID               `json:"flow_id"`
	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Anonymous    string                  `json:"anonymous"`

	// Created
	CallIDs      []uuid.UUID `json:"call_ids"`
	GroupcallIDs []uuid.UUID `json:"groupcall_ids"`

	// Timestamps
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the external-facing webhook shape.
func (h *ProviderCall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		CustomerID:   h.CustomerID,
		ProviderID:   h.ProviderID,
		FlowID:       h.FlowID,
		Source:       h.Source,
		Destinations: h.Destinations,
		Anonymous:    h.Anonymous,

		CallIDs:      h.CallIDs,
		GroupcallIDs: h.GroupcallIDs,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent.
func (h *ProviderCall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
