package engine_openai_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"

	"github.com/sashabaranov/go-openai"
)

// sseHandler writes the given data payloads as SSE events, flushes and returns.
func sseHandler(payloads ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, p := range payloads {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", p)
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func chunkContent(s string) string {
	b, _ := json.Marshal(s)
	return `{"choices":[{"index":0,"delta":{"content":` + string(b) + `}}]}`
}

func chunkFinish(reason string) string {
	return `{"choices":[{"index":0,"delta":{},"finish_reason":"` + reason + `"}]}`
}

func chunkToolStart(id, name string) string {
	return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"` + id + `","type":"function","function":{"name":"` + name + `","arguments":""}}]}}]}`
}

func chunkToolArgs(args string) string {
	b, _ := json.Marshal(args)
	return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":` + string(b) + `}}]}}]}`
}

// drainStream reads both channels until they are closed and fails on timeout.
func drainStream(t *testing.T, chanMsg <-chan string, chanTool <-chan *message.ToolCall) ([]string, []*message.ToolCall) {
	t.Helper()

	var texts []string
	var tcs []*message.ToolCall
	msgOpen, toolOpen := true, true
	timeout := time.After(10 * time.Second)

	for msgOpen || toolOpen {
		select {
		case s, ok := <-chanMsg:
			if !ok {
				msgOpen = false
				chanMsg = nil
				continue
			}
			texts = append(texts, s)
		case tc, ok := <-chanTool:
			if !ok {
				toolOpen = false
				chanTool = nil
				continue
			}
			tcs = append(tcs, tc)
		case <-timeout:
			t.Fatalf("timed out waiting for channels to close")
		}
	}
	return texts, tcs
}

