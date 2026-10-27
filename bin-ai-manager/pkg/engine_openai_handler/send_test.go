package engine_openai_handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

func Test_Send_success(t *testing.T) {
	var gotPath string
	var gotReq openai.ChatCompletionRequest

	h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		writeJSON(w, http.StatusOK, `{"id":"cmpl-3","choices":[{"index":0,"message":{"role":"assistant","content":"sent ok"}}]}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.Send(ctx, &openai.ChatCompletionRequest{
		Model:    "send-model",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "cmpl-3" || len(res.Choices) != 1 || res.Choices[0].Message.Content != "sent ok" {
		t.Errorf("wrong response: %+v", res)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("wrong path: %s", gotPath)
	}
	if gotReq.Model != "send-model" || len(gotReq.Messages) != 1 || gotReq.Messages[0].Content != "ping" {
		t.Errorf("wrong request body: %+v", gotReq)
	}
}
