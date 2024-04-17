package activeflow

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	FlowID uuid.UUID `json:"flow_id"`
	Status Status    `json:"status"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	CurrentAction action.Action `json:"current_action"`

	ForwardActionID uuid.UUID `json:"forward_action_id"`

	ExecutedActions []action.Action `json:"executed_actions"` // list of executed actions

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Activeflow) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		FlowID: h.FlowID,
		Status: h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		CurrentAction: h.CurrentAction,

		ForwardActionID: h.ForwardActionID,

		ExecutedActions: h.ExecutedActions,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Activeflow) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
