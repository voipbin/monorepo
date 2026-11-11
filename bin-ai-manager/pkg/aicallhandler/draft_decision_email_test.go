package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/message"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	ememail "monorepo/bin-email-manager/models/email"
)

// emailSendCall is one recorded EmailV1EmailSend request.
type emailSendCall struct {
	customerID   uuid.UUID
	activeflowID uuid.UUID
	destinations []commonaddress.Address
	subject      string
	content      string
	attachments  []ememail.Attachment
}

// expectEmailSend allows the email send request any number of times and records every call, so
// that "exactly one" is asserted from the recorded calls. Nothing is ever sent.
func (x *draftHarness) expectEmailSend(result error) *[]emailSendCall {
	var mu sync.Mutex
	calls := []emailSendCall{}
	x.req.EXPECT().EmailV1EmailSend(gomock.Any(), x.c.CustomerID, x.c.ActiveflowID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, customerID, activeflowID uuid.UUID, dests []commonaddress.Address, subject, content string, attachments []ememail.Attachment) (*ememail.Email, error) {
			mu.Lock()
			calls = append(calls, emailSendCall{customerID: customerID, activeflowID: activeflowID, destinations: dests, subject: subject, content: content, attachments: attachments})
			mu.Unlock()
			atomic.AddInt32(x.sends, 1)
			if result != nil {
				return nil, result
			}
			return &ememail.Email{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("d9000000-0000-11f0-0000-0000000000e1")}}, nil
		}).AnyTimes()
	return &calls
}

// newEmailDraft stores a draft row the way ToolHandle does (a role=tool row with an email_draft block).
func (x *draftHarness) newEmailDraft(recipients []string, subject, content string) *message.Message {
	x.t.Helper()
	return x.newEmailDraftBlock(CardBlock{Type: message.BlockTypeEmailDraft, Title: "Email draft", Recipients: recipients, Subject: subject, Content: content})
}

func (x *draftHarness) newEmailDraftBlock(block CardBlock) *message.Message {
	x.t.Helper()
	mc := &messageContent{
		ToolCallID: "tc-draft",
		Result:     "success",
		Message:    "Displayed an email draft for review.",
		Blocks:     []CardBlock{block},
	}
	content, _ := json.Marshal(mc)
	d, err := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(content), nil, "tc-draft")
	if err != nil {
		x.t.Fatalf("could not create the draft row. err: %v", err)
	}
	return d
}

func messageOf(err error) string {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		return ve.Message
	}
	return ""
}

func decodeContent(t *testing.T, m *message.Message) messageContent {
	t.Helper()
	var mc messageContent
	if err := json.Unmarshal([]byte(m.Content), &mc); err != nil {
		t.Fatalf("content of %s: %v", m.ID, err)
	}
	return mc
}

