package engine_openai_handler

//go:generate mockgen -package engine_openai_handler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

const (
	defaultModel = "gpt-4-turbo"
)

// EngineOpenaiHandler define
type EngineOpenaiHandler interface {
	MessageSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (*message.Message, error)

	Send(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
	StreamingSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (<-chan string, error)
}

// engineOpenaiHandler define
type engineOpenaiHandler struct {
	client *openai.Client
}

// NewEngineOpenaiHandler define
func NewEngineOpenaiHandler(apiKey string) EngineOpenaiHandler {
	client := openai.NewClient(apiKey)

	return &engineOpenaiHandler{
		client: client,
	}
}
