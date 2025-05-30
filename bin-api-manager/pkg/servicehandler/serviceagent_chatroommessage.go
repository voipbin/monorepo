package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentChatroommessageGet gets the chatroommessage of the given id.
// It returns chatroommessage if it succeed.
func (h *serviceHandler) ServiceAgentChatroommessageGet(ctx context.Context, a *amagent.Agent, chatroomMessageID uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "ServiceAgentChatroommessageGet",
		"agent":               a,
		"chatroom_message_id": chatroomMessageID,
	})
	log.Debug("Getting a chatroommessage.")

	// get chat
	tmp, err := h.chatroommessageGet(ctx, chatroomMessageID)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	if tmp.OwnerID != a.ID {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentChatroommessageGets sends a request to chat-manager
// to getting the given chatroom's list of chatroom message.
// it returns list of chatroom messages if it succeed.
func (h *serviceHandler) ServiceAgentChatroommessageGets(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroommessageGets",
		"agent":       a,
		"chatroom_id": chatroomID,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmp, err := h.chatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", tmp).Debugf("Found chatroom info. chatroom_id: %s", chatroomID)

	if tmp.OwnerID != a.ID {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// filters
	filters := map[string]string{
		"chatroom_id": chatroomID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	tmps, err := h.chatroommessageGetsWithFilters(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get chatroom messages. err: %v", err)
		return nil, err
	}

	// create result
	res := []*chatmessagechatroom.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil

}

// ServiceAgentChatroommessageCreate creates the chatroom message of the given chatroom id.
// It returns created chatroommessages if it succeed.
func (h *serviceHandler) ServiceAgentChatroommessageCreate(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, message string, medias []chatmedia.Media) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroommessageCreate",
		"agent":       a,
		"chatroom_id": chatroomID,
		"message":     message,
		"medias":      medias,
	})
	log.Debug("Creating the chatroom message.")

	// get chatroom info
	cr, err := h.chatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	if cr.OwnerID != a.ID {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// create chatmessage
	source := commonaddress.Address{
		Type:       commonaddress.TypeAgent,
		Target:     a.ID.String(),
		TargetName: a.Name,
	}
	cm, err := h.chatmessageCreate(ctx, a.CustomerID, cr.ChatID, source, chatmessagechat.TypeNormal, message, medias)
	if err != nil {
		log.Errorf("Could not create chatmessage. err: %v", err)
		return nil, err
	}

	// get message chatroom by chatmessage.
	filters := map[string]string{
		"chatroom_id":    cr.ID.String(),
		"messagechat_id": cm.ID.String(),
	}
	tmps, err := h.chatroommessageGetsWithFilters(ctx, 1, h.utilHandler.TimeGetCurTime(), filters)
	if err != nil {
		log.Errorf("Could not get message chatroom. err: %v", err)
		return nil, err
	}

	if len(tmps) < 1 {
		log.Errorf("Could not create message chatroom.")
		return nil, fmt.Errorf("could not create chatroom message")
	}

	res := tmps[0].ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentChatroommessageDelete deletes the chatroom message.
func (h *serviceHandler) ServiceAgentChatroommessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "ServiceAgentChatroommessageDelete",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"chatroommessage_id": id,
	})
	log.Debug("Deleting a chatroommessage.")

	// get chatroommessage
	cr, err := h.chatroommessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	if cr.OwnerID != a.ID {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.chatroommessageDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroommessage. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
