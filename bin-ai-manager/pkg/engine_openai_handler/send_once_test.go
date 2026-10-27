package engine_openai_handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// newTestEngine builds a handler bound to a loopback test server.
func newTestEngine(t *testing.T, fn http.HandlerFunc) (*engineOpenaiHandler, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(fn)
	t.Cleanup(srv.Close)

	key := "test-key-synthetic"
	h, ok := NewEngineOpenaiHandlerWithConfig(key, srv.URL).(*engineOpenaiHandler)
	if !ok {
		t.Fatalf("unexpected handler type")
	}
	return h, srv
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func Test_SendOnce_success(t *testing.T) {
	var gotPath, gotMethod string
	var gotReq openai.ChatCompletionRequest

	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		writeJSON(w, http.StatusOK, `{"id":"cmpl-1","choices":[{"index":0,"message":{"role":"assistant","content":"hello there"},"finish_reason":"stop"}]}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &openai.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}},
	}
	res, err := h.SendOnce(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Choices) != 1 || res.Choices[0].Message.Content != "hello there" || res.Choices[0].Message.Role != "assistant" {
		t.Errorf("wrong response: %+v", res)
	}
	if gotMethod != http.MethodPost || gotPath != "/chat/completions" {
		t.Errorf("wrong request line: %s %s", gotMethod, gotPath)
	}
	if gotReq.Model != "test-model" || len(gotReq.Messages) != 1 || gotReq.Messages[0].Content != "hi" {
		t.Errorf("wrong request body: %+v", gotReq)
	}
}

func Test_SendOnce_empty_choices(t *testing.T) {
	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"cmpl-2","choices":[]}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.SendOnce(ctx, &openai.ChatCompletionRequest{Model: "m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.ID != "cmpl-2" || len(res.Choices) != 0 {
		t.Errorf("wrong response: %+v", res)
	}
}

func Test_SendOnce_error_status(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"bad key","type":"invalid_request_error"}}`},
		{"bad_request", http.StatusBadRequest, `{"error":{"message":"bad input","type":"invalid_request_error"}}`},
		{"server_error", http.StatusInternalServerError, `{"error":{"message":"boom","type":"server_error"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, tt.status, tt.body)
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			res, err := h.SendOnce(ctx, &openai.ChatCompletionRequest{Model: "m"})
			if err == nil {
				t.Fatalf("expected error, got response: %+v", res)
			}
			if res != nil {
				t.Errorf("expected nil response, got %+v", res)
			}
			var apiErr *openai.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %T: %v", err, err)
			}
			if apiErr.HTTPStatusCode != tt.status {
				t.Errorf("wrong status. expect: %d, got: %d", tt.status, apiErr.HTTPStatusCode)
			}
		})
	}
}

func Test_SendOnce_invalid_json(t *testing.T) {
	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"choices": [`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.SendOnce(ctx, &openai.ChatCompletionRequest{Model: "m"})
	if err == nil || res != nil {
		t.Errorf("expected error and nil response, got res=%+v err=%v", res, err)
	}
}

func Test_SendOnce_context_deadline(t *testing.T) {
	release := make(chan struct{})
	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	res, err := h.SendOnce(ctx, &openai.ChatCompletionRequest{Model: "m"})
	if err == nil || res != nil {
		t.Fatalf("expected error and nil response, got res=%+v err=%v", res, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected deadline exceeded, got: %v", err)
	}
}
