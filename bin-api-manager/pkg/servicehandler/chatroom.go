package servicehandler

import (
	"context"
	"fmt"

	chatchat "monorepo/bin-chat-manager/models/chat"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// chatroomGet returns the chatroom info.
func (h *serviceHandler) chatroomGet(ctx context.Context, chatroomID uuid.UUID) (*chatchatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatroomGet",
		"chatroom_id": chatroomID,
	})

	// send request
	res, err := h.reqHandler.ChatV1ChatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get the chatroom info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", res).Debug("Received result.")

	// create result
	return res, nil
}

// ChatroomGetsByOwnerID gets the list of chatrooms of the given owner id.
// It returns list of chatrooms if it succeed.
func (h *serviceHandler) ChatroomGetsByOwnerID(ctx context.Context, a *amagent.Agent, ownerID uuid.UUID, size uint64, token string) ([]*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomGetsByOwnerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"owner_id":    ownerID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chatrooms.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get owner
	owner, err := h.agentGet(ctx, ownerID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("owner", owner).Debugf("Found owner info. owner_id: %s", owner.ID)

	if !h.hasPermission(ctx, a, owner.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	filters := map[string]string{
		"deleted":  "false",
		"owner_id": owner.ID.String(),
	}

	tmps, err := h.chatroomGetsByFilters(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get chatrooms list. err: %v", err)
		return nil, err
	}

	// create result
	res := []*chatchatroom.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// chatroomGetsByFilters gets the list of chatrooms of the given filters.
// It returns list of chatrooms if it succeed.
func (h *serviceHandler) chatroomGetsByFilters(ctx context.Context, size uint64, token string, filters map[string]string) ([]chatchatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "chatroomGetsByFilters",
		"size":    size,
		"token":   token,
		"filters": filters,
	})
	log.Debug("Getting a chatrooms.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertChatroomFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	// get chatrooms
	res, err := h.reqHandler.ChatV1ChatroomGets(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get chats info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	return res, nil
}

// chatroomGetByChatIDAndOwnerID returns the chatroom info of the given chat_id and owner_id.
func (h *serviceHandler) chatroomGetByChatIDAndOwnerID(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, ownerID uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatroomGetByChatIDAndOwnerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     chatID,
		"owner_id":    ownerID,
	})
	log.Debug("Getting a chatrooms.")

	filters := map[string]string{
		"deleted":  "false",
		"chat_id":  chatID.String(),
		"owner_id": ownerID.String(),
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertChatroomFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.reqHandler.ChatV1ChatroomGets(ctx, h.utilHandler.TimeGetCurTime(), 1, typedFilters)
	if err != nil {
		log.Errorf("Could not get chatroom info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroom info. err: %v", err)
	}

	if len(tmps) < 1 {
		log.Errorf("Could not get chatroom info.")
		return nil, fmt.Errorf("could not get chatroom info")
	}

	res := tmps[0].ConvertWebhookMessage()
	return res, nil
}

// ChatroomGet gets the chatroom of the given id.
// It returns chatroom if it succeed.
func (h *serviceHandler) ChatroomGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chatroom_id": id,
	})
	log.Debug("Getting a chatroom.")

	// get chat
	tmp, err := h.chatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatroom info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroom info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatroomUpdateBasicInfo updates the chatroom's basic info.
// It returns updated chatroom if it succeed.
func (h *serviceHandler) ChatroomUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chatroom_id": id,
	})
	log.Debug("Updating the chatroom.")

	// get chat
	c, err := h.chatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) || a.ID != c.OwnerID {
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.chatroomUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatroomUpdateBasicInfo updates the chatroom's basic info.
// It returns updated chatroom if it succeed.
func (h *serviceHandler) chatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*chatchatroom.Chatroom, error) {
	res, err := h.reqHandler.ChatV1ChatroomUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the chat. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ChatroomDelete deletes the chatroom.
func (h *serviceHandler) ChatroomDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroomDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chatroom_id": id,
	})
	log.Debug("Deleting a chat.")

	// get chat
	cr, err := h.chatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, cr.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.chatroomDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// chatroomDelete deletes the chatroom.
func (h *serviceHandler) chatroomDelete(ctx context.Context, id uuid.UUID) (*chatchatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatroomDelete",
		"chatroom_id": id,
	})
	log.Debug("Deleting a chat.")

	res, err := h.reqHandler.ChatV1ChatroomDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ChatroomCreate creates the chatroom message of the given chatroom id.
// It returns created chatroommessages if it succeed.
func (h *serviceHandler) ChatroomCreate(ctx context.Context, a *amagent.Agent, participantIDs []uuid.UUID, name string, detail string) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ChatroomCreate",
		"agent":           a,
		"participant_ids": participantIDs,
		"name":            name,
		"detail":          detail,
	})
	log.Debug("Creating the chatroom.")

	// check participant ids
	found := false
	for _, participantID := range participantIDs {
		if participantID == a.ID {
			found = true
			continue
		}

		tmp, err := h.agentGet(ctx, participantID)
		if err != nil {
			log.Errorf("Could not get participant info. err: %v", err)
			return nil, err
		}

		if tmp.CustomerID != a.CustomerID {
			log.Errorf("Customer id does not match. customer_id: %s", tmp.CustomerID)
			return nil, fmt.Errorf("customer id does not match")
		}
	}
	if !found {
		log.Debugf("Could not find agent id in the participant ids. Adding automatically. agent_id: %s", a.ID)
		participantIDs = append(participantIDs, a.ID)
	}

	ct := chatchat.TypeNormal
	if len(participantIDs) > 2 {
		ct = chatchat.TypeGroup
	}

	c, err := h.chatCreate(ctx, a.CustomerID, ct, a.ID, participantIDs, name, detail)
	if err != nil {
		log.Errorf("Could not create the chat. err: %v", err)
		return nil, err
	}
	log.WithField("chat", c).Debugf("Created chat. chat_id: %s", c.ID)

	// get chatroom
	res, err := h.chatroomGetByChatIDAndOwnerID(ctx, a, c.ID, a.ID)
	if err != nil {
		log.Errorf("Could not get created chatroom info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", res).Debugf("Created chatroom. chatroom_id: %s", res.ID)

	return res, nil
}

// convertChatroomFilters converts map[string]string to map[chatchatroom.Field]any
func (h *serviceHandler) convertChatroomFilters(filters map[string]string) (map[chatchatroom.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, chatchatroom.Chatroom{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[chatchatroom.Field]any, len(typed))
	for k, v := range typed {
		result[chatchatroom.Field(k)] = v
	}

	return result, nil
}
