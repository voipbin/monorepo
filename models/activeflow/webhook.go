package activeflow

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	FlowID     uuid.UUID `json:"flow_id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	CurrentAction   action.Action `json:"current_action"`
	ForwardActionID uuid.UUID     `json:"forward_action_id"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *ActiveFlow) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		FlowID: h.FlowID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		CurrentAction:   h.CurrentAction,
		ForwardActionID: h.ForwardActionID,

		Actions: h.Actions,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *ActiveFlow) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