func Test_DraftDecision_emailApprove(t *testing.T) {
	x := newDraftHarness(t)
	calls := x.expectEmailSend(nil)
	d := x.newEmailDraft([]string{"a@example.com", "b@example.org"}, "Subject <1> & more", "Line 1\nLine 2 <b>x</b>")
	x.setNow(d.TMCreate.Add(30 * time.Second))

	before := testutil.ToFloat64(promAIcallToolExecuteTotal.WithLabelValues("send_email"))
	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if after := testutil.ToFloat64(promAIcallToolExecuteTotal.WithLabelValues("send_email")); after != before+1 {
		t.Errorf("send_email execute metric went from %v to %v, want +1", before, after)
	}

	if len(*calls) != 1 {
		t.Fatalf("send calls = %d, want exactly 1", len(*calls))
	}
	call := (*calls)[0]
	if call.customerID != x.c.CustomerID || call.activeflowID != x.c.ActiveflowID {
		t.Errorf("customer/activeflow = %s/%s", call.customerID, call.activeflowID)
	}
	if len(call.destinations) != 2 ||
		call.destinations[0] != (commonaddress.Address{Type: commonaddress.TypeEmail, Target: "a@example.com"}) ||
		call.destinations[1] != (commonaddress.Address{Type: commonaddress.TypeEmail, Target: "b@example.org"}) {
		t.Errorf("destinations = %+v, want two email targets without a name", call.destinations)
	}
	if call.subject != "Subject <1> & more" || call.content != "Line 1\nLine 2 <b>x</b>" {
		t.Errorf("subject/content = %q/%q, want the stored values verbatim", call.subject, call.content)
	}
	if len(call.attachments) != 0 {
		t.Errorf("attachments = %+v, want none", call.attachments)
	}

	wantID := "dd_" + strings.ReplaceAll(d.ID.String(), "-", "")
	if res.Role != message.RoleTool || res.ToolCallID != wantID {
		t.Errorf("result row role/tool_call_id = %s/%s, want tool/%s", res.Role, res.ToolCallID, wantID)
	}
	mc := decodeContent(t, res)
	if mc.Result != "success" || mc.ResourceType != "email" || mc.ResourceID != "d9000000-0000-11f0-0000-0000000000e1" {
		t.Errorf("result content = %+v, want success/email with the email id", mc)
	}
	if mc.Message != "Email accepted for sending. Delivery is not confirmed." {
		t.Errorf("result message = %q", mc.Message)
	}

	// rows: draft tool row + claim notification + assistant tool call + tool result.
	if x.countRole(message.RoleNotification) != 1 || x.countRole(message.RoleAssistant) != 1 || x.countRole(message.RoleTool) != 2 {
		t.Errorf("unexpected row set: %d notification, %d assistant, %d tool", x.countRole(message.RoleNotification), x.countRole(message.RoleAssistant), x.countRole(message.RoleTool))
	}
	for _, m := range x.rows() {
		switch m.Role {
		case message.RoleAssistant:
			if len(m.ToolCalls) != 1 || m.ToolCalls[0].ID != wantID || m.ToolCalls[0].Function.Name != message.FunctionCallNameSendEmail || len(m.ToolCalls[0].ID) != 35 {
				t.Errorf("assistant row tool calls = %+v", m.ToolCalls)
			}
			var opt fmaction.OptionEmailSend
			if errUnmarshal := json.Unmarshal([]byte(m.ToolCalls[0].Function.Arguments), &opt); errUnmarshal != nil {
				t.Fatalf("the call arguments must decode as OptionEmailSend: %v", errUnmarshal)
			}
			if len(opt.Destinations) != 2 || opt.Subject != "Subject <1> & more" || opt.Content != "Line 1\nLine 2 <b>x</b>" || len(opt.Attachments) != 0 {
				t.Errorf("call arguments = %+v", opt)
			}
			if strings.Contains(m.ToolCalls[0].Function.Arguments, `"attachments"`) || strings.Contains(m.ToolCalls[0].Function.Arguments, `"target_name"`) {
				t.Errorf("the arguments must hold no attachments and no target name: %s", m.ToolCalls[0].Function.Arguments)
			}
		case message.RoleNotification:
			if m.ID != draftDecisionClaimID(d.ID) {
				t.Errorf("claim row id = %s, want %s", m.ID, draftDecisionClaimID(d.ID))
			}
		}
	}
}

func Test_DraftDecision_emailApproveSendFailure(t *testing.T) {
	x := newDraftHarness(t)
	calls := x.expectEmailSend(stderrors.New("customer identity verification required to send email"))
	d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
	x.setNow(d.TMCreate.Add(time.Second))

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if err != nil {
		t.Fatalf("a send failure is a recorded result, not an error: %v", err)
	}
	mc := decodeContent(t, res)
	if mc.Result != "failed" || mc.Message != "customer identity verification required to send email Delivery may be unconfirmed." || mc.ResourceType != "" || mc.ResourceID != "" {
		t.Errorf("result content = %+v", mc)
	}
	if x.countRole(message.RoleAssistant) != 1 {
		t.Errorf("the failed attempt still writes the pair")
	}

	// the same draft cannot be retried.
	_, err = x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if reasonOf(err) != "DRAFT_ALREADY_DECIDED" {
		t.Errorf("retry err = %v, want DRAFT_ALREADY_DECIDED", err)
	}
	if len(*calls) != 1 {
		t.Errorf("sends = %d, want 1", len(*calls))
	}
}

