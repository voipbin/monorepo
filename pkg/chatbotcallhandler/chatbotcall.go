package chatbotcallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
)

// Create is creating a new chatbotcall.
// it increases corresponded counter
func (h *chatbotcallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	chatbotID uuid.UUID,
	referenceType chatbotcall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	gender chatbotcall.Gender,
	language string,
) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "Create",
			"customer_id":    customerID,
			"chatbotcall_id": chatbotID,
		},
	)

	id := h.utilHandler.CreateUUID()
	tmp := &chatbotcall.Chatbotcall{
		ID:         id,
		CustomerID: customerID,
		ChatbotID:  chatbotID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		ConfbridgeID: confbridgeID,

		Gender:   gender,
		Language: language,

		Status: chatbotcall.StatusInitiating,
	}
	log = log.WithField("chatbotcall_id", id.String())
	log.WithField("chatbotcall", tmp).Debugf("Creating chatbotcall. chatbotcall_id: %s", tmp.ID)

	if errCreate := h.db.ChatbotcallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new chatbotcall. err: %v", errCreate)
		return nil, errCreate
	}
	promChatbotcallCreateTotal.WithLabelValues(string(tmp.ReferenceType)).Inc()

	res, err := h.db.ChatbotcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created chatbotcall info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, chatbotcall.EventTypeChatbotcallInitializing, res)

	// todo: start health check

	return res, nil
}

// Get is handy function for getting a chatbotcall.
func (h *chatbotcallHandler) Get(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "Get",
			"chatbotcall_id": id,
		},
	)

	res, err := h.db.ChatbotcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatbotcall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID returns a chatbotcall by the reference_id.
func (h *chatbotcallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "GetByReferenceID",
			"reference_id": referenceID,
		},
	)

	res, err := h.db.ChatbotcallGetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get chatbotcall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByTranscribeID returns a chatbotcall by the transcribe_id.
func (h *chatbotcallHandler) GetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "GetByTranscribeID",
			"reference_id": transcribeID,
		},
	)

	res, err := h.db.ChatbotcallGetByTranscribeID(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get chatbotcall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateStatusStart updates the status to start
func (h *chatbotcallHandler) UpdateStatusStart(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "UpdateStatusStart",
			"chatbot_id": id,
		},
	)

	if errUpdate := h.db.ChatbotcallUpdateStatusProgressing(ctx, id, transcribeID); errUpdate != nil {
		log.Errorf("Could not get chatbotcall info. err: %v", errUpdate)
		return nil, errUpdate
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chatbotcall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateStatusEnd updates the status to end
func (h *chatbotcallHandler) UpdateStatusEnd(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "UpdateStatusEnd",
			"chatbot_id": id,
		},
	)

	if errUpdate := h.db.ChatbotcallUpdateStatusEnd(ctx, id); errUpdate != nil {
		log.Errorf("Could not get chatbotcall info. err: %v", errUpdate)
		return nil, errUpdate
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chatbotcall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of chatbotcalls.
func (h *chatbotcallHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
	})

	res, err := h.db.ChatbotcallGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get chatbotcalls. err: %v", err)
		return nil, err
	}

	return res, nil
}