func Test_StreamingSend_text(t *testing.T) {
	tests := []struct {
		name     string
		payloads []string
		expect   []string
	}{
		{
			name: "sentence_split_on_punctuation",
			payloads: []string{
				chunkContent("Hello"), chunkContent(" world."), chunkContent(" How"), chunkContent(" are you?"),
				chunkContent(" fine"), chunkFinish("stop"),
			},
			expect: []string{"Hello world.", "How are you?", "fine"},
		},
		{
			name:     "newline_splits",
			payloads: []string{chunkContent("line one\n"), chunkContent("line two"), "[DONE]"},
			expect:   []string{"line one", "line two"},
		},
		{
			name:     "eof_without_done_flushes_remaining",
			payloads: []string{chunkContent("partial text")},
			expect:   []string{"partial text"},
		},
		{
			name:     "empty_content_chunks_produce_nothing",
			payloads: []string{chunkContent(""), chunkFinish("stop")},
			expect:   nil,
		},
		{
			name:     "no_choices_chunk_is_ignored",
			payloads: []string{`{"choices":[]}`, chunkContent("after."), "[DONE]"},
			expect:   []string{"after."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestEngine(t, sseHandler(tt.payloads...))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, []*message.Message{{Role: message.RoleUser, Content: "q"}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			texts, tcs := drainStream(t, chanMsg, chanTool)
			if len(tcs) != 0 {
				t.Errorf("unexpected tool calls: %+v", tcs)
			}
			if len(texts) != len(tt.expect) {
				t.Fatalf("wrong text count. expect: %q, got: %q", tt.expect, texts)
			}
			for i := range texts {
				if texts[i] != tt.expect[i] {
					t.Errorf("wrong text[%d]. expect: %q, got: %q", i, tt.expect[i], texts[i])
				}
			}
		})
	}
}

func Test_StreamingSend_tool_calls(t *testing.T) {
	tests := []struct {
		name         string
		payloads     []string
		expectTexts  []string
		expectToolID []string
	}{
		{
			name: "single_tool_call_then_finish",
			payloads: []string{
				chunkToolStart("call-1", "connect"), chunkToolArgs(`{"destinations":`), chunkToolArgs(`[]}`),
				chunkFinish("tool_calls"),
			},
			expectToolID: []string{"call-1"},
		},
		{
			name: "two_tool_calls_in_sequence",
			payloads: []string{
				chunkToolStart("call-a", "connect"), chunkToolArgs(`{}`),
				chunkToolStart("call-b", "message_send"), chunkToolArgs(`{}`),
				chunkFinish("tool_calls"),
			},
			expectToolID: []string{"call-a", "call-b"},
		},
		{
			name: "content_after_tool_flushes_tool_first",
			payloads: []string{
				chunkToolStart("call-c", "connect"), chunkToolArgs(`{}`),
				chunkContent("Done."), chunkFinish("stop"),
			},
			expectTexts:  []string{"Done."},
			expectToolID: []string{"call-c"},
		},
		{
			name: "arguments_without_name_emit_no_tool",
			payloads: []string{
				chunkToolArgs(`{}`), chunkFinish("stop"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestEngine(t, sseHandler(tt.payloads...))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			texts, tcs := drainStream(t, chanMsg, chanTool)
			if len(texts) != len(tt.expectTexts) {
				t.Fatalf("wrong texts. expect: %q, got: %q", tt.expectTexts, texts)
			}
			for i := range texts {
				if texts[i] != tt.expectTexts[i] {
					t.Errorf("wrong text[%d]. expect: %q, got: %q", i, tt.expectTexts[i], texts[i])
				}
			}
			if len(tcs) != len(tt.expectToolID) {
				t.Fatalf("wrong tool count. expect: %d, got: %d", len(tt.expectToolID), len(tcs))
			}
			for i, tc := range tcs {
				if tc.ID != tt.expectToolID[i] || tc.Type != message.ToolTypeFunction {
					t.Errorf("wrong tool[%d]. expect id: %s, got: %+v", i, tt.expectToolID[i], tc)
				}
			}
		})
	}
}

func Test_StreamingSend_request(t *testing.T) {
	var gotReq openai.ChatCompletionRequest
	var gotPath string

	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		sseHandler(chunkFinish("stop"))(w, r)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cc := &aicall.AIcall{AIEngineModel: "openai.gpt-5.2"}
	chanMsg, chanTool, err := h.StreamingSend(ctx, cc, []*message.Message{
		{Role: message.RoleUser, Content: "first"},
		{Role: message.RoleAssistant, Content: "second"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	drainStream(t, chanMsg, chanTool)

	if gotPath != "/chat/completions" {
		t.Errorf("wrong path: %s", gotPath)
	}
	if gotReq.Model != "gpt-5.2" || !gotReq.Stream {
		t.Errorf("wrong model/stream: %s/%v", gotReq.Model, gotReq.Stream)
	}
	if len(gotReq.Messages) != 3 {
		t.Fatalf("wrong message count: %d", len(gotReq.Messages))
	}
	if gotReq.Messages[0].Role != "system" || gotReq.Messages[0].Content != defaultSystemPrompt {
		t.Errorf("first message is not the default system prompt: %+v", gotReq.Messages[0])
	}
	if gotReq.Messages[1].Content != "first" || gotReq.Messages[1].Role != "user" ||
		gotReq.Messages[2].Content != "second" || gotReq.Messages[2].Role != "assistant" {
		t.Errorf("wrong forwarded messages: %+v", gotReq.Messages[1:])
	}
	if len(gotReq.Tools) != 2 || gotReq.Tools[0].Function.Name != "connect" || gotReq.Tools[1].Function.Name != "message_send" {
		t.Errorf("wrong tools: %+v", gotReq.Tools)
	}
}

func Test_StreamingSend_default_model(t *testing.T) {
	var gotReq openai.ChatCompletionRequest
	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		sseHandler(chunkFinish("stop"))(w, r)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	drainStream(t, chanMsg, chanTool)

	if gotReq.Model != defaultModel {
		t.Errorf("wrong model. expect: %s, got: %s", defaultModel, gotReq.Model)
	}
}

func Test_StreamingSend_error_status(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"bad key","type":"invalid_request_error"}}`},
		{"server_error", http.StatusInternalServerError, `{"error":{"message":"boom","type":"server_error"}}`},
		{"non_json_body", http.StatusBadGateway, `not json`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, tt.status, tt.body)
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
			if err == nil {
				t.Fatalf("expected error")
			}
			if chanMsg != nil || chanTool != nil {
				t.Errorf("expected nil channels on error")
			}

			var apiErr *openai.APIError
			var reqErr *openai.RequestError
			switch {
			case errors.As(err, &apiErr):
				if apiErr.HTTPStatusCode != tt.status {
					t.Errorf("wrong status. expect: %d, got: %d", tt.status, apiErr.HTTPStatusCode)
				}
			case errors.As(err, &reqErr):
				if reqErr.HTTPStatusCode != tt.status {
					t.Errorf("wrong status. expect: %d, got: %d", tt.status, reqErr.HTTPStatusCode)
				}
			default:
				t.Errorf("expected an openai API or request error, got: %v", err)
			}
		})
	}
}

func Test_StreamingSend_canceled_context_before_send(t *testing.T) {
	h, _ := newTestEngine(t, sseHandler(chunkContent("never.")))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if chanMsg != nil || chanTool != nil {
		t.Errorf("expected nil channels on error")
	}
}

func Test_StreamingSend_context_canceled_mid_stream(t *testing.T) {
	release := make(chan struct{})
	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancel()
	texts, tcs := drainStream(t, chanMsg, chanTool)
	if len(texts) != 0 || len(tcs) != 0 {
		t.Errorf("expected no output, got texts=%q tools=%+v", texts, tcs)
	}
}

func Test_StreamingSend_malformed_event(t *testing.T) {
	h, _ := newTestEngine(t, sseHandler(chunkContent("before."), `{not json`, chunkContent("after.")))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	texts, tcs := drainStream(t, chanMsg, chanTool)
	if len(tcs) != 0 {
		t.Errorf("unexpected tool calls: %+v", tcs)
	}
	if len(texts) != 1 || texts[0] != "before." {
		t.Errorf("expected only the text before the malformed event, got: %q", texts)
	}
}

func Test_streamingResponseHandleText(t *testing.T) {
	h := &engineOpenaiHandler{}

	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"empty_sends_nothing", "", nil},
		{"trims_whitespace", "  padded text \n", []string{"padded text"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan string, 1)
			var sb strings.Builder
			sb.WriteString(tt.input)
			h.streamingResponseHandleText(ch, sb)
			close(ch)

			var got []string
			for s := range ch {
				got = append(got, s)
			}
			if len(got) != len(tt.expect) || (len(got) == 1 && got[0] != tt.expect[0]) {
				t.Errorf("wrong output. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_streamingResponseHandleTool(t *testing.T) {
	h := &engineOpenaiHandler{}

	tests := []struct {
		name       string
		id         string
		toolName   string
		expectSent bool
	}{
		{"empty_name_sends_nothing", "call-x", "", false},
		{"named_tool_sent", "call-y", "connect", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan *message.ToolCall, 1)
			var sb strings.Builder
			sb.WriteString(`{"k":"v"}`)
			h.streamingResponseHandleTool(ch, tt.id, tt.toolName, sb)
			close(ch)

			tc, ok := <-ch
			if ok != tt.expectSent {
				t.Fatalf("wrong sent state. expect: %v, got: %v", tt.expectSent, ok)
			}
			if ok && (tc.ID != tt.id || tc.Type != message.ToolTypeFunction) {
				t.Errorf("wrong tool call: %+v", tc)
			}
		})
	}
}

func Test_StreamingSend_finish_reason_ends_stream_while_connection_open(t *testing.T) {
	release := make(chan struct{})
	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		sseHandler(chunkContent("complete answer"), chunkFinish("stop"))(w, r)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chanMsg, chanTool, err := h.StreamingSend(ctx, &aicall.AIcall{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	texts, tcs := drainStream(t, chanMsg, chanTool)
	if len(tcs) != 0 || len(texts) != 1 || texts[0] != "complete answer" {
		t.Errorf("wrong output. texts=%q tools=%+v", texts, tcs)
	}
}
