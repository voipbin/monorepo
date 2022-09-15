package chathandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
)

const (
	maxCountChatsTypeGroup = 100
)

// Get returns the chat
func (h *chatHandler) Get(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Get",
		"chat_id": id,
	})

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chat info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByCustomerID returns the chats by the given customer id.
func (h *chatHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByCustomerID",
		"customer_id": customerID,
	})

	// get
	res, err := h.db.ChatGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get chat info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new chat
func (h *chatHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	chatType chat.Type,
	ownerID uuid.UUID,
	participantIDs []uuid.UUID,
	name string,
	detail string,
) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"customer_id":     customerID,
		"type":            chatType,
		"owner_id":        ownerID,
		"participant_ids": participantIDs,
	})

	// sort the participants
	sortParticipantIDs(participantIDs)

	// validate request
	if chatType == chat.TypeNormal {
		// check the exist
		tmp, err := h.db.ChatGetByTypeAndParticipantsID(ctx, customerID, chatType, participantIDs)
		if err == nil && tmp != nil {
			log.WithField("chat", tmp).Debugf("The given chat is already exist. chat_id: %s", tmp.ID)
			return nil, fmt.Errorf("already exist")
		}
	} else {
		// check the max chat count
		curTime := dbhandler.GetCurTime()
		tmp, err := h.db.ChatGetsByType(ctx, customerID, chat.TypeGroup, curTime, maxCountChatsTypeGroup)
		if err != nil {
			log.Errorf("Could not get list of chats. err: %v", err)
			return nil, err
		}

		if len(tmp) >= maxCountChatsTypeGroup {
			log.Warnf("Exceeded max chat count. max_chat_count: %d", maxCountChatsTypeGroup)
			return nil, fmt.Errorf("exceeded max chat count")
		}
	}

	// create chat
	res, err := h.create(
		ctx,
		customerID,
		chatType,
		ownerID,
		participantIDs,
		name,
		detail,
	)
	if err != nil {
		log.Errorf("Could not create a chat. err: %v", err)
		return nil, err
	}
	log.WithField("chat", res).Debugf("Created a new chat. chat_id: %s", res.ID)

	// create chatrooms
	chatroomType := chatroom.ConvertType(chatType)
	for _, participantID := range participantIDs {
		tmp, err := h.chatroomHandler.Create(
			ctx,
			customerID,
			chatroomType,
			res.ID,
			participantID,
			participantIDs,
			name,
			detail,
		)
		if err != nil {
			log.Errorf("Could not create a chatroom. err: %v", err)
			continue
		}
		log.WithField("chatroom", tmp).Debugf("Created a new chatroom. chat_id: %s, chatroom_id: %s, owner_id: %s", res.ID, tmp.ID, participantID)
	}

	return res, nil
}

// Create creates a new chat
func (h *chatHandler) create(
	ctx context.Context,
	customerID uuid.UUID,
	chatType chat.Type,
	ownerID uuid.UUID,
	participantIDs []uuid.UUID,
	name string,
	detail string,
) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"customer_id":     customerID,
		"type":            chatType,
		"owner_id":        ownerID,
		"participant_ids": participantIDs,
	})

	id := uuid.Must(uuid.NewV4())
	curTime := dbhandler.GetCurTime()
	tmp := &chat.Chat{
		ID:             id,
		CustomerID:     customerID,
		Type:           chatType,
		OwnerID:        ownerID,
		ParticipantIDs: participantIDs,
		Name:           name,
		Detail:         detail,
		TMCreate:       curTime,
		TMUpdate:       curTime,
		TMDelete:       dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.ChatCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new chat correctly. err: %v", errCreate)
		return nil, errCreate
	}

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a created chat. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chat.EventTypeChatCreated, res)

	return res, nil
}

// UpdateBasicInfo updates the chat's basic info
func (h *chatHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateBasicInfo",
		"chat_id": id,
	})

	if errUpdate := h.db.ChatUpdateBasicInfo(ctx, id, name, detail); errUpdate != nil {
		log.Errorf("Could not update the chat's basic info. err: %v", errUpdate)
		return nil, errUpdate
	}

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chat.EventTypeChatUpdated, res)

	return res, nil
}

// UpdateOwnerID updates the chat's owner_id
func (h *chatHandler) UpdateOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateOwnerID",
		"chat_id":      id,
		"new_owner_id": ownerID,
	})

	if errUpdate := h.db.ChatUpdateOwnerID(ctx, id, ownerID); errUpdate != nil {
		log.Errorf("Could not update the chat. err: %v", errUpdate)
		return nil, errUpdate
	}

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chat.EventTypeChatUpdated, res)

	return res, nil
}

