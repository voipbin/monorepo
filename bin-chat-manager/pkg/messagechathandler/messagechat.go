package messagechathandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/dbhandler"
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

// List returns the chats by the given chatroom id.
func (h *messagechatHandler) List(ctx context.Context, token string, limit uint64, filters map[messagechat.Field]any) ([]*messagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"token":   token,
		"limit":   limit,
		"filters": filters,
	})

	// get
	res, err := h.db.MessagechatList(ctx, token, limit, filters)
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
	chatroomFilters := map[chatroom.Field]any{
		chatroom.FieldChatID:  res.ChatID,
		chatroom.FieldDeleted: false,
	}
	curTime := h.utilHandler.TimeGetCurTime()
	chatrooms, err := h.chatroomHandler.List(ctx, curTime, 100000, chatroomFilters)
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
			cr.OwnerID,
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
	curTime := h.utilHandler.TimeGetCurTime()
	tmp := &messagechat.Messagechat{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ChatID:   chatID,
		Source:   source,
		Type:     messageType,
		Text:     text,
		Medias:   medias,
		TMCreate: curTime,
		TMUpdate: curTime,
		TMDelete: dbhandler.DefaultTimeStamp,
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
	filters := map[messagechatroom.Field]any{
		messagechatroom.FieldDeleted:       false,
		messagechatroom.FieldMessagechatID: id,
	}
	messagechatrooms, err := h.messagechatroomHandler.List(ctx, dbhandler.DefaultTimeStamp, 100000, filters)
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
