package servicehandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// chatroommessageGet gets the chatroommessage info.
func (h *serviceHandler) chatroommessageGet(ctx context.Context, id uuid.UUID) (*chatmessagechatroom.Messagechatroom, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "chatroommessageGet",
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
	tmp, err := h.chatroommessageGet(ctx, id)
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

// ChatroommessageCreate creates the chatroom message of the given chatroom id.
// It returns created chatroommessages if it succeed.
func (h *serviceHandler) ChatroommessageCreate(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, message string, medias []chatmedia.Media) (*chatmessagechatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatroommessageCreate",
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

	if !h.hasPermission(ctx, a, cr.CustomerID, amagent.PermissionCustomerManager|amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// create chatmessage
	source := commonaddress.Address{
		Type:       commonaddress.TypeAgent,
		Target:     a.ID.String(),
		TargetName: a.Name,
	}
	cm, err := h.ChatmessageCreate(ctx, a, cr.ChatID, source, chatmessagechat.TypeNormal, message, medias)
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

	tmp, err := h.chatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get owner info. err: %v", err)
		return nil, err
	}
	log.WithField("chatroom", tmp).Debugf("Found chatroom info. chatroom_id: %s", chatroomID)

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerManager|amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	filters := map[string]string{
		"deleted":     "false",
		"chatroom_id": chatroomID.String(),
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

// ChatroommessageGetsWithFilters gets the list of chatroommessages of the given filters.
// It returns list of chatroommessages if it succeed.
func (h *serviceHandler) chatroommessageGetsWithFilters(ctx context.Context, size uint64, token string, filters map[string]string) ([]chatmessagechatroom.Messagechatroom, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	res, err := h.reqHandler.ChatV1MessagechatroomGets(ctx, token, size, filters)
	if err != nil {
		return nil, fmt.Errorf("could not find chatroommessages info. err: %v", err)
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
	cr, err := h.chatroommessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatroommessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatroommessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, cr.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
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

func (h *serviceHandler) chatroommessageDelete(ctx context.Context, id uuid.UUID) (*chatmessagechatroom.Messagechatroom, error) {
	res, err := h.reqHandler.ChatV1MessagechatroomDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}
