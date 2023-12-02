package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
)

// chatroomGet validates the chatroom's ownership and returns the chatroom info.
func (h *serviceHandler) chatroomGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchatroom.Chatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatroomGet",
		"customer_id": a.CustomerID,
		"chatroom_id": id,
	})

	// send request
	res, err := h.reqHandler.ChatV1ChatroomGet(ctx, id)
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
	owner, err := h.agentGet(ctx, a, ownerID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("owner", owner).Debugf("Found owner info. owner_id: %s", owner.ID)

	if !h.hasPermission(ctx, a, owner.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	tmps, err := h.reqHandler.ChatV1ChatroomGetsByOwnerID(ctx, ownerID, token, size)
	if err != nil {
		log.Errorf("Could not get chats info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*chatchatroom.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

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
	tmp, err := h.chatroomGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatroom info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroom info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
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
	cr, err := h.chatroomGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, cr.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ChatV1ChatroomDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
