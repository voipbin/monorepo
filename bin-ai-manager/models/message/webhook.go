package message

import (
	"encoding/json"
	"time"

	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type WebhookMessage struct {
	identity.Identity

	AIcallID uuid.UUID `json:"aicall_id,omitempty"`

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

		AIcallID: h.AIcallID,

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
