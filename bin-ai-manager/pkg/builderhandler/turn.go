package builderhandler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/builder"
)

// Sender is the single engine method the Builder needs. The shared
// engine_openai_handler satisfies it as-is, and the evaluation's fake engines
// and user simulator implement just this one method.
type Sender interface {
	SendOnce(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
}

// TurnResult is the outcome of one model turn. Usage and FinishReason are
// returned alongside parse failures too, because the platform pays for tokens
// whether or not the answer is usable.
type TurnResult struct {
	Parsed       *ParsedResponse
	Usage        openai.Usage
	FinishReason string
}

// RunTurn performs exactly one model call and parses the answer.
//
// It is a package function, not part of the BuilderHandler interface: it is
// the shared path that both production (Chat) and the evaluation harness call,
// so the evaluation measures the production code. Variants are expressed by
// copying cfg and changing an evaluation-override field.
//
// RunTurn applies cfg.LLMTimeout itself and never retries. Returned errors are
// the package sentinels (ErrTimeout, ErrLLM, ErrTruncated, ErrInvalidResponse)
// and carry no provider text or user input, because a provider 4xx can echo the
// prompt back.
func RunTurn(ctx context.Context, sender Sender, cfg Config, req *builder.ChatRequest) (*TurnResult, error) {
	if sender == nil {
		return nil, errors.New("builder: sender is nil")
	}
	if req == nil {
		return nil, errors.New("builder: request is nil")
	}

	chatReq := &openai.ChatCompletionRequest{
		Model:     cfg.Model,
		MaxTokens: cfg.MaxOutputTokens,
		Messages:  turnMessages(cfg, req),
	}
	if cfg.ReasoningEffort != "" {
		chatReq.ReasoningEffort = cfg.ReasoningEffort
	}
	chatReq.ResponseFormat = responseFormat(cfg.JSONMode)

	callCtx := ctx
	if cfg.LLMTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, cfg.LLMTimeout)
		defer cancel()
	}

	resp, err := sender.SendOnce(callCtx, chatReq)
	if err != nil {
		// The deadline we set is a timeout. A cancelled caller is not.
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, ErrTimeout
		}
		return nil, &LLMError{Code: ClassifyLLMError(err)}
	}
	if resp == nil {
		return &TurnResult{}, ErrInvalidResponse
	}
	if len(resp.Choices) == 0 {
		// The platform was still billed for the tokens.
		return &TurnResult{Usage: resp.Usage}, ErrInvalidResponse
	}

	choice := resp.Choices[0]
	res := &TurnResult{Usage: resp.Usage, FinishReason: string(choice.FinishReason)}

	// A length cut means the answer is incomplete even if a prefix happens to
	// parse, so it is never trusted.
	if choice.FinishReason == openai.FinishReasonLength {
		return res, ErrTruncated
	}

	parsed, err := Parse(choice.Message.Content)
	if err != nil {
		return res, err
	}
	res.Parsed = parsed
	return res, nil
}

// turnMessages builds the model input for cfg. Without DataBlockInSystem the
// block prefixes the last user message (BuildChatMessages). With it, the same
// block, built by the same function, is appended to the system prompt.
func turnMessages(cfg Config, req *builder.ChatRequest) []openai.ChatCompletionMessage {
	if !cfg.DataBlockInSystem {
		return BuildChatMessages(cfg.SystemPrompt, req)
	}
	block, hist := buildParts(req)
	system := cfg.SystemPrompt + "\n\n" + block
	return append([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleSystem, Content: system}}, hist...)
}

func responseFormat(mode JSONMode) *openai.ChatCompletionResponseFormat {
	switch mode {
	case JSONModeNone:
		return nil
	case JSONModeObject:
		return &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	default:
		return &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   ResponseSchemaName,
				Schema: json.RawMessage(ResponseSchema),
				// Strict json_schema is not supported by the Gemini
				// OpenAI-compatible endpoint; Parse validates instead.
				Strict: false,
			},
		}
	}
}

// ClassifyLLMError reduces an engine error to a short fixed code. It never
// returns any part of the error text, which may contain the prompt.
func ClassifyLLMError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	status := 0
	var apiErr *openai.APIError
	var reqErr *openai.RequestError
	switch {
	case errors.As(err, &apiErr):
		status = apiErr.HTTPStatusCode
	case errors.As(err, &reqErr):
		status = reqErr.HTTPStatusCode
	}
	switch {
	case status == 401 || status == 403:
		return "auth"
	case status == 429:
		return "rate_limit"
	case status >= 500:
		return "provider_5xx"
	case status >= 400:
		return "provider_4xx"
	default:
		return "other"
	}
}
