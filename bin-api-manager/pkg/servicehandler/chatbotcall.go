package servicehandler

import (
	"context"
	"fmt"

	chatbotchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// chatbotcallGet validates the chatbotcall's ownership and returns the chatbotcall info.
func (h *serviceHandler) chatbotcallGet(ctx context.Context, id uuid.UUID) (*chatbotchatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "chatbotcallGet",
		"chatbotcall_id": id,
	})

	// send request
	res, err := h.reqHandler.ChatbotV1ChatbotcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the resource info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ChatbotcallGetsByCustomerID gets the list of chatbotcalls of the given customer id.
// It returns list of chatbots if it succeed.
func (h *serviceHandler) ChatbotcallGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatbotchatbotcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatbotcallGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chatbotcalls.")

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

	// get chatbotcalls
	tmps, err := h.reqHandler.ChatbotV1ChatbotcallGetsByCustomerID(ctx, a.CustomerID, token, size, filters)
	if err != nil {
		log.Errorf("Could not get chatbotcalls info from the chatbot manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatbotcalls info. err: %v", err)
	}

	// create result
	res := []*chatbotchatbotcall.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatbotcallGet gets the chatbotcall of the given id.
// It returns chatbot if it succeed.
func (h *serviceHandler) ChatbotcallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbotcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ChatbotcallGet",
		"customer_id":    a.CustomerID,
		"username":       a.Username,
		"chatbotcall_id": id,
	})
	log.Debug("Getting a chatbotcall.")

	// get chatbot
	tmp, err := h.chatbotcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chatbot manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatbotcall info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatbotcallDelete deletes the chatbotcall.
func (h *serviceHandler) ChatbotcallDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbotcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ChatbotcallDelete",
		"customer_id":    a.CustomerID,
		"username":       a.Username,
		"chatbotcall_id": id,
	})
	log.Debug("Deleting a chatbotcall.")

	// get chatbotcall
	c, err := h.chatbotcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chatbot manager. err: %v", err)
		return nil, fmt.Errorf("could not find chatbotcall info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.ChatbotV1ChatbotcallDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatbotcall. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
