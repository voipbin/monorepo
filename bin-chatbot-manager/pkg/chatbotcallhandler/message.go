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

func (h *chatbotcallHandler) Message(ctx context.Context, cc *chatbotcall.Chatbotcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Message",
		"chatbotcall_id": cc.ID,
	})

	var messages []chatbotcall.Message
	var err error
	start := time.Now()
	// send message
	switch cc.ChatbotEngineType {
	case chatbot.EngineTypeChatGPT:
		// chat to the chatbot engine and get answer from them.
		messages, err = h.chatgptHandler.ChatMessage(ctx, cc, message)
		if err != nil {
			log.Errorf("Could not get chat message from the chatbot engine. err: %v", err)
			return errors.Wrap(err, "could not get chat message from the chatbot engine")
		}

	default:
		log.Errorf("Could not find correct chatbot engine handler. engine_type: %s", cc.ChatbotEngineType)
		return fmt.Errorf("could not find chatbot engine type handler. engine_type: %s", cc.ChatbotEngineType)
	}
	elapsed := time.Since(start)
	promChatMessageProcessTime.WithLabelValues(string(cc.ChatbotEngineType)).Observe(float64(elapsed.Milliseconds()))
	log.WithField("response_message", messages[len(messages)-1]).Debugf("Processed chat message. elapsed: %v, response_content: %s", elapsed, messages[len(messages)-1].Content)

	// update chatbotcall messages
	tmp, err := h.UpdateChatbotcallMessages(ctx, cc.ID, messages)
	if err != nil {
		log.Errorf("Could not update the chatbotcall's messages. err: %v", err)
		return errors.Wrap(err, "could not update the chatbotcall's messages")
	}

}
