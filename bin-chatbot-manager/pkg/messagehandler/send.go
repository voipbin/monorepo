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
	"github.com/sirupsen/logrus"
)

func (h *messageHandler) Send(ctx context.Context, chatbotcallID uuid.UUID, role message.Role, content string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Send",
		"chatbotcall_id": chatbotcallID,
		"role":           role,
		"content":        content,
	})
	log.Debugf("Sending chatbot message.")

	cc, err := h.chatbotcallHandler.Get(ctx, chatbotcallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the chatbotcall correctly")
	}

	if cc.Status == chatbotcall.StatusEnd {
		return nil, errors.New("chatbotcall is already ended")
	}

	// create a message for outgoing(request)
	m, err := h.Create(ctx, cc.CustomerID, chatbotcallID, message.DirectionOutgoing, role, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the sending message correctly")
	}

	t1 := time.Now()
	var tmpMessage *message.Message

	modelTarget := chatbot.GetEngineModelTarget(cc.ChatbotEngineModel)
	switch modelTarget {
	case chatbot.EngineModelTargetOpenai:
		tmpMessage, err = h.sendOpenai(ctx, cc)

	case chatbot.EngineModelTargetDialogflow:
		tmpMessage, err = h.sendDialogflow(ctx, cc, m)

	default:
		err = fmt.Errorf("unsupported chatbot engine model: %s", cc.ChatbotEngineModel)

	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}
	log.Debugf("Received response message from the chatbot engine. message: %v", tmpMessage)

	t2 := time.Since(t1)
	promMessageProcessTime.WithLabelValues(string(cc.ChatbotEngineType)).Observe(float64(t2.Milliseconds()))

	if len(tmpMessage.Content) == 0 {
		// if the messsage is empty, return the message as it is
		return tmpMessage, nil
	}

	// create a message for incoming(response)
	res, err := h.Create(ctx, cc.CustomerID, cc.ID, message.DirectionIncoming, tmpMessage.Role, tmpMessage.Content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the recevied message correctly")
	}

	return res, nil
}

func (h *messageHandler) sendOpenai(ctx context.Context, cc *chatbotcall.Chatbotcall) (*message.Message, error) {
	filters := map[string]string{
		"deleted": "false",
	}

	// note: because of chatgpt needs entire message history, we need to send all messages
	messages, err := h.Gets(ctx, cc.ID, 1000, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages correctly")
	}

	slices.Reverse(messages)
	res, err := h.engineOpenaiHandler.MessageSend(ctx, cc, messages)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return res, nil
}

func (h *messageHandler) sendDialogflow(ctx context.Context, cc *chatbotcall.Chatbotcall, m *message.Message) (*message.Message, error) {
	res, err := h.engineDialogflowHandler.MessageSend(ctx, cc, m)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return res, nil
}
