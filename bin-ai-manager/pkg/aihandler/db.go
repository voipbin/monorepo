package aihandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"
)

// Create creates a new ai record.
func (h *aiHandler) dbCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType ai.EngineType,
	engineModel ai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*ai.AI, error) {
	id := h.utilHandler.UUIDCreate()
	c := &ai.AI{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Name:   name,
		Detail: detail,

		EngineType:  engineType,
		EngineModel: engineModel,
		EngineData:  engineData,

		InitPrompt: initPrompt,
	}

	if err := h.db.AICreate(ctx, c); err != nil {
		return nil, errors.Wrapf(err, "could not create ai")
	}

	res, err := h.db.AIGet(ctx, c.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created ai")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeCreated, res)

	return res, nil
}

// Get returns ai.
func (h *aiHandler) Get(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai")
	}

	return res, nil
}

// Gets returns list of ais.
func (h *aiHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*ai.AI, error) {
	res, err := h.db.AIGets(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ais")
	}

	return res, nil
}

// Delete deletes the ai.
func (h *aiHandler) Delete(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	if err := h.db.AIDelete(ctx, id); err != nil {
		return nil, errors.Wrapf(err, "could not delete ai")
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted ai")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeDeleted, res)

	return res, nil
}

// Update updates the ai info
func (h *aiHandler) dbUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineType ai.EngineType,
	engineModel ai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*ai.AI, error) {
	if err := h.db.AISetInfo(ctx, id, name, detail, engineType, engineModel, engineData, initPrompt); err != nil {
		return nil, errors.Wrapf(err, "could not update ai")
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated ai")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)

	return res, nil
}