// AddParticipantID adds the given pariticipant_id to the given chat's pariticipant_ids
func (h *chatHandler) AddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AddParticipantID",
		"chat_id":        id,
		"participant_id": participantID,
	})

	// add the participant to the chat
	res, err := h.addParticipantID(ctx, id, participantID)
	if err != nil {
		log.Errorf("Could not add the participant. err: %v", err)
		return nil, err
	}

	// get chatrooms
	curTime := dbhandler.GetCurTime()
	chatrooms, err := h.chatroomHandler.GetsByChatID(ctx, id, curTime, 100000)
	if err != nil {
		log.Errorf("Could not get list of chatrooms. err: %v", err)
		return nil, err
	}

	// update the each chatrooms
	for _, cr := range chatrooms {
		tmp, err := h.chatroomHandler.AddParticipantID(ctx, cr.ID, participantID)
		if err != nil {
			log.Errorf("Could not add the participant. chatroom_id: %s, err: %v", tmp.ID, err)
			continue
		}
		log.WithField("chatroom", tmp).Debugf("Updated chatroom participant. chatroom_id: %s", tmp.ID)
	}

	// crate a new chatroom for new participant
	chatroomType := chatroom.ConvertType(res.Type)
	newChatroom, err := h.chatroomHandler.Create(
		ctx,
		res.CustomerID,
		chatroomType,
		res.ID,
		participantID,
		res.ParticipantIDs,
		res.Name,
		res.Detail,
	)
	if err != nil {
		log.Errorf("Could not create a new chatroom for new participant. err: %v", err)
	}
	log.WithField("chatroom", newChatroom).Debugf("Created a new chatroom. chatroom_id: %s", newChatroom.ID)

	return res, nil
}

// AddParticipantID adds the given pariticipant_id to the given chat's pariticipant_ids
func (h *chatHandler) addParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "addParticipantID",
		"chat_id":        id,
		"participant_id": participantID,
	})

	if errRemove := h.db.ChatAddParticipantID(ctx, id, participantID); errRemove != nil {
		log.Errorf("Could not add the participant id to the chat. err: %v", errRemove)
		return nil, errRemove
	}

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chat.EventTypeChatUpdated, res)

	return res, nil
}

// RemoveParticipantID adds the given pariticipant_id to the given chat's pariticipant_ids
func (h *chatHandler) RemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RemoveParticipantID",
		"chat_id":        id,
		"participant_id": participantID,
	})

	// remove the participant from the chat
	res, err := h.removeParticipantID(ctx, id, participantID)
	if err != nil {
		log.Errorf("Could not add the participant. err: %v", err)
		return nil, err
	}

	// get chatrooms
	curTime := dbhandler.GetCurTime()
	chatrooms, err := h.chatroomHandler.GetsByChatID(ctx, id, curTime, 100000)
	if err != nil {
		log.Errorf("Could not get list of chatrooms. err: %v", err)
		return nil, err
	}

	// update the each chatrooms
	chatroomID := uuid.Nil
	for _, cr := range chatrooms {
		if cr.OwnerID == participantID {
			chatroomID = cr.ID
		}

		tmp, err := h.chatroomHandler.RemoveParticipantID(ctx, cr.ID, participantID)
		if err != nil {
			log.Errorf("Could not add the participant. chatroom_id: %s, err: %v", tmp.ID, err)
			continue
		}
		log.WithField("chatroom", tmp).Debugf("Updated chatroom participant. chatroom_id: %s", tmp.ID)
	}

	// delete the removed participant's chatroom
	tmp, err := h.chatroomHandler.Delete(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return res, nil
	}
	log.WithField("chatroom", tmp).Debugf("Deleted removed participant's chatroom. chatroom_id: %s", tmp.ID)

	return res, nil
}

// removeParticipantID removes the given pariticipant_id from the given chat's pariticipant_ids
func (h *chatHandler) removeParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "removeParticipantID",
		"chat_id":        id,
		"participant_id": participantID,
	})

	if errRemove := h.db.ChatRemoveParticipantID(ctx, id, participantID); errRemove != nil {
		log.Errorf("Could not remove the participant id from the chat. err: %v", errRemove)
		return nil, errRemove
	}

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chat.EventTypeChatUpdated, res)

	return res, nil
}

// Delete deletes the chat
func (h *chatHandler) Delete(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Delete",
		"chat_id": id,
	})

	if errDelete := h.db.ChatDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the chat. err: %v", errDelete)
		return nil, errDelete
	}

	// get
	res, err := h.db.ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted chat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chat.EventTypeChatDeleted, res)

	return res, nil
}
