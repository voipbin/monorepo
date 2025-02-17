package chatgpthandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

// ChatNew starts a new chat
func (h *chatgptHandler) ChatNew(ctx context.Context, cc *chatbotcall.Chatbotcall, initPrompt string) ([]chatbotcall.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatNew",
		"init_prompt": initPrompt,
	})

	log.Debugf("Sending initial prompt message. init_prompt: %s", initPrompt)
	res, err := h.messageSend(ctx, cc, openai.ChatMessageRoleSystem, initPrompt)
	if err != nil {
		log.Errorf("Could not start a new chat. err: %v", err)
		return nil, errors.Wrap(err, "could not start a new chat")
	}
	log.Debugf("Init prompt done. res: %s", res)

	return res, nil
}

// ChatMessage sends/receives the chat messages
func (h *chatgptHandler) ChatMessage(ctx context.Context, cc *chatbotcall.Chatbotcall, text string) ([]chatbotcall.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "ChatMessage",
		"text": text,
	})

	res, err := h.messageSend(ctx, cc, openai.ChatMessageRoleUser, text)
	if err != nil {
		log.Errorf("Could not send the message. err: %v", err)
		return nil, errors.Wrap(err, "could not send the message")
	}

	return res, nil
}
