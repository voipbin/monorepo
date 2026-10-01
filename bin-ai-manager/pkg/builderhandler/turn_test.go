package builderhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/builder"
)

// fakeSender is a hand-written Sender. It records the request it was given.
type fakeSender struct {
	resp *openai.ChatCompletionResponse
	err  error
	got  *openai.ChatCompletionRequest
	wait time.Duration
}

func (f *fakeSender) SendOnce(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	f.got = req
	if f.wait > 0 {
		select {
		case <-time.After(f.wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func reply(content string, finish openai.FinishReason) *openai.ChatCompletionResponse {
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Role: "assistant", Content: content}, FinishReason: finish}},
		Usage:   openai.Usage{PromptTokens: 11, CompletionTokens: 7, TotalTokens: 18},
	}
}

func baseReq() *builder.ChatRequest {
	return &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleUser, Content: "고객센터 봇을 만들고 싶어요"}}}
}

func Test_RunTurn_ok(t *testing.T) {
	f := &fakeSender{resp: reply(`{"message":"어떤 채널인가요?"}`, openai.FinishReasonStop)}
	res, err := RunTurn(context.Background(), f, DefaultConfig(), baseReq())
	if err != nil {
		t.Fatal(err)
	}
	if res.Parsed.Message != "어떤 채널인가요?" {
		t.Fatalf("message: %q", res.Parsed.Message)
	}
	if res.Usage.PromptTokens != 11 || res.Usage.CompletionTokens != 7 {
		t.Fatalf("usage not carried: %+v", res.Usage)
	}
	if res.FinishReason != "stop" {
		t.Fatalf("finish reason: %q", res.FinishReason)
	}
}

