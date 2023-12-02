package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
)

// chatroommessageGet validates the chatroommessage's ownership and returns the chatroommessage info.
func (h *serviceHandler) chatroommessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "chatroommessageGet",
		"customer_id":        a.CustomerID,
		"chatroommessage_id": id,
	})

	// send request
	res, err := h.reqHandler.ChatV1MessagechatroomGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the chatroommessage info. err: %v", err)
		return nil, err
	}
	log.WithField("messagechatroom", res).Debug("Received result.")

	// create result
	return res, nil
}

// ChatroommessageGet gets the chatroommessage of the given id.
// It returns chatroommessage if it succeed.
func (h *serviceHandler) ChatroommessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroommessageGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Getting a chatroommessage.")

	// get chat
	tmp, err := h.chatroommessageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatroommessageGetsByChatroomID gets the list of chatroommessages of the given owner id.
// It returns list of chatroommessages if it succeed.
func (h *serviceHandler) ChatroommessageGetsByChatroomID(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroommessageGetsByChatroomID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chatroom_id": chatroomID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chatroommessages.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmp, err := h.chatroomGet(ctx, a, chatroomID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", tmp).Debugf("Found chatroom info. chatroom_id: %s", chatroomID)

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	tmps, err := h.reqHandler.ChatV1MessagechatroomGetsByChatroomID(ctx, chatroomID, token, size)
	if err != nil {
		log.Errorf("Could not get chatroommessages info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessages info. err: %v", err)
	}

	// create result
	res := []*chatmessagechatroom.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatroommessageDelete deletes the chatroom.
func (h *serviceHandler) ChatroommessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "ChatroommessageDelete",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"chatroommessage_id": id,
	})
	log.Debug("Deleting a chatroommessage.")

	// get chatroommessage
	cr, err := h.chatroommessageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, cr.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ChatV1MessagechatroomDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroommessage. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
