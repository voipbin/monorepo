package messagehandler

import (
	"context"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *messageHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	aicallID uuid.UUID,
	direction message.Direction,
	role message.Role,
	content string,
	toolCalls []message.ToolCall,
	toolCallID string,
) (*message.Message, error) {
	id := h.utilHandler.UUIDCreate()

	tmpToolCalls := toolCalls
	if tmpToolCalls == nil {
		tmpToolCalls = []message.ToolCall{}
	}

	m := &message.Message{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		AIcallID: aicallID,

		Direction:  direction,
		Role:       role,
		Content:    content,
		ToolCalls:  tmpToolCalls,
		ToolCallID: toolCallID,
	}
	if err := h.db.MessageCreate(ctx, m); err != nil {
		return nil, err
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, err
	}

	promMessageCreateTotal.WithLabelValues(string(role)).Inc()

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)
	return res, nil
}

// Get returns ai.
func (h *messageHandler) Get(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data")
	}

	return res, nil
}

func (h *messageHandler) Gets(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error) {
	res, err := h.db.MessageGets(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}
