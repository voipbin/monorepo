package engine_openai_handler

//go:generate mockgen -package engine_openai_handler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"

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
	SendOnce(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
	StreamingSend(ctx context.Context, cc *aicall.AIcall, messages []*message.Message) (<-chan string, <-chan *message.ToolCall, error)
}

// KeyFunc hands the API key to the engine at the time of a call. An empty value with no error is
// "no key configured" and the request is sent without one, as before. An error stops the call.
type KeyFunc = func(ctx context.Context) (string, error)

// engineOpenaiHandler define
//
// The key is resolved when a request is sent, not when the handler is built, so a rotated key is
// used by the next request. The client is cheap to create: the SDK client shares the default HTTP
// transport, so the connection pool is shared.
type engineOpenaiHandler struct {
	keyFunc KeyFunc
	baseURL string
}

// NewEngineOpenaiHandler define
func NewEngineOpenaiHandler(keyFunc KeyFunc) EngineOpenaiHandler {
	return &engineOpenaiHandler{
		keyFunc: keyFunc,
	}
}

// NewEngineOpenaiHandlerWithConfig builds an engine against a custom base URL
// (e.g. the Gemini OpenAI-compatible endpoint). Used by the analysis gateway so
// the provider is selectable by config. When baseURL is empty the SDK default
// (OpenAI, https://api.openai.com/v1) is used.
func NewEngineOpenaiHandlerWithConfig(keyFunc KeyFunc, baseURL string) EngineOpenaiHandler {
	return &engineOpenaiHandler{
		keyFunc: keyFunc,
		baseURL: baseURL,
	}
}

// newClient resolves the key now and builds a client with it. The error is the error of the key
// function, unchanged, so a caller can tell a key that could not be read from a provider error.
func (h *engineOpenaiHandler) newClient(ctx context.Context) (*openai.Client, error) {
	if h.keyFunc == nil {
		return nil, errors.New("engine_openai_handler: no key function")
	}

	apiKey, err := h.keyFunc(ctx)
	if err != nil {
		return nil, err
	}

	cfg := openai.DefaultConfig(apiKey)
	if h.baseURL != "" {
		cfg.BaseURL = h.baseURL
	}

	return openai.NewClientWithConfig(cfg), nil
}

const (
	defaultSystemPrompt = `
	Role:
You are an AI assistant integrated with voipbin. 
Your role is to follow the user's system or custom prompt strictly, provide natural responses, and call external tools when necessary.

Context:
- Users will set their own instructions (persona, style, context).
- You must adapt to those instructions consistently.
- If user requests or situation requires, use available tools to gather data or perform actions.

Input Values:
- User-provided system/custom prompt
- User query
- Available tools list

Instructions:
- Always prioritize the user's provided prompt instructions.
- Generate a helpful, coherent, and contextually appropriate response.
- If tools are available and required, call them responsibly and return results clearly.
- **Do not mention tool names or the fact that a tool is being used in the user-facing response.**
- Maintain consistency with the user-defined tone and role.
- If ambiguity exists, ask clarifying questions before answering.
- Before giving the final answer, outline a short execution plan (2–4 steps), then provide a concise summary (1–2 sentences) and the final answer.  
- For each Input Value, ask clarifying questions **one at a time in sequence**. Wait for the user's answer before moving to the next question.  

Constraints:
- Avoid hallucination; use tools for factual queries.  
- Keep answers aligned with user's persona and tone.  
- Respect conversation history and continuity.  
	`
)
