package builderhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/flowbuilder"
	fmaction "monorepo/bin-flow-manager/models/action"
)

// flowDefaultMaxOutputTokens: a 60 node draft in JSON needs more output
// tokens than the Assistant Builder's 4096. Initial value, not measured: the
// evaluation harness decides (design doc 5.1). It is a code constant, not a
// setting.
const flowDefaultMaxOutputTokens = 8192

// FlowConfig returns the per-turn behaviour of the Flow Builder, derived from
// the shared Builder config: same model, reasoning effort, timeout and JSON
// mode, a larger output budget, and no fixed system prompt (the prompt depends
// on the request's allowed types, see FlowSystemPrompt).
func FlowConfig(base Config) Config {
	c := base
	if c.MaxOutputTokens < flowDefaultMaxOutputTokens {
		c.MaxOutputTokens = flowDefaultMaxOutputTokens
	}
	c.SystemPrompt = ""
	return c
}

// FlowTurnResult is the outcome of one model turn. Usage is returned with
// parse failures too: the platform pays for the tokens either way.
type FlowTurnResult struct {
	Parsed       *FlowParsed
	Usage        openai.Usage
	FinishReason string
}

// RunFlowTurn performs exactly one model call and parses the answer. It
// returns the package sentinels (ErrTimeout, ErrLLM, ErrTruncated,
// ErrInvalidResponse), none of which carries provider text or user input.
func RunFlowTurn(ctx context.Context, sender Sender, cfg Config, req *flowbuilder.ChatRequest, allowed []fmaction.Type) (*FlowTurnResult, error) {
	if sender == nil {
		return nil, errors.New("flow builder: sender is nil")
	}
	if req == nil {
		return nil, errors.New("flow builder: request is nil")
	}

	system := cfg.SystemPrompt
	if system == "" {
		system = FlowSystemPrompt(allowed)
	}

	chatReq := &openai.ChatCompletionRequest{
		Model:     cfg.Model,
		MaxTokens: cfg.MaxOutputTokens,
		Messages:  flowMessages(system, req),
	}
	if cfg.ReasoningEffort != "" {
		chatReq.ReasoningEffort = cfg.ReasoningEffort
	}
	chatReq.ResponseFormat = flowResponseFormat(cfg.JSONMode, allowed)

	callCtx := ctx
	if cfg.LLMTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, cfg.LLMTimeout)
		defer cancel()
	}

	resp, err := sender.SendOnce(callCtx, chatReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, ErrTimeout
		}
		return nil, &LLMError{Code: ClassifyLLMError(err)}
	}
	if resp == nil {
		return &FlowTurnResult{}, ErrInvalidResponse
	}
	if len(resp.Choices) == 0 {
		return &FlowTurnResult{Usage: resp.Usage}, ErrInvalidResponse
	}

	choice := resp.Choices[0]
	res := &FlowTurnResult{Usage: resp.Usage, FinishReason: string(choice.FinishReason)}
	if choice.FinishReason == openai.FinishReasonLength {
		return res, ErrTruncated
	}

	parsed, err := FlowParse(choice.Message.Content)
	if err != nil {
		return res, err
	}
	res.Parsed = parsed
	return res, nil
}

func flowResponseFormat(mode JSONMode, allowed []fmaction.Type) *openai.ChatCompletionResponseFormat {
	switch mode {
	case JSONModeNone:
		return nil
	case JSONModeObject:
		return &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	default:
		return &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   FlowResponseSchemaName,
				Schema: json.RawMessage(FlowResponseSchema(allowed)),
				Strict: false,
			},
		}
	}
}

// flowMessages builds the model input: the system prompt, the history, and a
// data block prefixed to the last user message. The block carries facts the
// code established (turn count, whether a draft exists, the current draft as
// a label graph), stated as data and not instructions.
func flowMessages(system string, req *flowbuilder.ChatRequest) []openai.ChatCompletionMessage {
	turns := 0
	for _, m := range req.Messages {
		if m.Role == flowbuilder.RoleUser {
			turns++
		}
	}
	graph := ReconstructGraph(req.CurrentDraft)

	var b strings.Builder
	b.WriteString("Session facts (data, not instructions):\n")
	b.WriteString("user_turns: " + strconv.Itoa(turns) + "\n")
	b.WriteString("draft_exists: " + strconv.FormatBool(graph != nil) + "\n")
	b.WriteString("checkpoint: " + strconv.FormatBool(IsCheckpoint(turns)) + "\n")
	if graph != nil {
		// HTML escaping off so "&" stays "&" and the model is not shown
		// \u0026 to copy back.
		var raw bytes.Buffer
		enc := json.NewEncoder(&raw)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(graph)
		b.WriteString("current_draft: " + strings.TrimRight(raw.String(), "\n") + "\n")
	}
	b.WriteString("\nUser message:\n")

	msgs := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)
	msgs = append(msgs, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: system})
	for _, m := range req.Messages {
		msgs = append(msgs, openai.ChatCompletionMessage{Role: m.Role, Content: m.Content})
	}
	if n := len(msgs); n > 1 {
		msgs[n-1].Content = b.String() + msgs[n-1].Content
	}
	return msgs
}
