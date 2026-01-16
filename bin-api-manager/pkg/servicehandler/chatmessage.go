package servicehandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// chatmessageGet validates the chatmessage's ownership and returns the chat info.
func (h *serviceHandler) chatmessageGet(ctx context.Context, id uuid.UUID) (*chatmessagechat.Messagechat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "chatmessageGet",
		"chat_id": id,
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

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.chatmessageCreate(ctx, a.CustomerID, chatID, source, messageType, text, medias)
	if err != nil {
		log.Errorf("Could not create a new messagechat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) chatmessageCreate(
	ctx context.Context,
	customerID uuid.UUID,
	chatID uuid.UUID,
	source commonaddress.Address,
	messageType chatmessagechat.Type,
	text string,
	medias []chatmedia.Media,
) (*chatmessagechat.Messagechat, error) {
	res, err := h.reqHandler.ChatV1MessagechatCreate(
		ctx,
		customerID,
		chatID,
		source,
		messageType,
		text,
		medias,
	)
	if err != nil {
		return nil, err
	}

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
	tmp, err := h.chatGet(ctx, chatID)
	if err != nil {
		log.Errorf("Could not get chat info. err: %v", err)
		return nil, err
	}
	log.WithField("chat", tmp).Debugf("Found chat info. chat_id: %s", tmp.ID)

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	filters := map[string]string{
		"deleted": "false",
		"chat_id": chatID.String(),
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertChatMessageFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.reqHandler.ChatV1MessagechatList(ctx, token, size, typedFilters)
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

// convertChatMessageFilters converts map[string]string to map[chatmessagechat.Field]any
func (h *serviceHandler) convertChatMessageFilters(filters map[string]string) (map[chatmessagechat.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, chatmessagechat.Messagechat{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[chatmessagechat.Field]any, len(typed))
	for k, v := range typed {
		result[chatmessagechat.Field(k)] = v
	}

	return result, nil
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
	tmp, err := h.chatmessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatmessage info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
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
	cm, err := h.chatmessageGet(ctx, id)
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
