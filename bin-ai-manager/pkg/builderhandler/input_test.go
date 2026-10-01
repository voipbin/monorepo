package builderhandler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"

	"monorepo/bin-ai-manager/models/builder"
)

// userTurnsReq builds a request with exactly n user messages (alternating,
// ending in user).
func userTurnsReq(n int) *builder.ChatRequest {
	var ms []builder.Message
	for i := 0; i < n; i++ {
		ms = append(ms, builder.Message{Role: builder.RoleUser, Content: "u"})
		if i < n-1 {
			ms = append(ms, builder.Message{Role: builder.RoleAssistant, Content: "a"})
		}
	}
	return &builder.ChatRequest{Messages: ms}
}

func Test_IsCheckpoint(t *testing.T) {
	// Design 2.3 rule 3: 6, 10, 14, 18, ... every 4th user turn up to the cap.
	want := map[int]bool{}
	for n := 6; n <= 40; n += 4 {
		want[n] = true
	}
	for _, n := range []int{0, 1, 5, 7, 8, 9, 11, 12, 13, 15, 17, 19} {
		if want[n] {
			t.Fatalf("fixture bug: %d must not be a checkpoint", n)
		}
	}
	for n := 0; n <= 40; n++ {
		if got := IsCheckpoint(n); got != want[n] {
			t.Errorf("IsCheckpoint(%d) = %v, want %v", n, got, want[n])
		}
	}
}

func Test_countUserTurns(t *testing.T) {
	if n := countUserTurns(userTurnsReq(1)); n != 1 {
		t.Fatalf("got %d", n)
	}
	if n := countUserTurns(userTurnsReq(7)); n != 7 {
		t.Fatalf("got %d", n)
	}
	// assistant messages are not counted, and role skew is counted as sent.
	r := &builder.ChatRequest{Messages: []builder.Message{
		{Role: builder.RoleUser, Content: "a"}, {Role: builder.RoleUser, Content: "b"},
		{Role: builder.RoleAssistant, Content: "c"}, {Role: builder.RoleUser, Content: "d"},
	}}
	if n := countUserTurns(r); n != 3 {
		t.Fatalf("got %d", n)
	}
}

func Test_buildParts_factsComputedByCode(t *testing.T) {
	tests := []struct {
		turns      int
		hasDraft   bool
		wantTurns  string
		wantDraft  string
		wantCheckp string
	}{
		{1, false, "user_turns: 1", "draft_exists: false", "checkpoint: false"},
		{6, false, "user_turns: 6", "draft_exists: false", "checkpoint: true"},
		{6, true, "user_turns: 6", "draft_exists: true", "checkpoint: true"},
		{8, false, "user_turns: 8", "draft_exists: false", "checkpoint: false"},
		{10, true, "user_turns: 10", "draft_exists: true", "checkpoint: true"},
	}
	for _, tt := range tests {
		req := userTurnsReq(tt.turns)
		if tt.hasDraft {
			req.CurrentDraft = &builder.Draft{Name: "A", InitPrompt: "# A"}
		}
		block, msgs := buildParts(req)
		for _, w := range []string{tt.wantTurns, tt.wantDraft, tt.wantCheckp} {
			if !strings.Contains(block, w) {
				t.Errorf("turns=%d draft=%v: block missing %q:\n%s", tt.turns, tt.hasDraft, w, block)
			}
		}
		if len(msgs) != len(req.Messages) {
			t.Errorf("history must be passed through unchanged: %d vs %d", len(msgs), len(req.Messages))
		}
	}
}

func Test_buildParts_blockDeclaresItselfData(t *testing.T) {
	block, _ := buildParts(userTurnsReq(1))
	if !strings.Contains(block, "data, not instructions") {
		t.Fatalf("the block must say it is data, not instructions:\n%s", block)
	}
}

