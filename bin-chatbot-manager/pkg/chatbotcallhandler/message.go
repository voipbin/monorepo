package chatbotcallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *chatbotcallHandler) messageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "messageSend",
		"chatbotcall": cc,
		"message":     message,
	})

	var tmpMessage *chatbotcall.Message
	var err error
	start := time.Now()

	// send message
	switch cc.ChatbotEngineType {
	case chatbot.EngineTypeChatGPT:
		tmpMessage, err = h.chatgptHandler.ChatMessage(ctx, cc, message)
		if err != nil {
			return nil, errors.Wrap(err, "could not get chat message from the chatbot engine")
		}

	default:
		return nil, fmt.Errorf("could not find chatbot engine type handler. engine_type: %s", cc.ChatbotEngineType)
	}
	elapsed := time.Since(start)
	promChatMessageProcessTime.WithLabelValues(string(cc.ChatbotEngineType)).Observe(float64(elapsed.Milliseconds()))
	log.WithField("response_message", tmpMessage).Debugf("Processed chat message. elapsed: %v, response_content: %s", elapsed, tmpMessage.Content)

	messages := append(cc.Messages, *message)
	messages = append(messages, *tmpMessage)

	// update chatbotcall messages
	res, err := h.UpdateChatbotcallMessages(ctx, cc.ID, messages)
	if err != nil {
		log.Errorf("Could not update the chatbotcall's messages. err: %v", err)
		return nil, errors.Wrap(err, "could not update the chatbotcall's messages")
	}

	return res, nil
}
