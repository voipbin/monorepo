package geminiproposalhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func newCov90Handler(t *testing.T, status int, body string) (*geminiProposalHandler, *map[string]any) {
	t.Helper()

	var gotRequest map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotRequest)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	cfg := openai.DefaultConfig("test-key-0001")
	cfg.BaseURL = srv.URL

	return &geminiProposalHandler{client: openai.NewClientWithConfig(cfg)}, &gotRequest
}

func Test_NewGeminiProposalHandler(t *testing.T) {
	h, ok := NewGeminiProposalHandler("test-key-0001").(*geminiProposalHandler)
	if !ok || h.client == nil {
		t.Fatalf("expected a proposal handler with a client, got: %T", h)
	}
}

func Test_BuildPrompt_ToolUsageReason(t *testing.T) {
	h := &geminiProposalHandler{}

	withTool := h.BuildPrompt("orig", []AuditBlock{{Index: 1, OverallScore: 2, ToolUsageR: "wrong tool called"}}, "en-US")
	if !strings.Contains(withTool, "tool_usage:      wrong tool called") {
		t.Errorf("expected the tool usage reason in the prompt, got: %s", withTool)
	}

	withoutTool := h.BuildPrompt("orig", []AuditBlock{{Index: 1, OverallScore: 2}}, "en-US")
	if strings.Contains(withoutTool, "tool_usage:") {
		t.Errorf("did not expect a tool usage line without a reason")
	}
}

func Test_Evaluate(t *testing.T) {
	valid := `{"proposed_prompt":"a better prompt","rationale":"because"}`

	tests := []struct {
		name      string
		status    int
		body      string
		expectErr string
	}{
		{"success", 200, `{"choices":[{"message":{"role":"assistant","content":` + fmt.Sprintf("%q", valid) + `}}]}`, ""},
		{"no choices", 200, `{"choices":[]}`, "gemini returned no choices"},
		{"unparsable proposal", 200, `{"choices":[{"message":{"role":"assistant","content":"{}"}}]}`, "invalid_evaluator_response"},
		{"api error", 500, `{"error":{"message":"upstream failure","type":"server_error"}}`, "gemini API error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, gotRequest := newCov90Handler(t, tt.status, tt.body)

			res, err := h.Evaluate(context.Background(), "the original prompt", []AuditBlock{{Index: 1, OverallScore: 2, Summary: "audit summary"}}, "en-US")

			if (*gotRequest)["model"] != geminiModel {
				t.Errorf("expected the request to name the model %q, got: %v", geminiModel, (*gotRequest)["model"])
			}
			messages, _ := (*gotRequest)["messages"].([]any)
			if len(messages) != 1 || !strings.Contains(fmt.Sprint(messages[0]), "audit summary") {
				t.Errorf("expected one message carrying the audit block, got: %v", messages)
			}

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("expected an error containing %q, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("expected no result with an error, got: %+v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.ProposedPrompt != "a better prompt" || res.Rationale != "because" {
				t.Errorf("unexpected proposal: %+v", res)
			}
		})
	}
}
