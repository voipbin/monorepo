package aicallhandler

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func draftTool(args string) *message.ToolCall {
	return &message.ToolCall{
		ID:   "d7000000-0000-11f0-0000-000000000002",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameDraftMessage,
			Arguments: args,
		},
	}
}

func Test_toolHandleDraftMessage(t *testing.T) {
	c := &aicall.AIcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("d7000000-0000-11f0-0000-000000000001")}}

	tests := []struct {
		name string
		args string

		wantResult  string
		wantErrPart string
		wantBlock   *CardBlock
	}{
		{
			name:       "valid with title",
			args:       `{"to":"+14155550100","text":"Hello","title":"Reminder"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "sms_draft", Title: "Reminder", To: "+14155550100", Text: "Hello"},
		},
		{
			name:       "default title and trimmed recipient",
			args:       `{"to":" +14155550100 ","text":"Hello"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "sms_draft", Title: "SMS draft", To: "+14155550100", Text: "Hello"},
		},
		{
			name:       "title is cut at 200 characters",
			args:       `{"to":"+14155550100","text":"Hello","title":"` + strings.Repeat("a", 250) + `"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "sms_draft", Title: strings.Repeat("a", 197) + "...", To: "+14155550100", Text: "Hello"},
		},
		{
			name:       "text at the limit is accepted (runes, not bytes)",
			args:       `{"to":"+14155550100","text":"` + strings.Repeat("가", 1600) + `"}`,
			wantResult: "success",
			wantBlock:  &CardBlock{Type: "sms_draft", Title: "SMS draft", To: "+14155550100", Text: strings.Repeat("가", 1600)},
		},
		{
			name:        "text over the limit is rejected, not cut",
			args:        `{"to":"+14155550100","text":"` + strings.Repeat("a", 1601) + `"}`,
			wantResult:  "failed",
			wantErrPart: "too long",
		},
		{name: "blank text", args: `{"to":"+14155550100","text":"   "}`, wantResult: "failed", wantErrPart: "text is required"},
		{name: "missing text", args: `{"to":"+14155550100"}`, wantResult: "failed", wantErrPart: "text is required"},
		{name: "missing recipient", args: `{"text":"Hello"}`, wantResult: "failed", wantErrPart: "E.164"},
		{name: "recipient without plus", args: `{"to":"14155550100","text":"Hello"}`, wantResult: "failed", wantErrPart: "E.164"},
		{name: "recipient too short", args: `{"to":"+12345","text":"Hello"}`, wantResult: "failed", wantErrPart: "E.164"},
		{name: "recipient with letters", args: `{"to":"+1415555abcd","text":"Hello"}`, wantResult: "failed", wantErrPart: "E.164"},
		{name: "invalid json", args: `{`, wantResult: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no request handler, no message handler: any send or write would panic.
			h := &aicallHandler{}
			res := h.toolHandleDraftMessage(context.Background(), c, draftTool(tt.args))

			if res.Result != tt.wantResult {
				t.Fatalf("result = %q (%s), want %q", res.Result, res.Message, tt.wantResult)
			}
			if tt.wantErrPart != "" && !strings.Contains(res.Message, tt.wantErrPart) {
				t.Errorf("message = %q, want it to contain %q", res.Message, tt.wantErrPart)
			}
			if tt.wantBlock == nil {
				if len(res.Blocks) != 0 {
					t.Errorf("a failed draft must not carry blocks: %+v", res.Blocks)
				}
				return
			}
			if len(res.Blocks) != 1 || res.Blocks[0].Type != tt.wantBlock.Type || res.Blocks[0].Title != tt.wantBlock.Title ||
				res.Blocks[0].To != tt.wantBlock.To || res.Blocks[0].Text != tt.wantBlock.Text {
				t.Errorf("blocks = %+v, want [%+v]", res.Blocks, tt.wantBlock)
			}
			if res.Message != "Displayed an SMS draft for review." || res.ResourceType != "card" {
				t.Errorf("message/resource_type = %q/%q", res.Message, res.ResourceType)
			}
			if strings.Contains(res.Message, tt.wantBlock.Text) {
				t.Errorf("the result message must never contain the draft body")
			}
		})
	}
}

// the info card wire shape is unchanged by the two new CardBlock fields.
func Test_CardBlock_infoWireShapeUnchanged(t *testing.T) {
	b, err := json.Marshal(CardBlock{Type: "info", Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"type":"info","title":"t"}` {
		t.Errorf("info block json = %s", b)
	}
}

// through ToolHandle: draft_message records a card and sends nothing, its stored
// result carries the blocks (the frontend card), the LLM-facing result does not,
// and the tool-call arguments stay in the stored call row.
func Test_ToolHandle_draftMessage(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(nil)

	args := `{"to":"+14155550100","text":"Secret body 12345","title":"Hi"}`
	res, err := x.h.ToolHandle(context.Background(), x.c.ID, "tc-1", message.ToolTypeFunction, message.FunctionCall{
		Name:      message.FunctionCallNameDraftMessage,
		Arguments: args,
	}, uuid.Nil)
	if err != nil {
		t.Fatalf("ToolHandle() err = %v", err)
	}

	if atomic.LoadInt32(x.sends) != 0 {
		t.Errorf("draft_message must not send")
	}
	if _, ok := res["blocks"]; ok {
		t.Errorf("the LLM-facing result must not carry blocks: %v", res)
	}
	for _, v := range res {
		if s, ok := v.(string); ok && strings.Contains(s, "Secret body") {
			t.Errorf("the LLM-facing result must not contain the draft body: %v", res)
		}
	}
	if res["result"] != "success" || res["tool_call_id"] != "tc-1" {
		t.Errorf("result = %v", res)
	}

	var draftRow, callRow *message.Message
	for _, m := range x.rows() {
		switch m.Role {
		case message.RoleTool:
			draftRow = m
		case message.RoleAssistant:
			callRow = m
		}
	}
	if draftRow == nil || callRow == nil {
		t.Fatalf("expected a call row and a result row")
	}
	var stored messageContent
	_ = json.Unmarshal([]byte(draftRow.Content), &stored)
	if len(stored.Blocks) != 1 || stored.Blocks[0].Type != "sms_draft" || stored.Blocks[0].Text != "Secret body 12345" || stored.Blocks[0].To != "+14155550100" {
		t.Errorf("stored blocks = %+v", stored.Blocks)
	}
	// the call arguments are kept so the assistant remembers its own draft.
	if len(callRow.ToolCalls) != 1 || callRow.ToolCalls[0].Function.Arguments != args {
		t.Errorf("tool call arguments must be kept as is: %+v", callRow.ToolCalls)
	}

	// replay: blocks are removed, the arguments remain.
	msgs, err := x.h.getPipecatcallMessages(context.Background(), x.c)
	if err != nil {
		t.Fatal(err)
	}
	foundArgs := false
	for _, m := range msgs {
		if content, _ := m["content"].(string); strings.Contains(content, "blocks") || strings.Contains(content, "sms_draft") {
			t.Errorf("blocks must not be replayed: %s", content)
		}
		if tcs, ok := m["tool_calls"].([]message.ToolCall); ok && len(tcs) == 1 && tcs[0].Function.Arguments == args {
			foundArgs = true
		}
	}
	if !foundArgs {
		t.Errorf("the draft_message tool call arguments must stay in the replay")
	}

	// the stored draft can then be approved: end to end through the real rows.
	d := draftRow
	x.setNow(d.TMCreate.Add(5_000_000_000))
	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove); err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if atomic.LoadInt32(x.sends) != 1 {
		t.Errorf("sends = %d, want 1", atomic.LoadInt32(x.sends))
	}
}
