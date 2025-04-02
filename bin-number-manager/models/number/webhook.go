package number

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Number string `json:"number"`

	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status Status `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	TMPurchase string `json:"tm_purchase"`
	TMRenew    string `json:"tm_renew"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Number) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Number: h.Number,

		CallFlowID:    h.CallFlowID,
		MessageFlowID: h.MessageFlowID,

		Name:   h.Name,
		Detail: h.Detail,

		Status: h.Status,

		T38Enabled:       h.T38Enabled,
		EmergencyEnabled: h.EmergencyEnabled,

		TMPurchase: h.TMPurchase,
		TMRenew:    h.TMRenew,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Number) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
