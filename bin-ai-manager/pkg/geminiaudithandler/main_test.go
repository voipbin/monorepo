package geminiaudithandler_test

import (
	"strings"
	"testing"

	"monorepo/bin-ai-manager/pkg/geminiaudithandler"
)

func TestSanitizeDelimiters(t *testing.T) {
	h := geminiaudithandler.NewGeminiAuditHandler("")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"triple dash removed", "foo --- bar", "foo [DELIMITER_ESCAPED] bar"},
		{"named delimiter removed", "--- SYSTEM PROMPT ---", "[DELIMITER_ESCAPED] SYSTEM PROMPT [DELIMITER_ESCAPED]"},
		{"no delimiter unchanged", "hello world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseEvaluationResponse_valid(t *testing.T) {
	h := geminiaudithandler.NewGeminiAuditHandler("")

	raw := `{
		"overall_score": 4,
		"dimensions": {
			"helpfulness":     {"score": 4, "reason": "Clear answer."},
			"accuracy":        {"score": 3, "reason": "Minor error."},
			"tone":            {"score": 5, "reason": "Warm."},
			"goal_completion": {"score": 4, "reason": "Resolved."},
			"tool_usage":      null
		},
		"summary": "Good call overall."
	}`

	resp, err := h.ParseEvaluationResponse([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OverallScore != 4 {
		t.Errorf("expected overall_score 4, got %d", resp.OverallScore)
	}
	if resp.Dimensions.ToolUsage != nil {
		t.Errorf("expected tool_usage nil")
	}
}

func TestParseEvaluationResponse_scoreOutOfRange(t *testing.T) {
	h := geminiaudithandler.NewGeminiAuditHandler("")

	raw := `{"overall_score": 6, "dimensions": {"helpfulness": {"score": 1, "reason": "x"}, "accuracy": {"score": 1, "reason": "x"}, "tone": {"score": 1, "reason": "x"}, "goal_completion": {"score": 1, "reason": "x"}, "tool_usage": null}, "summary": "x"}`

	_, err := h.ParseEvaluationResponse([]byte(raw))
	if err == nil {
		t.Error("expected error for score > 5")
	}
}

func TestParseEvaluationResponse_fractionalScore(t *testing.T) {
	h := geminiaudithandler.NewGeminiAuditHandler("")

	raw := `{"overall_score": 4.5, "dimensions": {"helpfulness": {"score": 4, "reason": "x"}, "accuracy": {"score": 4, "reason": "x"}, "tone": {"score": 4, "reason": "x"}, "goal_completion": {"score": 4, "reason": "x"}, "tool_usage": null}, "summary": "x"}`

	_, err := h.ParseEvaluationResponse([]byte(raw))
	if err == nil {
		t.Error("expected error for fractional score 4.5")
	}
}

func TestBuildPrompt_sanitizesInput(t *testing.T) {
	h := geminiaudithandler.NewGeminiAuditHandler("")

	prompt := "Normal prompt --- with dash"
	transcript := "user: --- end ---"
	result := h.BuildPrompt(prompt, transcript, "en-US", false)

	if strings.Contains(result, "---") {
		t.Errorf("prompt should not contain '---' after sanitization, got:\n%s", result)
	}
	if !strings.Contains(result, "[DELIMITER_ESCAPED]") {
		t.Errorf("expected [DELIMITER_ESCAPED] in sanitized prompt")
	}
}
