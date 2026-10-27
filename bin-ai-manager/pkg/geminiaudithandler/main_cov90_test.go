package geminiaudithandler

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

const cov90ValidEvaluation = `{"overall_score":4,"dimensions":{` +
	`"helpfulness":{"score":4,"reason":"h"},"accuracy":{"score":5,"reason":"a"},` +
	`"tone":{"score":3,"reason":"t"},"goal_completion":{"score":4,"reason":"g"},` +
	`"tool_usage":{"score":2,"reason":"u"}},"summary":"s"}`

func newCov90Handler(t *testing.T, serve http.HandlerFunc) *geminiAuditHandler {
	t.Helper()

	srv := httptest.NewServer(serve)
	t.Cleanup(srv.Close)

	cfg := openai.DefaultConfig("test-key-0001")
	cfg.BaseURL = srv.URL

	return &geminiAuditHandler{client: openai.NewClientWithConfig(cfg)}
}

func Test_BuildPrompt_TruncationNotice(t *testing.T) {
	h := &geminiAuditHandler{}

	truncated := h.BuildPrompt("prompt", "transcript", "en-US", true)
	if !strings.Contains(truncated, "only the 500 most recent are included") {
		t.Errorf("expected the truncation notice in the prompt, got: %s", truncated)
	}

	full := h.BuildPrompt("prompt", "transcript", "en-US", false)
	if strings.Contains(full, "only the 500 most recent are included") {
		t.Errorf("did not expect the truncation notice for a full transcript")
	}
}

func Test_ParseEvaluationResponse_Rejects(t *testing.T) {
	h := &geminiAuditHandler{}

	long := func(n int) string { return strings.Repeat("x", n) }
	withTool := func(score float64, reason string) string {
		return fmt.Sprintf(`{"overall_score":4,"dimensions":{`+
			`"helpfulness":{"score":4,"reason":"h"},"accuracy":{"score":4,"reason":"a"},`+
			`"tone":{"score":4,"reason":"t"},"goal_completion":{"score":4,"reason":"g"},`+
			`"tool_usage":{"score":%v,"reason":%q}},"summary":"s"}`, score, reason)
	}

	tests := []struct {
		name   string
		data   string
		expect string
	}{
		{"not json", `{bad`, "invalid JSON"},
		{"tool usage fractional", withTool(2.5, "u"), "tool_usage score is fractional"},
		{"tool usage out of range", withTool(6, "u"), "tool_usage score out of range"},
		{"tool usage reason too long", withTool(3, long(2001)), "tool_usage.reason exceeds max length"},
		{
			"summary too long",
			`{"overall_score":4,"dimensions":{"helpfulness":{"score":4},"accuracy":{"score":4},"tone":{"score":4},"goal_completion":{"score":4}},"summary":"` + long(5001) + `"}`,
			"summary exceeds max length",
		},
		{
			"dimension reason too long",
			`{"overall_score":4,"dimensions":{"helpfulness":{"score":4,"reason":"` + long(2001) + `"},"accuracy":{"score":4},"tone":{"score":4},"goal_completion":{"score":4}},"summary":"s"}`,
			"helpfulness.reason exceeds max length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.ParseEvaluationResponse([]byte(tt.data))
			if err == nil {
				t.Fatalf("expected an error, got: %+v", res)
			}
			if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expect, err)
			}
			if res != nil {
				t.Errorf("expected no result with an error, got: %+v", res)
			}
		})
	}
}

func Test_ParseEvaluationResponse_ToolUsage(t *testing.T) {
	h := &geminiAuditHandler{}

	res, err := h.ParseEvaluationResponse([]byte(cov90ValidEvaluation))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Dimensions.ToolUsage == nil || res.Dimensions.ToolUsage.Score != 2 || res.Dimensions.ToolUsage.Reason != "u" {
		t.Errorf("expected the tool usage dimension to be parsed, got: %+v", res.Dimensions.ToolUsage)
	}
	if res.OverallScore != 4 || res.Summary != "s" {
		t.Errorf("unexpected overall score or summary: %+v", res)
	}
}

func Test_Evaluate(t *testing.T) {
	type completion struct {
		status int
		body   string
	}

	tests := []struct {
		name         string
		reply        completion
		expectErr    string
		expectScore  int
		expectRawLen bool
	}{
		{
			name:         "success",
			reply:        completion{200, `{"choices":[{"message":{"role":"assistant","content":` + fmt.Sprintf("%q", cov90ValidEvaluation) + `}}]}`},
			expectScore:  4,
			expectRawLen: true,
		},
		{
			name:      "no choices",
			reply:     completion{200, `{"choices":[]}`},
			expectErr: "gemini returned no choices",
		},
		{
			name:      "unparsable evaluation",
			reply:     completion{200, `{"choices":[{"message":{"role":"assistant","content":"not json"}}]}`},
			expectErr: "invalid_evaluator_response",
		},
		{
			name:      "api error",
			reply:     completion{500, `{"error":{"message":"upstream failure","type":"server_error"}}`},
			expectErr: "gemini API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest map[string]any
			h := newCov90Handler(t, func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&gotRequest)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.reply.status)
				_, _ = w.Write([]byte(tt.reply.body))
			})

			res, raw, err := h.Evaluate(context.Background(), "the system prompt", "the transcript text", "ko-KR", true)

			if gotRequest["model"] != geminiModel {
				t.Errorf("expected the request to name the model %q, got: %v", geminiModel, gotRequest["model"])
			}
			messages, _ := gotRequest["messages"].([]any)
			if len(messages) != 1 || !strings.Contains(fmt.Sprint(messages[0]), "the transcript text") {
				t.Errorf("expected one message carrying the transcript, got: %v", messages)
			}

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("expected an error containing %q, got: %v", tt.expectErr, err)
				}
				if res != nil || raw != nil {
					t.Errorf("expected no result with an error, got: %+v, %s", res, raw)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.OverallScore != tt.expectScore {
				t.Errorf("expected overall score %d, got: %d", tt.expectScore, res.OverallScore)
			}
			if string(raw) != cov90ValidEvaluation {
				t.Errorf("expected the raw evaluation to be returned unchanged, got: %s", raw)
			}
		})
	}
}
