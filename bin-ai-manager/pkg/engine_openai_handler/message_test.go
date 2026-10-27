package engine_openai_handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"

	"github.com/sashabaranov/go-openai"
)

func Test_MessageSend(t *testing.T) {
	tests := []struct {
		name string

		engineModel ai.EngineModel
		messages    []*message.Message
		respBody    string

		expectModel   string
		expectMsgs    []openai.ChatCompletionMessage
		expectRole    message.Role
		expectContent string
	}{
		{
			name:        "model_from_engine_model",
			engineModel: ai.EngineModelOpenaiGPT5Dot2,
			messages: []*message.Message{
				{Role: message.RoleSystem, Content: "be brief"},
				{Role: message.RoleUser, Content: "hello"},
			},
			respBody: `{"choices":[{"message":{"role":"assistant","content":"answer one"}}]}`,

			expectModel: "gpt-5.2",
			expectMsgs: []openai.ChatCompletionMessage{
				{Role: "system", Content: "be brief"},
				{Role: "user", Content: "hello"},
			},
			expectRole:    message.RoleAssistant,
			expectContent: "answer one",
		},
		{
			name:        "empty_engine_model_uses_default",
			engineModel: ai.EngineModel(""),
			messages: []*message.Message{
				{Role: message.RoleUser, Content: "hi"},
			},
			respBody: `{"choices":[{"message":{"role":"assistant","content":"answer two"}}]}`,

			expectModel: defaultModel,
			expectMsgs: []openai.ChatCompletionMessage{
				{Role: "user", Content: "hi"},
			},
			expectRole:    message.RoleAssistant,
			expectContent: "answer two",
		},
		{
			name:        "empty_choices_returns_none_role",
			engineModel: ai.EngineModel("openai."),
			messages:    []*message.Message{},
			respBody:    `{"choices":[]}`,

			expectModel:   defaultModel,
			expectMsgs:    []openai.ChatCompletionMessage{},
			expectRole:    message.RoleNone,
			expectContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq openai.ChatCompletionRequest
			h, _ := newTestEngine(t, func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&gotReq)
				writeJSON(w, http.StatusOK, tt.respBody)
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cc := &aicall.AIcall{AIEngineModel: tt.engineModel}
			res, err := h.MessageSend(ctx, cc, tt.messages)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Role != tt.expectRole || res.Content != tt.expectContent {
				t.Errorf("wrong result. expect: %s/%q, got: %s/%q", tt.expectRole, tt.expectContent, res.Role, res.Content)
			}
			if gotReq.Model != tt.expectModel {
				t.Errorf("wrong model. expect: %s, got: %s", tt.expectModel, gotReq.Model)
			}
			if len(gotReq.Messages) != len(tt.expectMsgs) {
				t.Fatalf("wrong message count. expect: %d, got: %d", len(tt.expectMsgs), len(gotReq.Messages))
			}
			for i, m := range tt.expectMsgs {
				if gotReq.Messages[i].Role != m.Role || gotReq.Messages[i].Content != m.Content {
					t.Errorf("wrong message[%d]. expect: %+v, got: %+v", i, m, gotReq.Messages[i])
				}
			}
		})
	}
}
