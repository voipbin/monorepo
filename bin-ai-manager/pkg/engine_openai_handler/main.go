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
- Maintain consistency with the user-defined tone and role.  
- If ambiguity exists, ask clarifying questions before answering.  
- Let's think step by step.  
- Ask question one by one for each Input Values, do not ask once at a time.

Constraints:
- Avoid hallucination; use tools for factual queries.  
- Keep answers aligned with user's persona and tone.  
- Respect conversation history and continuity.  
- Answer in korean (unless user specifies otherwise).

Output Indicator:
- Output format: Plain Text (default), but adapt if user/system requires (e.g., JSON, bullet points, table).  
- Output fields:  
   1. Response (assistant'’'s reply)  
   2. Tool call (if used)  
- Output examples:

Response: Will connect you to the agent.  
Tool call: { "action": "connect", "destinations": [{"target":"1234"}] }
	`
)
