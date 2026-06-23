package analysishandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/analysis"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
)

func newTestHandler(mockEngine engine_openai_handler.EngineOpenaiHandler) *analysisHandler {
	h := NewAnalysisHandler(
		mockEngine,
		"gemini-2.5-flash",
		[]string{"gemini-2.5-flash", "gemini-2.5-pro"},
		1024,
		2048,
	)
	return h.(*analysisHandler)
}

func Test_Run_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockEngine := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
	h := newTestHandler(mockEngine)

	req := &analysis.Request{
		Prompt:     "analyze",
		Data:       json.RawMessage(`{"k":"v"}`),
		Schema:     json.RawMessage(`{"type":"object"}`),
		SchemaName: "verdict",
		Model:      "gemini-2.5-pro",
	}

	mockEngine.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, chatReq *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
			// the requested (allowed) model must be used
			if chatReq.Model != "gemini-2.5-pro" {
				t.Errorf("wrong model. expected: gemini-2.5-pro, got: %s", chatReq.Model)
			}
			// max tokens must be set
			if chatReq.MaxTokens != 2048 {
				t.Errorf("wrong max tokens. expected: 2048, got: %d", chatReq.MaxTokens)
			}
			// response format must be json_schema (Strict is false for Gemini compat)
			if chatReq.ResponseFormat == nil || chatReq.ResponseFormat.Type != openai.ChatCompletionResponseFormatTypeJSONSchema {
				t.Errorf("response format not json_schema")
			}
			if chatReq.ResponseFormat.JSONSchema == nil || chatReq.ResponseFormat.JSONSchema.Strict {
				t.Errorf("json schema strict must be false for Gemini compat")
			}
			if chatReq.ResponseFormat.JSONSchema.Name != "verdict" {
				t.Errorf("wrong schema name. got: %s", chatReq.ResponseFormat.JSONSchema.Name)
			}
			return &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message:      openai.ChatCompletionMessage{Content: `{"ok":true}`},
						FinishReason: openai.FinishReasonStop,
					},
				},
				Usage: openai.Usage{PromptTokens: 11, CompletionTokens: 22},
			}, nil
		},
	)

	res, err := h.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(res.Result) != `{"ok":true}` {
		t.Errorf("wrong result. got: %s", res.Result)
	}
	if res.Model != "gemini-2.5-pro" {
		t.Errorf("wrong model. got: %s", res.Model)
	}
	if res.FinishReason != "stop" {
		t.Errorf("wrong finish reason. got: %s", res.FinishReason)
	}
	if res.Truncated {
		t.Errorf("expected not truncated")
	}
	if res.PromptTokens != 11 || res.OutputTokens != 22 {
		t.Errorf("wrong token counts. got: %d/%d", res.PromptTokens, res.OutputTokens)
	}
}

func Test_Run_modelFallbackToDefault(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockEngine := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
	h := newTestHandler(mockEngine)

	req := &analysis.Request{
		Prompt:     "analyze",
		Schema:     json.RawMessage(`{"type":"object"}`),
		SchemaName: "verdict",
		Model:      "not-allowed-model",
	}

	mockEngine.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, chatReq *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
			if chatReq.Model != "gemini-2.5-flash" {
				t.Errorf("expected fallback to default gemini-2.5-flash, got: %s", chatReq.Model)
			}
			return &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{Message: openai.ChatCompletionMessage{Content: `{}`}, FinishReason: openai.FinishReasonStop},
				},
			}, nil
		},
	)

	res, err := h.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Model != "gemini-2.5-flash" {
		t.Errorf("wrong model. got: %s", res.Model)
	}
}

func Test_Run_truncatedOutput(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockEngine := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
	h := newTestHandler(mockEngine)

	req := &analysis.Request{
		Schema:     json.RawMessage(`{"type":"object"}`),
		SchemaName: "verdict",
	}

	mockEngine.EXPECT().Send(gomock.Any(), gomock.Any()).Return(
		&openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: `{"partial":`}, FinishReason: openai.FinishReasonLength},
			},
		}, nil,
	)

	res, err := h.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Truncated || res.FinishReason != "length" {
		t.Errorf("expected truncated/length. got: %v / %s", res.Truncated, res.FinishReason)
	}
}

func Test_Run_validationErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockEngine := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
	h := newTestHandler(mockEngine)
	// no Send expected on any of these — strict mock fails if called.

	tests := []struct {
		name string
		req  *analysis.Request
	}{
		{"nil request", nil},
		{"missing schema", &analysis.Request{SchemaName: "x"}},
		{"missing schema name", &analysis.Request{Schema: json.RawMessage(`{}`)}},
		{
			"input too large",
			&analysis.Request{
				Schema:     json.RawMessage(`{}`),
				SchemaName: "x",
				Prompt:     string(make([]byte, 2048)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.Run(context.Background(), tt.req)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func Test_Run_engineError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockEngine := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
	h := newTestHandler(mockEngine)

	req := &analysis.Request{
		Schema:     json.RawMessage(`{"type":"object"}`),
		SchemaName: "verdict",
	}

	mockEngine.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("boom"))

	_, err := h.Run(context.Background(), req)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func Test_Run_noChoices(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockEngine := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
	h := newTestHandler(mockEngine)

	req := &analysis.Request{
		Schema:     json.RawMessage(`{"type":"object"}`),
		SchemaName: "verdict",
	}

	mockEngine.EXPECT().Send(gomock.Any(), gomock.Any()).Return(&openai.ChatCompletionResponse{Choices: nil}, nil)

	_, err := h.Run(context.Background(), req)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
