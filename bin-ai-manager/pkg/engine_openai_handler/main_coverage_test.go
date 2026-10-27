package engine_openai_handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

func Test_NewEngineOpenaiHandlerWithConfig_custom_base_url(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&openai.ChatCompletionRequest{})
		writeJSON(w, http.StatusOK, `{"id":"cmpl-9","choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
	}))
	t.Cleanup(srv.Close)

	key := "test-key-synthetic"
	h := NewEngineOpenaiHandlerWithConfig(key, srv.URL+"/v1beta/openai")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.SendOnce(ctx, &openai.ChatCompletionRequest{Model: "m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "cmpl-9" {
		t.Errorf("wrong response: %+v", res)
	}
	if gotPath != "/v1beta/openai/chat/completions" {
		t.Errorf("wrong path: %s", gotPath)
	}
	if gotAuth != "Bearer "+key {
		t.Errorf("wrong authorization header")
	}
}
