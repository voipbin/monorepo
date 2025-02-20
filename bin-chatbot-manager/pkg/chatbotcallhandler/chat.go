package chatbotcallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

// ChatMessage sends/receives the messages from/to a chatbot
func (h *chatbotcallHandler) ChatMessage(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) error {
	switch cc.ReferenceType {
	case chatbotcall.ReferenceTypeCall:
		return h.chatMessageReferenceTypeCall(ctx, cc, message)

	case chatbotcall.ReferenceTypeNone:
		return h.chatMessageReferenceTypeNone(ctx, cc, message)

	default:
		return fmt.Errorf("unsupported reference type. reference_type: %s", cc.ReferenceType)
	}
}

// chatMessageActionsHandle handles chat message actions
func (h *chatbotcallHandler) chatMessageActionsHandle(ctx context.Context, cc *chatbotcall.Chatbotcall, actions []fmaction.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatMessageActionsHandle",
		"chatbotcall": cc,
		"actions":     actions,
	})

	// push the actions
	af, err := h.reqHandler.FlowV1ActiveflowPushActions(ctx, cc.ActiveflowID, actions)
	if err != nil {
		log.Errorf("Could not push the actions to the activeflow. err: %v", err)
		return errors.Wrap(err, "could not push the actions to the activeflow")
	}
	log.WithField("activeflow", af).Debugf("Pushed actions to the activeflow. activeflow_id: %s", af.ID)

	// destroy the confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, cc.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not terminate the chatbotcall confbridge. err: %v", err)
		return errors.Wrap(err, "could not terminate the chatbotcall confbridge")
	}
	log.WithField("confbridge", tmp).Debugf("Terminated confbridge. confbridge_id: %s", tmp.ID)

	return nil
}

// chatMessageTextHandle handles chat message text
func (h *chatbotcallHandler) chatMessageTextHandle(ctx context.Context, cc *chatbotcall.Chatbotcall, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatMessageTextHandle",
		"chatbotcall": cc,
		"text":        text,
	})

	if errTalk := h.reqHandler.CallV1CallTalk(ctx, cc.ReferenceID, text, string(cc.Gender), cc.Language, 10000); errTalk != nil {
		log.Errorf("Could not talk to the call. err: %v", errTalk)
		return errors.Wrap(errTalk, "could not talk to the call")
	}

	return nil
}

// ChatInit sends the chat's init_prompt
func (h *chatbotcallHandler) ChatInit(ctx context.Context, cb *chatbot.Chatbot, cc *chatbotcall.Chatbotcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatInit",
		"chatbot":     cb,
		"chatbotcall": cc,
	})

	message := &chatbotcall.Message{
		Role:    chatbotcall.MessageRoleSystem,
		Content: cb.InitPrompt,
	}

	var err error
	var tmpMessage *chatbotcall.Message
	start := time.Now()
	switch cb.EngineType {
	case chatbot.EngineTypeChatGPT:
		tmpMessage, err = h.chatgptHandler.ChatNew(ctx, cc, message)

	default:
		log.Errorf("Unsupported engine type. engine_type: %s", cb.EngineType)
		return fmt.Errorf("unsupported engine type")
	}
	if err != nil {
		log.Errorf("Could not start new chat. err: %v", err)
		return errors.Wrap(err, "could not start new chat")
	}

	messages := append(cc.Messages, *message)
	messages = append(messages, *tmpMessage)

	tmp, err := h.UpdateChatbotcallMessages(ctx, cc.ID, messages)
	if err != nil {
		log.Errorf("Could not update the chatbotcall messages. err: %v", err)
		return errors.Wrap(err, "could not update the chatbotcall messages")
	}
	log.WithField("chatbotcall", tmp).Debugf("Updated chatbotcall messages. chatbotcall_id: %s", tmp.ID)

	elapsed := time.Since(start)
	promChatInitProcessTime.WithLabelValues(string(cc.ChatbotEngineType)).Observe(float64(elapsed.Milliseconds()))
	log.Debugf("Chat has initialized. elapsed: %v", elapsed)

	return nil
}

func (h *chatbotcallHandler) chatMessageReferenceTypeCall(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatMessageReferenceTypeCall",
		"chatbotcall": cc,
		"message":     message,
	})

	// currently only the reference type call supported
	if cc.ReferenceType != chatbotcall.ReferenceTypeCall {
		log.Errorf("Unsupported reference type. reference_type: %s", cc.ReferenceType)
		return fmt.Errorf("unsupported referencd type")
	}

	// stop the media because chat will talk soon
	if errStop := h.reqHandler.CallV1CallMediaStop(ctx, cc.ReferenceID); errStop != nil {
		log.Errorf("Could not stop the media. err: %v", errStop)
		return errors.Wrap(errStop, "Could not stop the media")
	}

	tmp, err := h.messageSend(ctx, cc, message)
	if err != nil {
		return errors.Wrapf(err, "could not send the message to the chatbot. chatbotcall_id: %s", cc.ID)
	}

	// get response message text
	text := tmp.Messages[len(tmp.Messages)-1].Content
	if text == "" {
		// nothing to say.
		log.Debug("Nothing to say.")
		return nil
	}

	// check the response message
	tmpActions := []fmaction.Action{}
	errUnmarshal := json.Unmarshal([]byte(text), &tmpActions)
	if errUnmarshal == nil {
		log.WithField("actions", tmpActions).Debugf("Got a action arrays. len_actions: %d", len(tmpActions))
		if errHandle := h.chatMessageActionsHandle(ctx, cc, tmpActions); errHandle != nil {
			log.Errorf("Could not handle the response actions correctly. err: %v", errHandle)
			return errors.Wrap(err, "could not handle the response actions correctly")
		}
	} else {
		log.WithField("text", text).Debugf("Got an message text. text: %s", text)
		if errHandle := h.chatMessageTextHandle(ctx, cc, text); errHandle != nil {
			log.Errorf("Could not handle the response message text correctly. err: %v", errHandle)
			return errors.Wrap(err, "could not handle the response message text correctly")
		}
	}

	return nil
}

func (h *chatbotcallHandler) chatMessageReferenceTypeNone(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) error {
	_, err := h.messageSend(ctx, cc, message)
	if err != nil {
		return errors.Wrapf(err, "could not send the message to the chatbot. chatbotcall_id: %s", cc.ID)
	}

	return nil
}