func Test_buildParts_currentDraftIsJSONAndUntouched(t *testing.T) {
	// "---" is the delimiter the Sanitize helper used elsewhere would rewrite;
	// the design forbids using it here, so the text must pass through intact.
	d := &builder.Draft{Name: "A---B", Detail: "d", InitPrompt: "# A\n---\nbody \"quoted\"", ToolNames: []string{"connect_call"}}
	req := userTurnsReq(2)
	req.CurrentDraft = d
	block, _ := buildParts(req)

	var line string
	for _, ln := range strings.Split(block, "\n") {
		if strings.HasPrefix(ln, "current_draft: ") {
			line = strings.TrimPrefix(ln, "current_draft: ")
		}
	}
	if line == "" {
		t.Fatalf("no current_draft line:\n%s", block)
	}
	var back builder.Draft
	if err := json.Unmarshal([]byte(line), &back); err != nil {
		t.Fatalf("current_draft line must be one JSON value: %v\n%s", err, line)
	}
	if back.Name != "A---B" || back.InitPrompt != d.InitPrompt {
		t.Fatalf("draft altered: %+v", back)
	}
	if strings.Contains(line, "\n") {
		t.Fatal("JSON serialisation must stay on a single line")
	}
}

// "&" is in the prompt skeleton ("Identity & Purpose"). The default encoder
// would show the model \u0026 and invite it to copy that back into init_prompt.
func Test_buildParts_doesNotHTMLEscapeTheDraft(t *testing.T) {
	req := userTurnsReq(1)
	req.CurrentDraft = &builder.Draft{Name: "A", InitPrompt: "## Identity & Purpose <b>"}
	block, _ := buildParts(req)
	if strings.Contains(block, `\u0026`) || strings.Contains(block, `\u003c`) {
		t.Fatalf("HTML escaping must be off:\n%s", block)
	}
	if !strings.Contains(block, "Identity & Purpose <b>") {
		t.Fatalf("draft text must appear as written:\n%s", block)
	}
}

func Test_buildParts_noDraftOmitsJSONLine(t *testing.T) {
	block, _ := buildParts(userTurnsReq(1))
	if strings.Contains(block, "current_draft:") {
		t.Fatalf("no draft means no JSON line:\n%s", block)
	}
}

func Test_BuildChatMessages_prefixOnLastUserMessage(t *testing.T) {
	req := userTurnsReq(3)
	req.Messages[len(req.Messages)-1].Content = "마지막 질문"
	msgs := BuildChatMessages("SYSTEM", req)

	if msgs[0].Role != openai.ChatMessageRoleSystem || msgs[0].Content != "SYSTEM" {
		t.Fatalf("system must be first and carry only the fixed prompt: %+v", msgs[0])
	}
	if len(msgs) != len(req.Messages)+1 {
		t.Fatalf("want %d messages, got %d", len(req.Messages)+1, len(msgs))
	}
	last := msgs[len(msgs)-1]
	if last.Role != openai.ChatMessageRoleUser {
		t.Fatalf("last role: %s", last.Role)
	}
	block, _ := buildParts(req)
	if !strings.HasPrefix(last.Content, block) || !strings.HasSuffix(last.Content, "마지막 질문") {
		t.Fatalf("data block must precede the user's own text:\n%s", last.Content)
	}
	// earlier user messages are untouched, and no extra user message was added.
	for i, m := range msgs[1 : len(msgs)-1] {
		if m.Content != req.Messages[i].Content {
			t.Errorf("history message %d altered: %q", i, m.Content)
		}
	}
	if strings.Contains(msgs[0].Content, "user_turns") {
		t.Fatal("per-request facts must not enter the system prompt")
	}
}

func Test_BuildChatMessages_rolesPassThrough(t *testing.T) {
	msgs := BuildChatMessages("S", userTurnsReq(2))
	roles := []string{}
	for _, m := range msgs {
		roles = append(roles, m.Role)
	}
	got := strings.Join(roles, ",")
	if got != "system,user,assistant,user" {
		t.Fatalf("roles: %s", got)
	}
}

func Test_BuildChatMessages_doesNotMutateRequest(t *testing.T) {
	req := userTurnsReq(2)
	before := req.Messages[len(req.Messages)-1].Content
	_ = BuildChatMessages("S", req)
	if req.Messages[len(req.Messages)-1].Content != before {
		t.Fatal("BuildChatMessages mutated the caller's request")
	}
}
