package chatgpthandler

//go:generate mockgen -package chatgpthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

// ChatgptHandler define
type ChatgptHandler interface {
	ChatNew(ctx context.Context, cc *chatbotcall.Chatbotcall, initPrompt string) ([]chatbotcall.Message, error)
	ChatMessage(ctx context.Context, cc *chatbotcall.Chatbotcall, text string) ([]chatbotcall.Message, error)
}

// chatgptHandler define
type chatgptHandler struct {
	client *openai.Client
}

// NewChatgptHandler define
func NewChatgptHandler(apiKey string) ChatgptHandler {
	client := openai.NewClient(apiKey)

	return &chatgptHandler{
		client: client,
	}
}
