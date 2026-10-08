package builderhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
// the shared Builder config: same model, reasoning effort and timeout, a
// larger output budget, and no fixed system prompt (the prompt depends on the
// request's allowed types, see FlowSystemPrompt).
//
// The response format is always JSONModeObject, whatever the shared config
// says (VOIP-1577). With json_schema the provider returned no draft and often
// generated until the output cap, which takes longer than the call timeout;
// with json_object the same model, prompt and parser returned a draft in a few
// seconds (measurements: docs/plans/2026-10-08-flow-builder-missing-draft.md).
// The prompt describes the answer shape, and FlowParse and AssembleFlowDraft
// validate every part of it, so nothing relies on a schema. A mode set on the
// shared config is ignored on purpose.
func FlowConfig(base Config) Config {
	c := base
	if c.MaxOutputTokens < flowDefaultMaxOutputTokens {
		c.MaxOutputTokens = flowDefaultMaxOutputTokens
	}
	c.SystemPrompt = ""
	c.JSONMode = JSONModeObject
	return c
}

// FlowTurnResult is the outcome of one model turn. Usage is returned with
// parse failures too: the platform pays for the tokens either way.
type FlowTurnResult struct {
	Parsed       *FlowParsed
	Usage        openai.Usage
	FinishReason string

	// Diag holds sizes and timings for the diagnostic log line. It carries
	// counts, durations and fixed classes only, never any text (VOIP-1576).
	Diag FlowTurnDiag
}

// The fixed values of FlowTurnDiag.InvalidKind, one per place that returns
// ErrInvalidResponse.
const (
	invalidKindNilResponse = "nil_response"
	invalidKindNoChoices   = "no_choices"
	invalidKindUnparsable  = "unparsable"
)

// FlowTurnDiag is what RunFlowTurn measured about one model call. Every field
// is a count, a size, a duration, a flag or a fixed class; none holds the
// customer's text, the model's answer, the prompt or the schema.
type FlowTurnDiag struct {
	Elapsed             time.Duration // the SendOnce call only
	BuildElapsed        time.Duration // prompt, schema and message building
	SystemChars         int           // runes of the system prompt
	RequestChars        int           // runes of all message contents sent
	SchemaBytes         int           // bytes of the JSON schema text, 0 unless the schema mode is used
	ResponseChars       int           // runes of the answer content, 0 when none
	UserTurns           int
	HistoryMessages     int
	AllowedTypes        int
	CurrentDraftPresent bool
	InvalidKind         string // set only where ErrInvalidResponse is returned
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

	buildStart := time.Now()

	system := cfg.SystemPrompt
	if system == "" {
		system = FlowSystemPrompt(allowed)
	}

	// The schema text is built once and only for the mode that sends it, so
	// SchemaBytes matches what goes out.
	schema := ""
	if flowUsesSchema(cfg.JSONMode) {
		schema = FlowResponseSchema(allowed)
	}

	chatReq := &openai.ChatCompletionRequest{
		Model:     cfg.Model,
		MaxTokens: cfg.MaxOutputTokens,
		Messages:  flowMessages(system, req),
	}
	if cfg.ReasoningEffort != "" {
		chatReq.ReasoningEffort = cfg.ReasoningEffort
	}
	chatReq.ResponseFormat = flowResponseFormat(cfg.JSONMode, schema)

	res := &FlowTurnResult{Diag: FlowTurnDiag{
		SystemChars:         utf8.RuneCountInString(system),
		SchemaBytes:         len(schema),
		UserTurns:           flowUserTurns(req),
		HistoryMessages:     len(req.Messages),
		AllowedTypes:        len(allowed),
		CurrentDraftPresent: req.CurrentDraft != nil,
	}}
	for _, m := range chatReq.Messages {
		res.Diag.RequestChars += utf8.RuneCountInString(m.Content)
	}

	callCtx := ctx
	if cfg.LLMTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, cfg.LLMTimeout)
		defer cancel()
	}

	callStart := time.Now()
	res.Diag.BuildElapsed = callStart.Sub(buildStart)
	resp, err := sender.SendOnce(callCtx, chatReq)
	res.Diag.Elapsed = time.Since(callStart)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return res, ErrTimeout
		}
		return res, &LLMError{Code: ClassifyLLMError(err)}
	}
	if resp == nil {
		res.Diag.InvalidKind = invalidKindNilResponse
		return res, ErrInvalidResponse
	}
	res.Usage = resp.Usage
	if len(resp.Choices) == 0 {
		res.Diag.InvalidKind = invalidKindNoChoices
		return res, ErrInvalidResponse
	}

	choice := resp.Choices[0]
	res.FinishReason = string(choice.FinishReason)
	res.Diag.ResponseChars = utf8.RuneCountInString(choice.Message.Content)
	if choice.FinishReason == openai.FinishReasonLength {
		return res, ErrTruncated
	}

	parsed, err := FlowParse(choice.Message.Content)
	if err != nil {
		res.Diag.InvalidKind = invalidKindUnparsable
		return res, err
	}
	res.Parsed = parsed
	return res, nil
}

// flowUsesSchema reports whether the mode sends the JSON schema. It is the
// complement of the two explicit cases in flowResponseFormat, so an empty or
// unknown mode also sends the schema, exactly as before.
func flowUsesSchema(mode JSONMode) bool {
	return mode != JSONModeNone && mode != JSONModeObject
}

// flowJSONModeName maps the mode to a fixed string that matches what is sent.
func flowJSONModeName(mode JSONMode) string {
	switch mode {
	case JSONModeNone:
		return "none"
	case JSONModeObject:
		return "object"
	default:
		return "schema"
	}
}

// flowUserTurns counts the user messages of the request.
func flowUserTurns(req *flowbuilder.ChatRequest) int {
	turns := 0
	for _, m := range req.Messages {
		if m.Role == flowbuilder.RoleUser {
			turns++
		}
	}
	return turns
}

func flowResponseFormat(mode JSONMode, schema string) *openai.ChatCompletionResponseFormat {
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
				Schema: json.RawMessage(schema),
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
