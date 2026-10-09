package flowbuilder

import (
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/builder"
	cerrors "monorepo/bin-common-handler/models/errors"
)

func validRequest() *ChatRequest {
	return &ChatRequest{
		Messages:             []Message{{Role: RoleUser, Content: "hello"}},
		SupportedActionTypes: []string{"talk", "hangup"},
	}
}

func Test_ValidateRequest(t *testing.T) {
	bigOption := map[string]any{"text": strings.Repeat("a", MaxOptionBytes)}
	deep := map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": map[string]any{"e": map[string]any{"f": map[string]any{"g": map[string]any{"h": map[string]any{"i": 1}}}}}}}}}

	manyActions := make([]map[string]any, MaxFlowNodes+1)
	for i := range manyActions {
		manyActions[i] = map[string]any{"id": "x", "type": "stop"}
	}

	tests := []struct {
		name    string
		mutate  func(r *ChatRequest)
		wantErr bool
	}{
		{"valid", func(r *ChatRequest) {}, false},
		{"nil request is checked separately", nil, true},
		{"empty messages", func(r *ChatRequest) { r.Messages = nil }, true},
		{"last message from the assistant", func(r *ChatRequest) {
			r.Messages = append(r.Messages, Message{Role: RoleAssistant, Content: "q"})
		}, true},
		{"message over the shared limit", func(r *ChatRequest) {
			r.Messages[0].Content = strings.Repeat("a", builder.MaxMessageRunes+1)
		}, true},
		{"no supported types", func(r *ChatRequest) { r.SupportedActionTypes = nil }, true},
		{"too many supported types", func(r *ChatRequest) {
			r.SupportedActionTypes = make([]string, MaxSupportedTypes+1)
			for i := range r.SupportedActionTypes {
				r.SupportedActionTypes[i] = "t"
			}
		}, true},
		{"empty supported type", func(r *ChatRequest) { r.SupportedActionTypes = []string{""} }, true},
		{"too many nodes", func(r *ChatRequest) { r.CurrentDraft = &Draft{Actions: manyActions} }, true},
		{"option over the byte limit", func(r *ChatRequest) {
			r.CurrentDraft = &Draft{Actions: []map[string]any{{"id": "a", "option": bigOption}}}
		}, true},
		{"option nested too deep", func(r *ChatRequest) {
			r.CurrentDraft = &Draft{Actions: []map[string]any{{"id": "a", "option": deep}}}
		}, true},
		{"more labels than actions", func(r *ChatRequest) {
			r.CurrentDraft = &Draft{
				Actions: []map[string]any{{"id": "a"}},
				Labels:  map[string]string{"a": "x", "b": "y"},
			}
		}, true},
		{"label too long", func(r *ChatRequest) {
			r.CurrentDraft = &Draft{
				Actions: []map[string]any{{"id": "a"}},
				Labels:  map[string]string{"a": strings.Repeat("l", MaxLabelRunes+1)},
			}
		}, true},
		{"a sane draft", func(r *ChatRequest) {
			r.CurrentDraft = &Draft{
				Actions: []map[string]any{{"id": "a", "type": "stop"}},
				Labels:  map[string]string{"a": "end"},
			}
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *ChatRequest
			if tt.mutate != nil {
				req = validRequest()
				tt.mutate(req)
			}

			err := ValidateRequest(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.wantErr, err)
			}
			if err == nil {
				return
			}
			ve, ok := err.(*cerrors.VoipbinError)
			if !ok || ve.Reason != ReasonInvalidArgument {
				t.Errorf("Wrong match. expect: INVALID_ARGUMENT VoipbinError, got: %T %v", err, err)
			}
		})
	}
}

func Test_ValidateRequest_neverEchoesCallerText(t *testing.T) {
	req := validRequest()
	req.SupportedActionTypes = []string{"SECRET-" + strings.Repeat("a", MaxSupportedTypeRunes)}

	err := ValidateRequest(req)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if strings.Contains(err.Error(), "SECRET-") {
		t.Errorf("Wrong match. the caller's text reached the error: %v", err)
	}
}

func Test_depth(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int
	}{
		{"scalar", 1, 0},
		{"empty map", map[string]any{}, 1},
		{"nested", map[string]any{"a": []any{map[string]any{"b": 1}}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := depth(tt.in); got != tt.want {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.want, got)
			}
		})
	}
}