func Test_DraftDecision_emailReject(t *testing.T) {
	x := newDraftHarness(t)
	calls := x.expectEmailSend(nil)
	d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
	x.setNow(d.TMCreate.Add(10 * time.Second))

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionReject)
	if err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("send calls = %d, want 0 for a rejection", len(*calls))
	}
	mc := decodeContent(t, res)
	if mc.Result != "rejected" || mc.Message != "The agent rejected this draft. The email was not sent." {
		t.Errorf("result content = %+v, want rejected", mc)
	}
	if x.countRole(message.RoleAssistant) != 1 || x.countRole(message.RoleNotification) != 1 {
		t.Errorf("a rejection must still write the claim and the paired rows")
	}
	for _, m := range x.rows() {
		if m.Role == message.RoleAssistant && (len(m.ToolCalls) != 1 || m.ToolCalls[0].Function.Name != message.FunctionCallNameSendEmail) {
			t.Errorf("the call row of an email rejection names send_email: %+v", m.ToolCalls)
		}
	}
}

func Test_DraftDecision_emailSecondDecisionRefused(t *testing.T) {
	tests := []struct {
		name      string
		first     message.DraftDecision
		second    message.DraftDecision
		wantSends int
	}{
		{"approve then reject", message.DraftDecisionApprove, message.DraftDecisionReject, 1},
		{"reject then approve", message.DraftDecisionReject, message.DraftDecisionApprove, 0},
		{"approve twice", message.DraftDecisionApprove, message.DraftDecisionApprove, 1},
		{"reject twice", message.DraftDecisionReject, message.DraftDecisionReject, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			calls := x.expectEmailSend(nil)
			d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
			x.setNow(d.TMCreate.Add(time.Second))

			if _, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, tt.first); err != nil {
				t.Fatalf("first decision err = %v", err)
			}
			rowsBefore := len(x.rows())
			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, tt.second)
			if reasonOf(err) != "DRAFT_ALREADY_DECIDED" {
				t.Fatalf("second decision err = %v, want DRAFT_ALREADY_DECIDED", err)
			}
			if len(*calls) != tt.wantSends {
				t.Errorf("send calls = %d, want %d", len(*calls), tt.wantSends)
			}
			if len(x.rows()) != rowsBefore {
				t.Errorf("a refused second decision must not write rows")
			}
		})
	}
}

func Test_DraftDecision_emailConcurrentApproveAndReject(t *testing.T) {
	for i := 0; i < 20; i++ {
		x := newDraftHarness(t)
		calls := x.expectEmailSend(nil)
		d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
		x.setNow(d.TMCreate.Add(time.Second))

		var wg sync.WaitGroup
		var ok, conflicts int32
		for _, dec := range []message.DraftDecision{message.DraftDecisionApprove, message.DraftDecisionReject, message.DraftDecisionApprove, message.DraftDecisionReject} {
			wg.Add(1)
			go func(dec message.DraftDecision) {
				defer wg.Done()
				_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, dec)
				switch {
				case err == nil:
					atomic.AddInt32(&ok, 1)
				case reasonOf(err) == "DRAFT_ALREADY_DECIDED":
					atomic.AddInt32(&conflicts, 1)
				default:
					t.Errorf("unexpected error: %v", err)
				}
			}(dec)
		}
		wg.Wait()

		if ok != 1 || conflicts != 3 {
			t.Fatalf("round %d: %d succeeded and %d conflicted, want 1 and 3", i, ok, conflicts)
		}
		if len(*calls) > 1 {
			t.Fatalf("round %d: %d sends, want at most 1", i, len(*calls))
		}
		if x.countRole(message.RoleNotification) != 1 || x.countRole(message.RoleAssistant) != 1 {
			t.Fatalf("round %d: the decision rows must exist exactly once", i)
		}
	}
}

