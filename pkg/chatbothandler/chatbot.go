package chatbothandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
)

// Create creates a new chatbot record.
func (h *chatbotHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType chatbot.EngineType,
) (*chatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"engine_type": engineType,
	})

	id := h.utilHandler.CreateUUID()
	c := &chatbot.Chatbot{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		EngineType: engineType,
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

	return res, nil
}

// Get returns chatbot.
func (h *chatbotHandler) Get(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Get",
			"chatbot_id": id,
		},
	)

	res, err := h.db.ChatbotGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatbot. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of chatbots.
func (h *chatbotHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*chatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
	})

	res, err := h.db.ChatbotGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get chatbots. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the chatbot.
func (h *chatbotHandler) Delete(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Delete",
			"chatbot_id": id,
		},
	)

	if err := h.db.ChatbotDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the chatbot. err: %v", err)
		return nil, err
	}

	res, err := h.db.ChatbotGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated chatbot. err: %v", err)
		return nil, err
	}

	return res, nil
}
