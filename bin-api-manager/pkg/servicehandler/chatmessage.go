package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// chatmessageGet validates the chatmessage's ownership and returns the chat info.
func (h *serviceHandler) chatmessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatmessageGet",
		"customer_id": a.CustomerID,
		"chat_id":     id,
	})

	// send request
	res, err := h.reqHandler.ChatV1MessagechatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the messagechat info. err: %v", err)
		return nil, err
	}
	log.WithField("messagechat", res).Debug("Received result.")

	// create result
	return res, nil
}

// ChatmessageCreate is a service handler for chatmessage creation.
func (h *serviceHandler) ChatmessageCreate(
	ctx context.Context,
	a *amagent.Agent,
	chatID uuid.UUID,
	source commonaddress.Address,
	messageType chatmessagechat.Type,
	text string,
	medias []chatmedia.Media,
) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatmessageCreate",
		"customer_id": a.CustomerID,
	})
	log.Debug("Creating a new messagechat.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ChatV1MessagechatCreate(
		ctx,
		a.CustomerID,
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

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatmessageGetsByChatID gets the list of chats of the given chat id.
// It returns list of messagechats if it succeed.
func (h *serviceHandler) ChatmessageGetsByChatID(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, size uint64, token string) ([]*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatmessageGetsByChatID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     chatID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a messagechats.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if size == 0 {
		size = 10
	}

	// get chat info
	tmp, err := h.chatGet(ctx, a, chatID)
	if err != nil {
		log.Errorf("Could not get chat info. err: %v", err)
		return nil, err
	}
	log.WithField("chat", tmp).Debugf("Found chat info. chat_id: %s", tmp.ID)

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	filters := map[string]string{
		"deleted": "false",
		"chat_id": chatID.String(),
	}
	tmps, err := h.reqHandler.ChatV1MessagechatGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get chats info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*chatmessagechat.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatmessageGet gets the chatmessage of the given id.
// It returns chat if it succeed.
func (h *serviceHandler) ChatmessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ChatmessageGet",
		"customer_id":    a.CustomerID,
		"username":       a.Username,
		"chatmessage_id": id,
	})
	log.Debug("Getting a ChatmessageGet.")

	// get chatmessage
	tmp, err := h.chatmessageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatmessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatmessageDelete deletes the chatmessage.
func (h *serviceHandler) ChatmessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ChatmessageDelete",
		"customer_id":    a.CustomerID,
		"username":       a.Username,
		"chatmessage_id": id,
	})
	log.Debug("Deleting a chatmessage.")

	// get chatmessage
	cm, err := h.chatmessageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatmessage info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatmessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, cm.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ChatV1MessagechatDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatmessage. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
