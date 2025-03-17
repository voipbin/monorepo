package engine_dialogflow_handler

//go:generate mockgen -package engine_dialogflow_handler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-ai-manager/models/chatbotcall"
	"monorepo/bin-ai-manager/models/message"
)

type EngineDialogflowHandler interface {
	MessageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, m *message.Message) (*message.Message, error)
}

type engineDialogflowHandler struct {
}

func NewEngineDialogflowHandler() EngineDialogflowHandler {
	return &engineDialogflowHandler{}
}
