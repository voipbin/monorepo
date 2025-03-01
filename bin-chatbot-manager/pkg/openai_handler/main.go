package openai_handler

//go:generate mockgen -package openai_handler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/message"
)

const (
	defaultModel = "gpt-4-turbo"
)

// OpenaiHandler define
type OpenaiHandler interface {
	MessageSend(ctx context.Context, cc *chatbotcall.Chatbotcall, messages []*message.Message) (*message.Message, error)
}

// openaiHandler define
type openaiHandler struct {
	client *openai.Client
}

// NewOpenaiHandler define
func NewOpenaiHandler(apiKey string) OpenaiHandler {
	client := openai.NewClient(apiKey)

	return &openaiHandler{
		client: client,
	}
}
