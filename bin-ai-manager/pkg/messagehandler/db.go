package messagehandler

import (
	"context"
	stderrors "errors"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *messageHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	aicallID uuid.UUID,
	activeflowID uuid.UUID,
	direction message.Direction,
	role message.Role,
	content string,
	toolCalls []message.ToolCall,
	toolCallID string,
	opts ...CreateOption,
) (*message.Message, error) {
	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
	}

	tmpToolCalls := toolCalls
	if tmpToolCalls == nil {
		tmpToolCalls = []message.ToolCall{}
	}

	// Apply defaults first, then override with caller-supplied options.
	// DeliveryStatusDelivered matches the legacy semantics + DB column default,
	// so existing callers that pass no opts continue to work unchanged.
	p := createParams{
		pipecatcallID:  uuid.Nil,
		deliveryStatus: message.DeliveryStatusDelivered,
	}
	for _, opt := range opts {
		opt(&p)
	}

	m := &message.Message{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		AIcallID:     aicallID,
		ActiveflowID: activeflowID,
		ActiveAIID:   p.activeAIID,

		Direction:  direction,
		Role:       role,
		Content:    content,
		ToolCalls:  tmpToolCalls,
		ToolCallID: toolCallID,

		PipecatcallID:  p.pipecatcallID,
		DeliveryStatus: p.deliveryStatus,
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
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"MESSAGE_NOT_FOUND",
				"The message was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not get data")
	}

	return res, nil
}

func (h *messageHandler) List(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error) {
	res, err := h.db.MessageList(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}