func Test_DraftDecision_emailRefusedBeforeClaim(t *testing.T) {
	otherCustomer := uuid.FromStringOrNil("da000000-0000-11f0-0000-000000000001")
	otherAIcall := uuid.FromStringOrNil("da000000-0000-11f0-0000-000000000002")
	infoContent, _ := json.Marshal(&messageContent{Result: "success", Blocks: []CardBlock{{Type: "info", Title: "t"}}})

	manyRecipients := []string{}
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"} {
		manyRecipients = append(manyRecipients, n+"@example.com")
	}

	storeBlock := func(x *draftHarness, block CardBlock) (uuid.UUID, uuid.UUID) {
		row := x.newEmailDraftBlock(block)
		return x.c.ID, row.ID
	}

	tests := []struct {
		name        string
		prepare     func(x *draftHarness) (aicallID, draftID uuid.UUID)
		decision    message.DraftDecision
		wantReason  string
		wantMessage string
	}{
		{
			name: "missing draft id",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return x.c.ID, uuid.FromStringOrNil("da000000-0000-11f0-0000-0000000000aa")
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "draft of another aicall",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
				row, _ := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, otherAIcall, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, d.Content, nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "draft of another customer",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
				row, _ := x.msg.Create(context.Background(), uuid.Nil, otherCustomer, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, d.Content, nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "not a tool row",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
				row, _ := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleAssistant, d.Content, nil, "")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "tool row with an info block",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				row, _ := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(infoContent), nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "email_draft block without recipients (approve)",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Title: "t", Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "email_draft block without recipients (reject)",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Title: "t", Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionReject, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "email_draft block with only the sms field to",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Title: "t", To: "+141****0100", Text: "x"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "sms_draft block with only the email fields",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeSMSDraft, Title: "t", Recipients: []string{"a@example.com"}, Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "stored recipient is malformed",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"not-an-address"}, Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID", wantMessage: "The draft recipients, subject or content are not valid.",
		},
		{
			name: "stored recipient is not in normalized form",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"A@Example.com"}, Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "stored recipient carries a display name",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"boss <b@example.com>"}, Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "stored recipients repeat",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"a@example.com", "a@example.com"}, Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "stored draft has too many recipients",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: manyRecipients, Subject: "S", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "stored subject is blank",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"a@example.com"}, Subject: " ", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "stored subject has a line break",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"a@example.com"}, Subject: "a\nb", Content: "C"})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "stored content is over the limit",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return storeBlock(x, CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"a@example.com"}, Subject: "S", Content: strings.Repeat("a", 10001)})
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "invalid decision value",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
				return x.c.ID, d.ID
			},
			decision: message.DraftDecision("maybe"), wantReason: "INVALID_DECISION",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			calls := x.expectEmailSend(nil)
			aicallID, draftID := tt.prepare(x)
			x.setNow(time.Now().Add(time.Second))

			_, err := x.h.DraftDecision(context.Background(), aicallID, draftID, tt.decision)
			if reasonOf(err) != tt.wantReason {
				t.Fatalf("err = %v, want reason %s", err, tt.wantReason)
			}
			if tt.wantMessage != "" && messageOf(err) != tt.wantMessage {
				t.Errorf("message = %q, want %q", messageOf(err), tt.wantMessage)
			}
			if x.countRole(message.RoleNotification) != 0 {
				t.Errorf("a refused request must not write the claim row")
			}
			if len(*calls) != 0 {
				t.Errorf("a refused request must not send")
			}
		})
	}
}

// a draft that fails the re-validation stays pending: the approve is refused before the claim and
// the agent can still reject it; reject does not re-validate.
func Test_DraftDecision_emailInvalidDraftCanStillBeRejected(t *testing.T) {
	x := newDraftHarness(t)
	calls := x.expectEmailSend(nil)
	row := x.newEmailDraftBlock(CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"not-an-address"}, Subject: "S", Content: "C"})
	x.setNow(row.TMCreate.Add(time.Second))

	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, row.ID, message.DraftDecisionApprove); reasonOf(err) != "DRAFT_INVALID" {
		t.Fatalf("approve err = %v, want DRAFT_INVALID", err)
	}
	if x.countRole(message.RoleNotification) != 0 {
		t.Fatalf("the refused approve must not write a claim")
	}

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, row.ID, message.DraftDecisionReject)
	if err != nil {
		t.Fatalf("reject err = %v", err)
	}
	if mc := decodeContent(t, res); mc.Result != "rejected" {
		t.Errorf("reject result = %+v", mc)
	}
	if len(*calls) != 0 {
		t.Errorf("nothing is sent")
	}
}

