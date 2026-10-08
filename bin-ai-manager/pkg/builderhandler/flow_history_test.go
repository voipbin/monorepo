package builderhandler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/flowbuilder"
)

// sentMessages runs the request through the real Flow Builder Chat path and
// returns the messages that reached the model.
func sentMessages(t *testing.T, req *flowbuilder.ChatRequest) []openai.ChatCompletionMessage {
	t.Helper()
	s := &recordSender{reply: flowGoodReply}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)
	if _, err := h.Chat(context.Background(), customerID, req); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	return s.req.Messages
}

func decodeMessageOnly(t *testing.T, content string) string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		t.Fatalf("Wrong match. expect: a JSON object, got: %q (%v)", content, err)
	}
	if len(m) != 1 {
		t.Fatalf("Wrong match. expect: exactly the key message, got: %v", m)
	}
	return m["message"]
}

func Test_FlowChat_historyAssistantTurnsAreSentAsJSON(t *testing.T) {
	tests := []struct {
		name      string
		assistant string
	}{
		{"korean", "배송 조회로 할까요?"},
		{"special characters", "say \"hi\" \\ and\nnew line & <b> done"},
		{"already looks like a wrapped message", `{"message":"x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := flowChatReq()
			req.Messages = []flowbuilder.Message{
				{Role: flowbuilder.RoleUser, Content: `{"looks":"like json"}`},
				{Role: flowbuilder.RoleAssistant, Content: tt.assistant},
				{Role: flowbuilder.RoleUser, Content: "go on"},
			}
			got := sentMessages(t, req)
			if len(got) != 4 || got[0].Role != openai.ChatMessageRoleSystem {
				t.Fatalf("Wrong match. expect: system plus 3 messages, got: %d", len(got))
			}
			wantRoles := []string{openai.ChatMessageRoleSystem, openai.ChatMessageRoleUser, openai.ChatMessageRoleAssistant, openai.ChatMessageRoleUser}
			for i, want := range wantRoles {
				if got[i].Role != want {
					t.Errorf("Wrong match. message %d expect role: %s, got: %s", i, want, got[i].Role)
				}
			}
			if v := decodeMessageOnly(t, got[2].Content); v != tt.assistant {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.assistant, v)
			}
			if !strings.HasSuffix(got[1].Content, `{"looks":"like json"}`) || strings.Contains(got[1].Content, `"message"`) {
				t.Errorf("Wrong match. expect: the user turn as typed, got: %q", got[1].Content)
			}
			if !strings.HasSuffix(got[3].Content, "go on") || !strings.Contains(got[3].Content, "User message:") {
				t.Errorf("Wrong match. expect: the facts block before the last user turn, got: %q", got[3].Content)
			}
			if strings.Contains(got[2].Content, "\\u0026") || strings.Contains(got[2].Content, "\\u003c") || strings.HasSuffix(got[2].Content, "\n") {
				t.Errorf("Wrong match. expect: no HTML escapes and no trailing newline, got: %q", got[2].Content)
			}
		})
	}
}

func Test_FlowChat_historyWithoutAssistantTurnsIsUnchanged(t *testing.T) {
	got := sentMessages(t, flowChatReq())
	if len(got) != 2 || !strings.HasSuffix(got[1].Content, "I want a greeting flow") {
		t.Errorf("Wrong match. expect: system plus the user turn, got: %+v", got)
	}
}

// The next turn of a draft conversation: the client sends only the visible
// message of the draft turn plus the current draft.
func Test_FlowChat_nextTurnOfADraftConversation(t *testing.T) {
	s := &recordSender{reply: flowGoodReply}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil).Times(2)

	first, err := h.Chat(context.Background(), customerID, flowChatReq())
	if err != nil || first.Draft == nil {
		t.Fatalf("Wrong match. expect: a draft, got: %v, %v", first, err)
	}

	next := flowChatReq()
	next.CurrentDraft = first.Draft
	next.Messages = []flowbuilder.Message{
		{Role: flowbuilder.RoleUser, Content: "I want a greeting flow"},
		{Role: flowbuilder.RoleAssistant, Content: first.Message},
		{Role: flowbuilder.RoleUser, Content: "change the greeting"},
	}
	if _, err := h.Chat(context.Background(), customerID, next); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	got := s.req.Messages
	if v := decodeMessageOnly(t, got[2].Content); v != first.Message {
		t.Errorf("Wrong match. expect: %q, got: %q", first.Message, v)
	}
	if !strings.Contains(got[3].Content, "current_draft: ") {
		t.Errorf("Wrong match. expect: the current draft in the facts block, got: %q", got[3].Content)
	}
}

// A real follow-up has several assistant turns. Every one is wrapped, every
// user turn is as typed, only the last user turn carries the facts block, and
// the system prompt is not wrapped.
func Test_FlowChat_everyAssistantTurnIsWrapped(t *testing.T) {
	req := flowChatReq()
	req.Messages = []flowbuilder.Message{
		{Role: flowbuilder.RoleUser, Content: "  first  "},
		{Role: flowbuilder.RoleAssistant, Content: "  question one \n"},
		{Role: flowbuilder.RoleUser, Content: "second"},
		{Role: flowbuilder.RoleAssistant, Content: "question two"},
		{Role: flowbuilder.RoleUser, Content: "third"},
	}
	got := sentMessages(t, req)
	if len(got) != 6 {
		t.Fatalf("Wrong match. expect: 6 messages, got: %d", len(got))
	}
	if got[0].Role != openai.ChatMessageRoleSystem || strings.HasPrefix(got[0].Content, "{") {
		t.Errorf("Wrong match. expect: the unwrapped system prompt, got: %.40q", got[0].Content)
	}
	for i, want := range []string{"system", "user", "assistant", "user", "assistant", "user"} {
		if got[i].Role != want {
			t.Errorf("Wrong match. message %d expect role: %s, got: %s", i, want, got[i].Role)
		}
	}
	if v := decodeMessageOnly(t, got[2].Content); v != "  question one \n" {
		t.Errorf("Wrong match. expect: the text untouched, got: %q", v)
	}
	if v := decodeMessageOnly(t, got[4].Content); v != "question two" {
		t.Errorf("Wrong match. expect: %q, got: %q", "question two", v)
	}
	for _, i := range []int{1, 3} {
		if strings.Contains(got[i].Content, "User message:") {
			t.Errorf("Wrong match. expect: the facts block only on the last user turn, got it on %d", i)
		}
	}
	if got[1].Content != "  first  " || got[3].Content != "second" {
		t.Errorf("Wrong match. expect: earlier user turns as typed, got: %q, %q", got[1].Content, got[3].Content)
	}
	if !strings.Contains(got[5].Content, "User message:\nthird") {
		t.Errorf("Wrong match. expect: the facts block before the last turn, got: %q", got[5].Content)
	}
}

// Characters that JSON escapes or that are not valid text.
func Test_FlowAssistantTurn_characters(t *testing.T) {
	tests := []struct {
		name, in, wantDecoded, wantContains string
	}{
		{"empty", "", "", `{"message":""}`},
		{"carriage return", "a\r\nb\rc", "a\r\nb\rc", `\r\n`},
		{"exact form", "x", "x", `{"message":"x"}`},
		{"tab and control", "a\tb\x01c", "a\tb\x01c", `\t`},
		{"line separators", "a\u2028b\u2029c", "a\u2028b\u2029c", `\u2028`},
		{"invalid utf-8 is replaced", "a\xffb", "a\ufffdb", ""},
		{"html characters stay", "a&b<c>", "a&b<c>", "a&b<c>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flowAssistantTurn(tt.in)
			if v := decodeMessageOnly(t, got); v != tt.wantDecoded {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.wantDecoded, v)
			}
			if tt.name == "exact form" && got != tt.wantContains {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.wantContains, got)
			}
			if !strings.Contains(got, tt.wantContains) || strings.HasSuffix(got, "\n") {
				t.Errorf("Wrong match. expect: %q inside and no trailing newline, got: %q", tt.wantContains, got)
			}
		})
	}
}
