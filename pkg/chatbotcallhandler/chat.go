package chatbotcallhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
)

// Chat sends/receives the messages from/to a chatbot
func (h *chatbotcallHandler) Chat(ctx context.Context, cb *chatbotcall.Chatbotcall, message string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "Chat",
			"chatbot_id":     cb.ID,
			"reference_type": cb.ReferenceType,
			"reference_id":   cb.ReferenceID,
		},
	)

	// currently only the reference type call supported
	if cb.ReferenceType != chatbotcall.ReferenceTypeCall {
		log.Errorf("Unsupported reference type. reference_type: %s", cb.ReferenceType)
		return fmt.Errorf("unsupported referencd type")
	}

	// stop the media because chat will talk soon
	if errStop := h.reqHandler.CallV1CallMediaStop(ctx, cb.ReferenceID); errStop != nil {
		log.Errorf("Could not stop the media. err: %v", errStop)
		return errors.Wrap(errStop, "Could not stop the media")
	}

	// get chatbot info
	c, err := h.chatbotHandler.Get(ctx, cb.ChatbotID)
	if err != nil {
		log.Errorf("Could not get chatbot info. err: %v", err)
		return errors.Wrap(err, "could not get chatbot info")
	}

	text := ""
	switch c.EngineType {
	case chatbot.EngineTypeChatGPT:
		// chat to the chatbot engine and get answer from them.
		text, err = h.chatgptHandler.Chat(ctx, message)
		if err != nil {
			log.Errorf("Could not get chat message from the chatbot engine. err: %v", err)
			return errors.Wrap(err, "could not get chat message from the chatbot engine")
		}

	default:
		log.Errorf("Could not find correct chatbot engine handler. engine_type: %s", c.EngineType)
		return fmt.Errorf("could not find chatbot engine type handler. engine_type: %s", c.EngineType)
	}

	if text == "" {
		// nothing to say.
		return nil
	}

	// talk to the call
	if errTalk := h.reqHandler.CallV1CallTalk(ctx, cb.ReferenceID, text, string(cb.Gender), cb.Language, 10000); errTalk != nil {
		log.Errorf("Could not talk to the call. err: %v", errTalk)
		return errors.Wrap(errTalk, "could not talk to the call")
	}

	return nil
}