// an expired draft that also fails validation answers DRAFT_INVALID on approve (re-validation runs
// before the expiry check) and DRAFT_EXPIRED on reject.
func Test_DraftDecision_emailExpiredAndInvalid(t *testing.T) {
	for _, tt := range []struct {
		decision   message.DraftDecision
		wantReason string
	}{
		{message.DraftDecisionApprove, "DRAFT_INVALID"},
		{message.DraftDecisionReject, "DRAFT_EXPIRED"},
	} {
		t.Run(string(tt.decision), func(t *testing.T) {
			x := newDraftHarness(t)
			x.expectEmailSend(nil)
			row := x.newEmailDraftBlock(CardBlock{Type: message.BlockTypeEmailDraft, Recipients: []string{"not-an-address"}, Subject: "S", Content: "C"})
			x.setNow(row.TMCreate.Add(700 * time.Second))

			_, err := x.h.DraftDecision(context.Background(), x.c.ID, row.ID, tt.decision)
			if reasonOf(err) != tt.wantReason {
				t.Fatalf("err = %v, want %s", err, tt.wantReason)
			}
		})
	}
}

func Test_DraftDecision_emailExpiryBoundary(t *testing.T) {
	tests := []struct {
		name       string
		elapsed    time.Duration
		expirySec  int
		wantReason string
	}{
		{"599 seconds is accepted", 599 * time.Second, 600, ""},
		{"600 seconds is accepted", 600 * time.Second, 600, ""},
		{"601 seconds is refused", 601 * time.Second, 600, "DRAFT_EXPIRED"},
		{"a setting of zero falls back to 600 (601)", 601 * time.Second, 0, "DRAFT_EXPIRED"},
		{"a configured shorter expiry applies", 31 * time.Second, 30, "DRAFT_EXPIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			calls := x.expectEmailSend(nil)
			config.SetAIcallDraftDecisionExpirySecondsForTest(tt.expirySec)
			d := x.newEmailDraft([]string{"a@example.com"}, "S", "C")
			x.setNow(d.TMCreate.Add(tt.elapsed))

			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
			if reasonOf(err) != tt.wantReason || (tt.wantReason == "" && err != nil) {
				t.Fatalf("err = %v, want reason %q", err, tt.wantReason)
			}
			if tt.wantReason != "" && (x.countRole(message.RoleNotification) != 0 || len(*calls) != 0) {
				t.Errorf("an expired draft leaves no history row and sends nothing")
			}
		})
	}
}

// the SMS texts are fixed as literals so that a wording change in the shared code fails a test
// (the SMS regression of the email extension).
func Test_DraftDecision_smsTextsUnchanged(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(nil)

	bad := x.newDraft("not-a-number", "x")
	x.setNow(bad.TMCreate.Add(time.Second))
	_, err := x.h.DraftDecision(context.Background(), x.c.ID, bad.ID, message.DraftDecisionApprove)
	if reasonOf(err) != "DRAFT_INVALID" || messageOf(err) != "The draft recipient is not a valid phone number." {
		t.Errorf("sms DRAFT_INVALID = %q / %q", reasonOf(err), messageOf(err))
	}

	d := x.newDraft("+141****0100", "x")
	x.setNow(d.TMCreate.Add(time.Second))
	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionReject)
	if err != nil {
		t.Fatal(err)
	}
	if mc := decodeContent(t, res); mc.Message != "The agent rejected this draft. The message was not sent." {
		t.Errorf("sms rejection message = %q", mc.Message)
	}
	for _, m := range x.rows() {
		if m.Role == message.RoleAssistant && (len(m.ToolCalls) != 1 || m.ToolCalls[0].Function.Name != message.FunctionCallNameSendMessage) {
			t.Errorf("the call row of an sms rejection names send_message: %+v", m.ToolCalls)
		}
	}

	if draftDeliveryUnconfirmedSuffix != " Delivery may be unconfirmed." {
		t.Errorf("draftDeliveryUnconfirmedSuffix = %q", draftDeliveryUnconfirmedSuffix)
	}
}