func Test_RunTurn_requestShape(t *testing.T) {
	f := &fakeSender{resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
	cfg := DefaultConfig()
	cfg.Model = "model-x"
	cfg.ReasoningEffort = "none"
	cfg.MaxOutputTokens = 1234
	if _, err := RunTurn(context.Background(), f, cfg, baseReq()); err != nil {
		t.Fatal(err)
	}
	r := f.got
	if r.Model != "model-x" || r.ReasoningEffort != "none" || r.MaxTokens != 1234 {
		t.Fatalf("config not applied: %+v", r)
	}
	if r.Messages[0].Role != openai.ChatMessageRoleSystem || r.Messages[0].Content != cfg.SystemPrompt {
		t.Fatal("system prompt must be first")
	}
	if r.ResponseFormat == nil || r.ResponseFormat.Type != openai.ChatCompletionResponseFormatTypeJSONSchema {
		t.Fatalf("default JSON mode is json_schema: %+v", r.ResponseFormat)
	}
	if r.ResponseFormat.JSONSchema.Strict {
		t.Fatal("Gemini compat endpoint does not support strict mode")
	}
	if r.ResponseFormat.JSONSchema.Name != ResponseSchemaName {
		t.Fatalf("schema name: %q", r.ResponseFormat.JSONSchema.Name)
	}
}

func Test_RunTurn_reasoningEffortEmptyOmitsField(t *testing.T) {
	f := &fakeSender{resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
	cfg := DefaultConfig()
	cfg.ReasoningEffort = ""
	if _, err := RunTurn(context.Background(), f, cfg, baseReq()); err != nil {
		t.Fatal(err)
	}
	if f.got.ReasoningEffort != "" {
		t.Fatalf("empty config must omit the field, got %q", f.got.ReasoningEffort)
	}
}

func Test_RunTurn_jsonModes(t *testing.T) {
	for mode, check := range map[JSONMode]func(*openai.ChatCompletionRequest) bool{
		JSONModeSchema: func(r *openai.ChatCompletionRequest) bool {
			return r.ResponseFormat != nil && r.ResponseFormat.Type == openai.ChatCompletionResponseFormatTypeJSONSchema
		},
		JSONModeObject: func(r *openai.ChatCompletionRequest) bool {
			return r.ResponseFormat != nil && r.ResponseFormat.Type == openai.ChatCompletionResponseFormatTypeJSONObject && r.ResponseFormat.JSONSchema == nil
		},
		JSONModeNone: func(r *openai.ChatCompletionRequest) bool { return r.ResponseFormat == nil },
	} {
		f := &fakeSender{resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
		cfg := DefaultConfig()
		cfg.JSONMode = mode
		if _, err := RunTurn(context.Background(), f, cfg, baseReq()); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if !check(f.got) {
			t.Errorf("mode %q produced %+v", mode, f.got.ResponseFormat)
		}
	}
}

func Test_RunTurn_systemPromptOverride(t *testing.T) {
	f := &fakeSender{resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
	cfg := DefaultConfig()
	cfg.SystemPrompt = "OVERRIDE"
	if _, err := RunTurn(context.Background(), f, cfg, baseReq()); err != nil {
		t.Fatal(err)
	}
	if f.got.Messages[0].Content != "OVERRIDE" {
		t.Fatalf("override ignored: %q", f.got.Messages[0].Content)
	}
}

// The evaluation compares the data block in the user message with the same
// block in the system message. Content must be identical, only the place moves.
func Test_RunTurn_dataBlockPlacement(t *testing.T) {
	req := baseReq()
	req.CurrentDraft = &builder.Draft{Name: "A", InitPrompt: "# A"}
	block, _ := buildParts(req)

	f := &fakeSender{resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
	cfg := DefaultConfig()
	cfg.SystemPrompt = "SYS"
	cfg.DataBlockInSystem = false
	if _, err := RunTurn(context.Background(), f, cfg, req); err != nil {
		t.Fatal(err)
	}
	last := f.got.Messages[len(f.got.Messages)-1]
	if f.got.Messages[0].Content != "SYS" || !strings.HasPrefix(last.Content, block) {
		t.Fatalf("default: block must prefix the last user message, system untouched\nsys=%q\nlast=%q", f.got.Messages[0].Content, last.Content)
	}

	f = &fakeSender{resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
	cfg.DataBlockInSystem = true
	if _, err := RunTurn(context.Background(), f, cfg, req); err != nil {
		t.Fatal(err)
	}
	sys := f.got.Messages[0].Content
	last = f.got.Messages[len(f.got.Messages)-1]
	if !strings.HasPrefix(sys, "SYS") || !strings.Contains(sys, block) {
		t.Fatalf("variant: the same block must be merged into system:\n%q", sys)
	}
	if last.Content != "고객센터 봇을 만들고 싶어요" {
		t.Fatalf("variant: the user message must be untouched, got %q", last.Content)
	}
}

func Test_RunTurn_truncated(t *testing.T) {
	f := &fakeSender{resp: reply(`{"message":"abc`, openai.FinishReasonLength)}
	res, err := RunTurn(context.Background(), f, DefaultConfig(), baseReq())
	if !errors.Is(err, ErrTruncated) {
		t.Fatalf("want ErrTruncated, got %v", err)
	}
	if res == nil || res.Usage.CompletionTokens != 7 {
		t.Fatal("usage must still be reported when truncated (the platform paid for it)")
	}
}

func Test_RunTurn_truncatedEvenIfJSONParses(t *testing.T) {
	// finish_reason=length means the model was cut off; treat it as truncated
	// even when a prefix happens to be valid JSON.
	f := &fakeSender{resp: reply(`{"message":"complete looking"}`, openai.FinishReasonLength)}
	if _, err := RunTurn(context.Background(), f, DefaultConfig(), baseReq()); !errors.Is(err, ErrTruncated) {
		t.Fatalf("want ErrTruncated, got %v", err)
	}
}

func Test_RunTurn_timeout(t *testing.T) {
	f := &fakeSender{wait: time.Second, resp: reply(`{"message":"x"}`, openai.FinishReasonStop)}
	cfg := DefaultConfig()
	cfg.LLMTimeout = 20 * time.Millisecond
	start := time.Now()
	_, err := RunTurn(context.Background(), f, cfg, baseReq())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("RunTurn must apply cfg.LLMTimeout itself")
	}
}

func Test_RunTurn_callerCancelIsNotTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeSender{wait: time.Second}
	go func() { time.Sleep(10 * time.Millisecond); cancel() }()
	_, err := RunTurn(ctx, f, DefaultConfig(), baseReq())
	if errors.Is(err, ErrTimeout) {
		t.Fatal("a cancelled caller is not an LLM deadline")
	}
	if !errors.Is(err, ErrLLM) {
		t.Fatalf("want ErrLLM, got %v", err)
	}
}

// If the caller's own deadline expires first (for example the RPC timeout
// that api-manager owns), that is not the Builder's LLM deadline and must not
// be reported as ErrTimeout: the platform's LLM-deadline counter would then
// mix two different causes.
func Test_RunTurn_callerDeadlineIsNotLLMTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	f := &fakeSender{wait: time.Second}
	cfg := DefaultConfig()
	cfg.LLMTimeout = time.Minute
	_, err := RunTurn(ctx, f, cfg, baseReq())
	if errors.Is(err, ErrTimeout) {
		t.Fatal("the caller's expired deadline must not be classified as the LLM deadline")
	}
	if !errors.Is(err, ErrLLM) {
		t.Fatalf("want ErrLLM, got %v", err)
	}
}

// A provider 4xx can echo the prompt. The error returned must carry no input.
func Test_RunTurn_engineErrorDoesNotEchoInput(t *testing.T) {
	secret := "SECRET-입력-문자열"
	f := &fakeSender{err: fmt.Errorf("provider said: bad request for %q", secret)}
	req := &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleUser, Content: secret}}}
	_, err := RunTurn(context.Background(), f, DefaultConfig(), req)
	if !errors.Is(err, ErrLLM) {
		t.Fatalf("want ErrLLM, got %v", err)
	}
	if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), "provider said") {
		t.Fatalf("error leaked provider/user text: %v", err)
	}
	if code := ClassifyLLMError(f.err); strings.Contains(code, "SECRET") {
		t.Fatalf("classification leaked text: %q", code)
	}
}

