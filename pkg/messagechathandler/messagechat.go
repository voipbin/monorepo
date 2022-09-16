package messagechathandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
)

// Get returns the messagechat
func (h *messagechatHandler) Get(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Get",
		"chat_id": id,
	})

	// get
	res, err := h.db.MessagechatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get messagechat info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByChatID returns the chats by the given chatroom id.
func (h *messagechatHandler) GetsByChatID(ctx context.Context, chatID uuid.UUID, token string, limit uint64) ([]*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByChatID",
		"customer_id": chatID,
	})

	// get
	res, err := h.db.MessagechatGetsByChatID(ctx, chatID, token, limit)
	if err != nil {
		log.Errorf("Could not get messagechat info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new messagechat
func (h *messagechatHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	chatID uuid.UUID,
	source *commonaddress.Address,
	messageType messagechat.Type,
	text string,
	medias []media.Media,
) (*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"chat_id":     chatID,
		"source":      source,
	})

	// create a message chat
	res, err := h.create(
		ctx,
		customerID,
		chatID,
		source,
		messageType,
		text,
		medias,
	)
	if err != nil {
		log.Errorf("Could not create a new messagechat. err: %v", err)
		return nil, err
	}
	log = log.WithField("messagechat_id", res.ID)
	log.WithField("messagechat", res).Debugf("Created a new messagechat. messagechat_id: %s", res.ID)

	// get chatrooms
	curTime := dbhandler.GetCurTime()
	chatrooms, err := h.chatroomHandler.GetsByChatID(ctx, res.ChatID, curTime, 100000)
	if err != nil {
		log.Errorf("Could not get list of chatrooms. err: %v", err)
		return nil, err
	}
	convertType := messagechatroom.ConvertType(messageType)

	// send a messagechatroom to the chatrooms
	for _, cr := range chatrooms {
		tmp, err := h.messagechatroomHandler.Create(
			ctx,
			res.CustomerID,
			cr.ID,
			res.ID,
			res.Source,
			convertType,
			res.Text,
			res.Medias,
		)
		if err != nil {
			log.Errorf("Could not create messagechatroom. chatroom_id: %s, err: %v", cr.ID, err)
			continue
		}
		log.WithField("messagechatroom", tmp).Debugf("Created a new messagechatroom. chatroom_id: %s, messagechatroom_id: %s", tmp.ChatroomID, tmp.ID)
	}

	return res, nil
}

// Create creates a new messagechat
func (h *messagechatHandler) create(
	ctx context.Context,
	customerID uuid.UUID,
	chatID uuid.UUID,
	source *commonaddress.Address,
	messageType messagechat.Type,
	text string,
	medias []media.Media,
) (*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "create",
		"customer_id":  customerID,
		"message_type": messageType,
	})

	id := uuid.Must(uuid.NewV4())
	curTime := dbhandler.GetCurTime()
	tmp := &messagechat.Messagechat{
		ID:         id,
		CustomerID: customerID,
		ChatID:     id,
		Source:     &address.Address{},
		Type:       messageType,
		Text:       text,
		Medias:     medias,
		TMCreate:   curTime,
		TMUpdate:   curTime,
		TMDelete:   dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.MessagechatCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new messagechat correctly. err: %v", errCreate)
		return nil, errCreate
	}

	// get
	res, err := h.db.MessagechatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a created messagechat. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, messagechat.EventTypeMessagechatCreated, res)

	return res, nil
}

// Delete deletes the messagechat
func (h *messagechatHandler) Delete(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Delete",
		"messagechat_id": id,
	})

	res, err := h.delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the messagechat. err: %v", err)
		return nil, err
	}

	// get messagechatrooms
	messagechatrooms, err := h.messagechatroomHandler.GetsByMessagechatID(ctx, id, dbhandler.DefaultTimeStamp, 100000)
	if err != nil {
		log.Errorf("Could not get messagechatrooms. err: %v", err)
		return nil, err
	}

	// delete each messagechatroom
	for _, mc := range messagechatrooms {
		tmp, err := h.messagechatroomHandler.Delete(ctx, mc.ID)
		if err != nil {
			log.Errorf("Could not delete messagechatroom. err: %v", err)
			continue
		}
		log.WithField("messagechatroom", tmp).Debugf("Deleted messagechatroom. messagechat_id: %s, messagechatroom_id: %s", id, tmp.ID)
	}

	return res, nil
}

// Delete deletes the messagechat
func (h *messagechatHandler) delete(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "GetsByChatroomID",
		"messagechat_id": id,
	})

	// delete
	if errDel := h.db.MessagechatDelete(ctx, id); errDel != nil {
		log.Errorf("Could not delete messagechat info. err: %v", errDel)
		return nil, errDel
	}

	// get deleted
	res, err := h.db.MessagechatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a deleted messagechat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, messagechat.EventTypeMessagechatDeleted, res)

	return res, nil
}
