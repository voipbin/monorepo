package builder

import (
	"testing"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/tool"
	cerrors "monorepo/bin-common-handler/models/errors"
)

func msgs(n int, content string) []Message {
	out := make([]Message, 0, n)
	for i := 0; i < n; i++ {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}
		out = append(out, Message{Role: role, Content: content})
	}
	return out
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func reasonOf(t *testing.T, err error) string {
	t.Helper()
	ve, ok := err.(*cerrors.VoipbinError)
	if !ok {
		t.Fatalf("want *VoipbinError, got %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", ve.Status)
	}
	return ve.Reason
}

func Test_ValidateRequest_ok(t *testing.T) {
	tests := []struct {
		name string
		req  *ChatRequest
	}{
		{"single user", &ChatRequest{Messages: msgs(1, "hello")}},
		{"40 messages ending in user", &ChatRequest{Messages: msgs(39, "a")}},
		{"2000 runes exactly", &ChatRequest{Messages: []Message{{Role: RoleUser, Content: repeat("가", 2000)}}}},
		{"draft at limits", &ChatRequest{
			Messages: msgs(1, "hi"),
			CurrentDraft: &Draft{
				Name:       repeat("a", MaxNameRunes),
				Detail:     repeat("b", MaxDetailRunes),
				InitPrompt: repeat("c", MaxInitPromptRunes),
				ToolNames:  []string{"connect_call"},
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateRequest(tt.req); err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func Test_ValidateRequest_reject(t *testing.T) {
	tests := []struct {
		name string
		req  *ChatRequest
	}{
		{"nil", nil},
		{"empty messages", &ChatRequest{}},
		{"41 messages", &ChatRequest{Messages: msgs(41, "a")}},
		{"2001 runes", &ChatRequest{Messages: []Message{{Role: RoleUser, Content: repeat("가", 2001)}}}},
		{"bad role", &ChatRequest{Messages: []Message{{Role: "system", Content: "x"}}}},
		{"empty role", &ChatRequest{Messages: []Message{{Role: "", Content: "x"}}}},
		{"empty content", &ChatRequest{Messages: []Message{{Role: RoleUser, Content: ""}}}},
		{"last is assistant", &ChatRequest{Messages: []Message{{Role: RoleUser, Content: "a"}, {Role: RoleAssistant, Content: "b"}}}},
		{"name too long", &ChatRequest{Messages: msgs(1, "a"), CurrentDraft: &Draft{Name: repeat("a", MaxNameRunes+1)}}},
		{"detail too long", &ChatRequest{Messages: msgs(1, "a"), CurrentDraft: &Draft{Detail: repeat("a", MaxDetailRunes+1)}}},
		{"init_prompt too long", &ChatRequest{Messages: msgs(1, "a"), CurrentDraft: &Draft{InitPrompt: repeat("a", MaxInitPromptRunes+1)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequest(tt.req)
			if err == nil {
				t.Fatal("want error")
			}
			if got := reasonOf(t, err); got != ReasonInvalidArgument {
				t.Fatalf("reason: want %s, got %s", ReasonInvalidArgument, got)
			}
		})
	}
}

// The aggregate cap must trip before the 40 x 2000 per-message product does.
func Test_ValidateRequest_totalRuneCap(t *testing.T) {
	// 20 messages x 2000 runes = 40000 (ok), ending in user (index 19 is assistant, so use 21).
	ok := &ChatRequest{Messages: msgs(19, repeat("가", 2000))} // 19 x 2000 = 38000
	if err := ValidateRequest(ok); err != nil {
		t.Fatalf("38000 runes should pass: %v", err)
	}
	// exactly 40000: 19 x 2000 + 1 x 2000 = 20 messages, last must be user -> use 21 msgs with small tail.
	exact := msgs(19, repeat("가", 2000)) // 38000, last index 18 -> user
	exact = append(exact, Message{Role: RoleAssistant, Content: repeat("가", 1999)}, Message{Role: RoleUser, Content: "가"})
	if n := totalRunes(&ChatRequest{Messages: exact}); n != MaxTotalRunes {
		t.Fatalf("fixture should be exactly %d runes, got %d", MaxTotalRunes, n)
	}
	if err := ValidateRequest(&ChatRequest{Messages: exact}); err != nil {
		t.Fatalf("exactly 40000 should pass: %v", err)
	}
	over := append([]Message{}, exact...)
	over[len(over)-1].Content = "가가"
	if err := ValidateRequest(&ChatRequest{Messages: over}); err == nil {
		t.Fatal("40001 should fail")
	}
}

// Draft fields count toward the aggregate cap, tool names included.
func Test_ValidateRequest_draftCountsTowardTotal(t *testing.T) {
	m := msgs(19, repeat("가", 2000)) // 38000
	req := &ChatRequest{Messages: m, CurrentDraft: &Draft{InitPrompt: repeat("a", 2001)}}
	if err := ValidateRequest(req); err == nil {
		t.Fatal("38000 + 2001 should exceed the aggregate cap")
	}
	req = &ChatRequest{Messages: m, CurrentDraft: &Draft{InitPrompt: repeat("a", 1990), ToolNames: []string{repeat("t", 11)}}}
	if err := ValidateRequest(req); err == nil {
		t.Fatal("tool names must be counted")
	}
}

// Korean text is 3 bytes per rune; the cap is in runes, not bytes.
func Test_ValidateRequest_runesNotBytes(t *testing.T) {
	s := repeat("가", 2000)
	if utf8.RuneCountInString(s) != 2000 || len(s) != 6000 {
		t.Fatal("fixture sanity")
	}
	if err := ValidateRequest(&ChatRequest{Messages: []Message{{Role: RoleUser, Content: s}}}); err != nil {
		t.Fatalf("2000 Korean runes (6000 bytes) must pass: %v", err)
	}
}

// Error text must never echo the caller's input.
func Test_ValidateRequest_doesNotEchoInput(t *testing.T) {
	secret := "SECRET-입력-문자열"
	req := &ChatRequest{Messages: []Message{{Role: "bogus", Content: secret}}}
	err := ValidateRequest(req)
	if err == nil {
		t.Fatal("want error")
	}
	if contains(err.Error(), secret) {
		t.Fatalf("error leaked input: %v", err)
	}
	req = &ChatRequest{Messages: []Message{{Role: RoleUser, Content: repeat(secret, 300)}}}
	if err := ValidateRequest(req); err == nil || contains(err.Error(), secret) {
		t.Fatalf("length error must not echo input: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func Test_AllowedTools_subsetOfAllToolNames(t *testing.T) {
	all := map[tool.ToolName]bool{}
	for _, n := range tool.AllToolNames {
		all[n] = true
	}
	if len(AllowedTools) != 6 {
		t.Fatalf("want 6 allowed tools, got %d", len(AllowedTools))
	}
	for _, n := range AllowedTools {
		if !all[n] {
			t.Errorf("%s is not in tool.AllToolNames", n)
		}
	}
	for _, banned := range []string{"create_call", "join_queue", "get_variables", "stop_flow"} {
		if IsAllowedTool(banned) {
			t.Errorf("%s must not be allowed", banned)
		}
	}
	for _, ok := range []string{"connect_call", "stop_service", "send_email", "send_message", "set_variables", "case_create"} {
		if !IsAllowedTool(ok) {
			t.Errorf("%s must be allowed", ok)
		}
	}
}

func Test_URIs(t *testing.T) {
	if URIChat != "/v1/ai_builder/chat" || URIStatus != "/v1/ai_builder/status" {
		t.Fatal("internal URIs are a wire contract shared with requesthandler and listenhandler")
	}
}
