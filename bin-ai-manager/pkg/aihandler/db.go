package aihandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

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
	log := logrus.WithFields(logrus.Fields{
		"func":         "Create",
		"customer_id":  customerID,
		"name":         name,
		"detail":       detail,
		"engine_type":  engineType,
		"engine_model": engineModel,
		"data":         engineData,
	})

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
	log.WithField("ai", c).Debugf("Creating a new ai. ai_id: %s", c.ID)

	if err := h.db.AICreate(ctx, c); err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	res, err := h.db.AIGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get a created call. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeCreated, res)

	return res, nil
}

// Get returns ai.
func (h *aiHandler) Get(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Get",
		"ai_id": id,
	})

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of ais.
func (h *aiHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*ai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
		"filters":     filters,
	})

	res, err := h.db.AIGets(ctx, customerID, size, token, filters)
	if err != nil {
		log.Errorf("Could not get ais. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the ai.
func (h *aiHandler) Delete(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Delete",
		"ai_id": id,
	})

	if err := h.db.AIDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the ai. err: %v", err)
		return nil, err
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated ai. err: %v", err)
		return nil, err
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
	log := logrus.WithFields(logrus.Fields{
		"func":         "Update",
		"ai_id":        id,
		"name":         name,
		"detail":       detail,
		"engine_type":  engineType,
		"engine_model": engineModel,
		"engine_data":  engineData,
		"init_prompt":  initPrompt,
	})

	if err := h.db.AISetInfo(ctx, id, name, detail, engineType, engineModel, engineData, initPrompt); err != nil {
		log.Errorf("Could not update the ai. err: %v", err)
		return nil, err
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated ai. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)

	return res, nil
}
