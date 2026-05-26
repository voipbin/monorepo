package aiaudithandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/message"
)

func TestResolveLanguage(t *testing.T) {
	tests := []struct {
		name        string
		requestLang string
		sttLang     string
		want        string
	}{
		{"request lang wins", "ko-KR", "en-US", "ko-KR"},
		{"stt fallback", "", "ja-JP", "ja-JP"},
		{"default", "", "", "en-US"},
		{"invalid request falls back to stt", "bad lang!", "fr-FR", "fr-FR"},
		{"invalid request and stt falls back to default", "bad!", "", "en-US"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveLanguage(tt.requestLang, tt.sttLang)
			if got != tt.want {
				t.Errorf("resolveLanguage(%q, %q) = %q; want %q", tt.requestLang, tt.sttLang, got, tt.want)
			}
		})
	}
}

func TestHasToolCalls(t *testing.T) {
	noTools := []*message.Message{{Content: "hello"}}
	withToolCall := []*message.Message{{ToolCalls: []message.ToolCall{{ID: "tc1"}}}}
	withToolResult := []*message.Message{{ToolCallID: "tc1", Content: "result"}}

	if hasToolCalls(noTools) {
		t.Error("expected false for messages without tool calls")
	}
	if !hasToolCalls(withToolCall) {
		t.Error("expected true for messages with ToolCalls")
	}
	if !hasToolCalls(withToolResult) {
		t.Error("expected true for messages with ToolCallID")
	}
}

func TestBuildTranscript_truncatedFlag(t *testing.T) {
	msgs := make([]*message.Message, maxMessages)
	for i := range msgs {
		msgs[i] = &message.Message{Content: "x", Role: "user"}
	}
	var truncated bool
	buildTranscript(msgs, &truncated)
	if !truncated {
		t.Error("expected truncated=true when len(msgs)==maxMessages")
	}
}

func TestBuildTranscript_toolCallFormatting(t *testing.T) {
	msgs := []*message.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", ToolCalls: []message.ToolCall{{ID: "tc1"}}},
		{Role: "tool", ToolCallID: "tc1", Content: "result_data"},
	}
	var truncated bool
	result := buildTranscript(msgs, &truncated)
	if truncated {
		t.Error("expected truncated=false for 3 messages")
	}
	if result == "" {
		t.Error("expected non-empty transcript")
	}
}

func TestExtractPromptSnapshots_missingKey(t *testing.T) {
	ac := &aicall.AIcall{
		Metadata: map[string]any{},
	}
	_, err := extractPromptSnapshots(ac)
	if err == nil {
		t.Error("expected error when prompt_snapshots key is missing")
	}
}

func TestExtractPromptSnapshots_nilMetadata(t *testing.T) {
	ac := &aicall.AIcall{}
	_, err := extractPromptSnapshots(ac)
	if err == nil {
		t.Error("expected error when Metadata is nil")
	}
}

func TestExtractPromptSnapshots_valid(t *testing.T) {
	aiID := uuid.Must(uuid.NewV4())
	histID := uuid.Must(uuid.NewV4())
	snaps := []map[string]any{
		{
			"ai_id":             aiID.String(),
			"prompt":            "system prompt text",
			"prompt_history_id": histID.String(),
		},
	}
	ac := &aicall.AIcall{
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: snaps,
		},
	}
	result, err := extractPromptSnapshots(ac)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(result))
	}
	_ = result
}

// Compile-check that aiaudit constants are accessible.
func TestAIAuditStatusConstants(t *testing.T) {
	_ = aiaudit.StatusProgressing
	_ = aiaudit.StatusCompleted
	_ = aiaudit.StatusFailed
}
