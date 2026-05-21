package message

import (
	"encoding/json"
	"time"

	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type WebhookMessage struct {
	identity.Identity

	AIcallID     uuid.UUID `json:"aicall_id,omitempty"`
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty"`

	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Direction Direction `json:"direction"`

	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
}

func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AIcallID:     h.AIcallID,
		ActiveflowID: h.ActiveflowID,
		ActiveAIID:   h.ActiveAIID,

		Role:      h.Role,
		Content:   h.Content,
		Direction: h.Direction,

		ToolCalls:  h.ToolCalls,
		ToolCallID: h.ToolCallID,

		TMCreate: h.TMCreate,
	}
}

func (h *Message) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// IntermediateWebhookMessage is the external-facing struct for aimessage_intermediate events.
// It carries delta text content (not full accumulated text) and a sequence number for ordering.
type IntermediateWebhookMessage struct {
	identity.Identity

	AIcallID     uuid.UUID `json:"aicall_id,omitempty"`
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty"`

	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Direction Direction `json:"direction"`

	Sequence int `json:"sequence"`
}

func (h *IntermediateWebhookMessage) CreateWebhookEvent() ([]byte, error) {
	return json.Marshal(h)
}
