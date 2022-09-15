package messagechatroomhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
)

// Get returns the messagechatroom
func (h *messagechatroomHandler) Get(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "Get",
		"messagechatroom_id": id,
	})

	// get
	res, err := h.db.MessagechatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get messagechat info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByChatroomID returns the messagechatrooms by the given chatroom id.
func (h *messagechatroomHandler) GetsByChatroomID(ctx context.Context, chatroomID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByChatroomID",
		"chatroom_id": chatroomID,
	})

	// get
	res, err := h.db.MessagechatroomGetsByChatroomID(ctx, chatroomID, token, limit)
	if err != nil {
		log.Errorf("Could not get messagechatroom info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByMessagechatID returns the list of messagechatrooms by the given messagechat id.
func (h *messagechatroomHandler) GetsByMessagechatID(ctx context.Context, messagechatID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByMessagechatID",
		"chatroom_id": messagechatID,
	})

	// get
	res, err := h.db.MessagechatroomGetsByMessagechatID(ctx, messagechatID, token, limit)
	if err != nil {
		log.Errorf("Could not get messagechatroom info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new messagechatroom
func (h *messagechatroomHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	chatroomID uuid.UUID,
	messagechatID uuid.UUID,
	source *commonaddress.Address,
	messageType message.Type,
	text string,
	medias []media.Media,
) (*messagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Create",
		"customer_id":  customerID,
		"message_type": messageType,
	})

	id := uuid.Must(uuid.NewV4())
	curTime := dbhandler.GetCurTime()
	tmp := &messagechatroom.Messagechatroom{
		ID:         id,
		CustomerID: customerID,

		ChatroomID:    chatroomID,
		MessagechatID: messagechatID,
		Message: message.Message{
			Source: source,
			Type:   messageType,
			Text:   text,
			Medias: medias,
		},

		TMCreate: curTime,
		TMUpdate: curTime,
		TMDelete: dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.MessagechatroomCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new messagechat correctly. err: %v", errCreate)
		return nil, errCreate
	}

	// get
	res, err := h.db.MessagechatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a created messagechat. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, messagechatroom.EventTypeMessagechatroomCreated, res)

	return res, nil
}

// Delete deletes the messagechatroom
func (h *messagechatroomHandler) Delete(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "Delete",
		"messagechatroom_id": id,
	})

	// delete
	if errDel := h.db.MessagechatroomDelete(ctx, id); errDel != nil {
		log.Errorf("Could not delete messagechatroom info. err: %v", errDel)
		return nil, errDel
	}

	// get deleted
	res, err := h.db.MessagechatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a deleted messagechatroom info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, messagechatroom.EventTypeMessagechatroomDeleted, res)

	return res, nil
}
