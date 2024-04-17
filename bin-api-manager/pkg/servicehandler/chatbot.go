package servicehandler

import (
	"context"
	"fmt"

	chatbotchatbot "monorepo/bin-chatbot-manager/models/chatbot"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// chatbotGet validates the chatbot's ownership and returns the chatbot info.
func (h *serviceHandler) chatbotGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbot.Chatbot, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatbotGet",
		"customer_id": a.CustomerID,
		"chatbot_id":  id,
	})

	// send request
	res, err := h.reqHandler.ChatbotV1ChatbotGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the resource info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ChatbotCreate is a service handler for chatbot creation.
func (h *serviceHandler) ChatbotCreate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	engineType chatbotchatbot.EngineType,
	initPrompt string,
) (*chatbotchatbot.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatbotCreate",
		"customer_id": a.CustomerID,
		"name":        name,
		"detail":      detail,
		"engine_type": engineType,
		"init_prompt": initPrompt,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.ChatbotV1ChatbotCreate(
		ctx,
		a.CustomerID,
		name,
		detail,
		engineType,
		initPrompt,
	)
	if err != nil {
		log.Errorf("Could not create a new chatbot. err: %v", err)
		return nil, err
	}
	log.WithField("chatbot", tmp).Debug("Created a new chatbot.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatbotGetsByCustomerID gets the list of chatbots of the given customer id.
// It returns list of chatbots if it succeed.
func (h *serviceHandler) ChatbotGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatbotchatbot.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatbotGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"deleted": "false", // we don't need deleted items
	}

	// get chatbots
	tmps, err := h.reqHandler.ChatbotV1ChatbotGetsByCustomerID(ctx, a.CustomerID, token, size, filters)
	if err != nil {
		log.Errorf("Could not get chatbots info from the chatobt manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*chatbotchatbot.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatbotGet gets the chatbot of the given id.
// It returns chatbot if it succeed.
func (h *serviceHandler) ChatbotGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbot.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatbotGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chatbot_id":  id,
	})

	// get chatbot
	tmp, err := h.chatbotGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatbot info from the chatobt manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatbot info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatbotDelete deletes the chatbot.
func (h *serviceHandler) ChatbotDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbot.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatbotDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chatbot_id":  id,
	})
	log.Debug("Deleting a chatbot.")

	// get chat
	c, err := h.chatbotGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatbot info from the chatbot-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatbot info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.ChatbotV1ChatbotDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatbot. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatbotUpdate is a service handler for chatbot update.
func (h *serviceHandler) ChatbotUpdate(
	ctx context.Context,
	a *amagent.Agent,
	id uuid.UUID,
	name string,
	detail string,
	engineType chatbotchatbot.EngineType,
	initPrompt string,
) (*chatbotchatbot.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatbotUpdate",
		"customer_id": a.CustomerID,
		"id":          id,
		"name":        name,
		"detail":      detail,
		"engine_type": engineType,
		"init_prompt": initPrompt,
	})

	// get chat
	c, err := h.chatbotGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chatbot info from the chatbot-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatbot info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.ChatbotV1ChatbotUpdate(
		ctx,
		id,
		name,
		detail,
		engineType,
		initPrompt,
	)
	if err != nil {
		log.Errorf("Could not update the chatbot. err: %v", err)
		return nil, err
	}
	log.WithField("chatbot", tmp).Debugf("Updated chatbot info. chatbot_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