func Test_RunTurn_noChoicesStillReportsUsage(t *testing.T) {
	f := &fakeSender{resp: &openai.ChatCompletionResponse{Usage: openai.Usage{PromptTokens: 9, CompletionTokens: 3, TotalTokens: 12}}}
	res, err := RunTurn(context.Background(), f, DefaultConfig(), baseReq())
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("want ErrInvalidResponse, got %v", err)
	}
	if res == nil || res.Usage.TotalTokens != 12 {
		t.Fatalf("the platform was billed for these tokens: %+v", res)
	}
}

// The classification travels as a typed value, so the caller never parses the
// error text to decide a reason or a metric label.
func Test_RunTurn_llmErrorCarriesTypedCode(t *testing.T) {
	f := &fakeSender{err: &openai.APIError{HTTPStatusCode: 429, Message: "SECRET echo"}}
	_, err := RunTurn(context.Background(), f, DefaultConfig(), baseReq())
	var le *LLMError
	if !errors.Is(err, ErrLLM) || !errors.As(err, &le) || le.Code != "rate_limit" {
		t.Fatalf("want ErrLLM with code rate_limit, got %v", err)
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("leaked: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RunTurn(ctx, &fakeSender{wait: time.Second}, DefaultConfig(), baseReq())
	if !errors.As(err, &le) || le.Code != "canceled" {
		t.Fatalf("a cancelled caller is code canceled, got %v", err)
	}
}

func Test_RunTurn_unparseable(t *testing.T) {
	f := &fakeSender{resp: reply("I'm sorry, I can't help with that.", openai.FinishReasonStop)}
	res, err := RunTurn(context.Background(), f, DefaultConfig(), baseReq())
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("want ErrInvalidResponse, got %v", err)
	}
	if res == nil || res.Usage.TotalTokens != 18 {
		t.Fatal("usage must be reported for an unparseable answer")
	}
	if strings.Contains(err.Error(), "sorry") {
		t.Fatal("error must not echo model output")
	}
}

func Test_RunTurn_nilArgs(t *testing.T) {
	if _, err := RunTurn(context.Background(), nil, DefaultConfig(), baseReq()); err == nil {
		t.Fatal("nil sender must error")
	}
	if _, err := RunTurn(context.Background(), &fakeSender{}, DefaultConfig(), nil); err == nil {
		t.Fatal("nil request must error")
	}
}

func Test_ClassifyLLMError(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "timeout"},
		{context.Canceled, "canceled"},
		{&openai.APIError{HTTPStatusCode: 401}, "auth"},
		{&openai.APIError{HTTPStatusCode: 403}, "auth"},
		{&openai.APIError{HTTPStatusCode: 429}, "rate_limit"},
		{&openai.APIError{HTTPStatusCode: 500}, "provider_5xx"},
		{&openai.APIError{HTTPStatusCode: 503}, "provider_5xx"},
		{&openai.APIError{HTTPStatusCode: 400}, "provider_4xx"},
		{&openai.RequestError{HTTPStatusCode: 429}, "rate_limit"},
		{errors.New("something else"), "other"},
	}
	for _, tt := range tests {
		if got := ClassifyLLMError(tt.err); got != tt.want {
			t.Errorf("%T %v: got %q, want %q", tt.err, tt.err, got, tt.want)
		}
	}
}

func Test_DefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.SystemPrompt != SystemPrompt {
		t.Fatal("the default system prompt must be the production constant")
	}
	if c.JSONMode != JSONModeSchema || c.DataBlockInSystem {
		t.Fatalf("defaults must be the production behaviour: %+v", c)
	}
	if c.LLMTimeout != 40*time.Second || c.MaxOutputTokens != 4096 || c.ReasoningEffort != "none" {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if c.Model == "" {
		t.Fatal("default model must be set")
	}
}
