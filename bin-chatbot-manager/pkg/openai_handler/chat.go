package openai_handler

import (
	"context"

	"github.com/pkg/errors"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

// ChatNew starts a new chat
func (h *openaiHandler) ChatNew(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) (*chatbotcall.Message, error) {
	res, err := h.messageSend(ctx, cc, message)
	if err != nil {
		return nil, errors.Wrap(err, "could not start a new chat")
	}

	return res, nil
}

// ChatMessage sends the message and return the receives the messages
func (h *openaiHandler) ChatMessage(ctx context.Context, cc *chatbotcall.Chatbotcall, message *chatbotcall.Message) (*chatbotcall.Message, error) {
	res, err := h.messageSend(ctx, cc, message)
	if err != nil {
		return nil, errors.Wrap(err, "could not send the message")
	}

	return res, nil
}
