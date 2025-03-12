package engine_dialogflow_handler

import (
	"context"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/message"
)

type EngineDialogflowHandler interface {
	MessageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, m *message.Message) (*message.Message, error)
}

type engineDialogflowHandler struct {
}

func NewEngineDialogflowHandler() EngineDialogflowHandler {
	return &engineDialogflowHandler{}
}
