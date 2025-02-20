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

func (h *chatbotcallHandler) messageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, message string) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "messageSend",
		"chatbotcall_id": cc.ID,
	})
	log.Debugf("Sending a message. message: %s", message)

	var messages []chatbotcall.Message
	var err error
	start := time.Now()

	// send message
	switch cc.ChatbotEngineType {
	case chatbot.EngineTypeChatGPT:
		messages, err = h.chatgptHandler.ChatMessage(ctx, cc, message)
		if err != nil {
			log.Errorf("Could not get chat message from the chatbot engine. err: %v", err)
			return nil, errors.Wrap(err, "could not get chat message from the chatbot engine")
		}

	default:
		log.Errorf("Could not find correct chatbot engine handler. engine_type: %s", cc.ChatbotEngineType)
		return nil, fmt.Errorf("could not find chatbot engine type handler. engine_type: %s", cc.ChatbotEngineType)
	}
	elapsed := time.Since(start)
	promChatMessageProcessTime.WithLabelValues(string(cc.ChatbotEngineType)).Observe(float64(elapsed.Milliseconds()))
	log.WithField("response_message", messages[len(messages)-1]).Debugf("Processed chat message. elapsed: %v, response_content: %s", elapsed, messages[len(messages)-1].Content)

	// update chatbotcall messages
	res, err := h.UpdateChatbotcallMessages(ctx, cc.ID, messages)
	if err != nil {
		log.Errorf("Could not update the chatbotcall's messages. err: %v", err)
		return nil, errors.Wrap(err, "could not update the chatbotcall's messages")
	}

	return res, nil
}
