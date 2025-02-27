package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	cbmessage "monorepo/bin-chatbot-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// chatbotmessageGet returns the chatbot message info.
func (h *serviceHandler) chatbotmessageGet(ctx context.Context, id uuid.UUID) (*cbmessage.Message, error) {
	res, err := h.reqHandler.ChatbotV1MessageGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the message info. id: %v", id)
	}
	return res, nil
}

// ChabotmessageCreate sends a message to the chatbotcall.
func (h *serviceHandler) ChatbotmessageCreate(
	ctx context.Context,
	a *amagent.Agent,
	chatbotcallID uuid.UUID,
	role cbmessage.Role,
	content string,
) (*cbmessage.WebhookMessage, error) {
	_, err := h.ChatbotcallGet(ctx, a, chatbotcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the chatbotcall info. id: %v", chatbotcallID)
	}

	tmp, err := h.reqHandler.ChatbotV1MessageSend(ctx, chatbotcallID, role, content, 30000)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new chatbot message. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatbotmessageGetsByChatbotcallID gets the list of chatbot messages of the given chat id.
// It returns list of chatbot messages if it succeed.
func (h *serviceHandler) ChatbotmessageGetsByChatbotcallID(ctx context.Context, a *amagent.Agent, chatbotcallID uuid.UUID, size uint64, token string) ([]*cbmessage.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if size == 0 {
		size = 100
	}

	_, err := h.ChatbotcallGet(ctx, a, chatbotcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the chatbotcall info. id: %v", chatbotcallID)
	}

	filters := map[string]string{
		"deleted":        "false",
		"chatbotcall_id": chatbotcallID.String(),
	}
	tmps, err := h.reqHandler.ChatbotV1MessageGetsByChatbotcallID(ctx, chatbotcallID, token, size, filters)
	if err != nil {
		return nil, fmt.Errorf("could not find chatbot messages info. err: %v", err)
	}

	res := []*cbmessage.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatbotmessageGet gets the chatbot message of the given id.
// It returns chatbot message if it succeed.
func (h *serviceHandler) ChatbotmessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cbmessage.WebhookMessage, error) {
	tmp, err := h.chatbotmessageGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not find chatbot message info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatbotmessageDelete deletes the chatbotmessage.
func (h *serviceHandler) ChatbotmessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cbmessage.WebhookMessage, error) {
	_, err := h.ChatbotmessageGet(ctx, a, id)
	if err != nil {
		return nil, fmt.Errorf("could not find chatbotmessage info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatbotV1MessageDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the chatbotmessage. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
