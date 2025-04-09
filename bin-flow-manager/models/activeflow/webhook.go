package activeflow

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	FlowID uuid.UUID `json:"flow_id,omitempty"`
	Status Status    `json:"status,omitempty"`

	ReferenceType         ReferenceType `json:"reference_type,omitempty"`
	ReferenceID           uuid.UUID     `json:"reference_id,omitempty"`
	ReferenceActiveflowID uuid.UUID     `json:"reference_activeflow_id,omitempty"`

	CurrentAction action.Action `json:"current_action,omitempty"`

	ForwardActionID uuid.UUID `json:"forward_action_id,omitempty"`

	ExecutedActions []action.Action `json:"executed_actions,omitempty"` // list of executed actions

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Activeflow) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		FlowID: h.FlowID,
		Status: h.Status,

		ReferenceType:         h.ReferenceType,
		ReferenceID:           h.ReferenceID,
		ReferenceActiveflowID: h.ReferenceActiveflowID,

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
