package messagehandler

import (
	"context"
	"fmt"
	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/message"
	"slices"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *messageHandler) Send(ctx context.Context, chatbotcallID uuid.UUID, role message.Role, content string) (*message.Message, error) {
	cc, err := h.chatbotcallHandler.Get(ctx, chatbotcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the chatbotcall correctly")
	}

	// create a message for outgoing(request)
	_, err = h.Create(ctx, cc.CustomerID, chatbotcallID, message.DirectionOutgoing, role, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the sending message correctly")
	}

	t1 := time.Now()
	var tmpMessage *message.Message
	switch cc.ChatbotEngineType {
	case chatbot.EngineTypeChatGPT:
		tmpMessage, err = h.sendChatGPT(ctx, cc)

	default:
		err = fmt.Errorf("unsupported chatbot engine type: %s", cc.ChatbotEngineType)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}
	t2 := time.Since(t1)
	promMessageProcessTime.WithLabelValues(string(cc.ChatbotEngineType)).Observe(float64(t2.Milliseconds()))

	// create a message for incoming(response)
	res, err := h.Create(ctx, cc.CustomerID, cc.ID, message.DirectionIncoming, tmpMessage.Role, tmpMessage.Content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the recevied message correctly")
	}

	return res, nil
}

func (h *messageHandler) sendChatGPT(ctx context.Context, cc *chatbotcall.Chatbotcall) (*message.Message, error) {
	filters := map[string]string{
		"deleted": "false",
	}

	// note: because of chatgpt needs entire message history, we need to send all messages
	messages, err := h.Gets(ctx, cc.ID, 1000, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages correctly")
	}

	slices.Reverse(messages)
	res, err := h.chatgptHandler.MessageSend(ctx, cc, messages)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return res, nil
}
