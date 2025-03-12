package chatbothandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-common-handler/models/identity"
)

// Create creates a new chatbot record.
func (h *chatbotHandler) dbCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType chatbot.EngineType,
	engineModel chatbot.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*chatbot.Chatbot, error) {
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
	c := &chatbot.Chatbot{
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
	log.WithField("chatbot", c).Debugf("Creating a new chatbot. chatbot_id: %s", c.ID)

	if err := h.db.ChatbotCreate(ctx, c); err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	res, err := h.db.ChatbotGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get a created call. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatbot.EventTypeChatbotCreated, res)

	return res, nil
}

// Get returns chatbot.
func (h *chatbotHandler) Get(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"chatbot_id": id,
	})

	res, err := h.db.ChatbotGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatbot. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of chatbots.
func (h *chatbotHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
		"filters":     filters,
	})

	res, err := h.db.ChatbotGets(ctx, customerID, size, token, filters)
	if err != nil {
		log.Errorf("Could not get chatbots. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the chatbot.
func (h *chatbotHandler) Delete(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"chatbot_id": id,
	})

	if err := h.db.ChatbotDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the chatbot. err: %v", err)
		return nil, err
	}

	res, err := h.db.ChatbotGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated chatbot. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatbot.EventTypeChatbotDeleted, res)

	return res, nil
}

// Update updates the chatbot info
func (h *chatbotHandler) dbUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineType chatbot.EngineType,
	engineModel chatbot.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*chatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Update",
		"chatbot_id":   id,
		"name":         name,
		"detail":       detail,
		"engine_type":  engineType,
		"engine_model": engineModel,
		"engine_data":  engineData,
		"init_prompt":  initPrompt,
	})

	if err := h.db.ChatbotSetInfo(ctx, id, name, detail, engineType, engineModel, engineData, initPrompt); err != nil {
		log.Errorf("Could not update the chatbot. err: %v", err)
		return nil, err
	}

	res, err := h.db.ChatbotGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated chatbot. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatbot.EventTypeChatbotUpdated, res)

	return res, nil
}
