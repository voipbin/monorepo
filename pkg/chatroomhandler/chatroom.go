package chatroomhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
)

// Get returns the chat
func (h *chatroomHandler) Get(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Get",
		"chat_id": id,
	})

	// get
	res, err := h.db.ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns the chatrooms by the given filters.
func (h *chatroomHandler) Gets(ctx context.Context, token string, limit uint64, filters map[string]string) ([]*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	// get
	res, err := h.db.ChatroomGets(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new chat
func (h *chatroomHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	agentID uuid.UUID,
	chatroomType chatroom.Type,
	chatID uuid.UUID,
	ownerID uuid.UUID,
	participantIDs []uuid.UUID,
	name string,
	detail string,
) (*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"customer_id":     customerID,
		"agent_id":        agentID,
		"type":            chatroomType,
		"chat_id":         chatID,
		"owner_id":        ownerID,
		"participant_ids": participantIDs,
	})

	id := uuid.Must(uuid.NewV4())
	curTime := h.utilHandler.TimeGetCurTime()
	tmp := &chatroom.Chatroom{
		ID:             id,
		CustomerID:     customerID,
		AgentID:        agentID,
		Type:           chatroomType,
		ChatID:         chatID,
		OwnerID:        ownerID,
		ParticipantIDs: participantIDs,
		Name:           name,
		Detail:         detail,
		TMCreate:       curTime,
		TMUpdate:       curTime,
		TMDelete:       dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.ChatroomCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new chatroom correctly. err: %v", errCreate)
		return nil, errCreate
	}

	// get
	res, err := h.db.ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a created chatroom. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatroom.EventTypeChatroomCreated, res)

	return res, nil
}

// UpdateBasicInfo updates the chat's basic info
func (h *chatroomHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateBasicInfo",
		"chat_id": id,
	})

	if errUpdate := h.db.ChatroomUpdateBasicInfo(ctx, id, name, detail); errUpdate != nil {
		log.Errorf("Could not update the chatroom's basic info. err: %v", errUpdate)
		return nil, errUpdate
	}

	// get
	res, err := h.db.ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chatroom info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatroom.EventTypeChatroomUpdated, res)

	return res, nil
}

// AddParticipantID adds the given pariticipant_id to the given chatroom's pariticipant_ids
func (h *chatroomHandler) AddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AddParticipantID",
		"chat_id":        id,
		"participant_id": participantID,
	})

	// send a request to the chathandler

	if errRemove := h.db.ChatroomAddParticipantID(ctx, id, participantID); errRemove != nil {
		log.Errorf("Could not add the participant id to the chatroom. err: %v", errRemove)
		return nil, errRemove
	}

	// get
	res, err := h.db.ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chatroom info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatroom.EventTypeChatroomUpdated, res)

	return res, nil
}

// RemoveParticipantID removes the given pariticipant_id from the given chatroom's pariticipant_ids
func (h *chatroomHandler) RemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RemoveParticipantID",
		"chat_id":        id,
		"participant_id": participantID,
	})

	if errRemove := h.db.ChatroomRemoveParticipantID(ctx, id, participantID); errRemove != nil {
		log.Errorf("Could not remove the participant id from the chatroom. err: %v", errRemove)
		return nil, errRemove
	}

	// get
	res, err := h.db.ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated chat info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatroom.EventTypeChatroomUpdated, res)

	return res, nil
}

// Delete deletes the chatroom
func (h *chatroomHandler) Delete(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Delete",
		"chat_id": id,
	})

	if errDelete := h.db.ChatroomDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the chatroom. err: %v", errDelete)
		return nil, errDelete
	}

	// get
	res, err := h.db.ChatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted chatroom info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, chatroom.EventTypeChatroomDeleted, res)

	return res, nil
}
