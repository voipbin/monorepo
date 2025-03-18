package messagehandler

import (
	"context"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *messageHandler) Create(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, direction message.Direction, role message.Role, content string) (*message.Message, error) {
	id := h.utilHandler.UUIDCreate()

	m := &message.Message{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		AIcallID: aicallID,

		Direction: direction,
		Role:      role,
		Content:   content,
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

func (h *messageHandler) Gets(ctx context.Context, aicallID uuid.UUID, size uint64, token string, filters map[string]string) ([]*message.Message, error) {
	res, err := h.db.MessageGets(ctx, aicallID, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}
