package aicallhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	mmmessage "monorepo/bin-message-manager/models/message"
)

// draftHarness wires the real message handler on a private in-memory sqlite
// database (a mocked message handler cannot reproduce a primary key conflict)
// with only the request handler, the util handler and the aicall store mocked.
// Nothing is ever sent: every send goes to the request handler mock.
type draftHarness struct {
	t *testing.T

	h     *aicallHandler
	msg   messagehandler.MessageHandler
	req   *requesthandler.MockRequestHandler
	sends *int32

	c *aicall.AIcall

	mu  sync.Mutex
	now time.Time
}

func (x *draftHarness) setNow(tm time.Time) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.now = tm
}

func newDraftHarness(t *testing.T) *draftHarness {
	t.Helper()
	config.SetAIcallDraftDecisionExpirySecondsForTest(600)

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	t.Cleanup(func() { config.SetAIcallDraftDecisionExpirySecondsForTest(0) })

	script, err := os.ReadFile("../../scripts/database_scripts_test/table_ai_messages.sql")
	if err != nil {
		t.Fatalf("could not read the test schema. err: %v", err)
	}
	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:draft_%s?mode=memory&cache=shared", strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")))
	if err != nil {
		t.Fatalf("could not open sqlite. err: %v", err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, errExec := conn.Exec(string(script)); errExec != nil {
		t.Fatalf("could not create the schema. err: %v", errExec)
	}

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, stderrors.New("cache miss")).AnyTimes()
	mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	realDB := dbhandler.NewHandler(conn, mockCache)

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockReq := requesthandler.NewMockRequestHandler(mc)
	msg := messagehandler.NewMessageHandler(mockReq, mockNotify, realDB, nil, nil, nil)

	c := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1000000-0000-11f0-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("d1000000-0000-11f0-0000-000000000002"),
		},
		ActiveflowID:   uuid.FromStringOrNil("d1000000-0000-11f0-0000-000000000003"),
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   uuid.FromStringOrNil("d1000000-0000-11f0-0000-000000000004"),
	}

	x := &draftHarness{t: t, msg: msg, req: mockReq, c: c, sends: new(int32), now: time.Now()}

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().AIcallGet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*aicall.AIcall, error) {
		if id == c.ID {
			return c, nil
		}
		return nil, dbhandler.ErrNotFound
	}).AnyTimes()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().DoAndReturn(func() *time.Time {
		x.mu.Lock()
		defer x.mu.Unlock()
		tm := x.now
		return &tm
	}).AnyTimes()
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("d1000000-0000-11f0-0000-0000000000ff")).AnyTimes()

	x.h = &aicallHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, messageHandler: msg}
	return x
}

// expectSend allows the send request any number of times, counts the calls and
// records the destination and text of the last one.
func (x *draftHarness) expectSend(result error) (*string, *string) {
	var to, text string
	var mu sync.Mutex
	x.req.EXPECT().MessageV1MessageSend(gomock.Any(), gomock.Any(), x.c.CustomerID, nil, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, id uuid.UUID, _ uuid.UUID, _ *commonaddress.Address, dests []commonaddress.Address, txt string) (*mmmessage.Message, error) {
			atomic.AddInt32(x.sends, 1)
			mu.Lock()
			if len(dests) == 1 {
				to = dests[0].Target
			}
			text = txt
			mu.Unlock()
			if result != nil {
				return nil, result
			}
			return &mmmessage.Message{Identity: commonidentity.Identity{ID: id}}, nil
		}).AnyTimes()
	return &to, &text
}

// newDraft stores a draft row the way ToolHandle does (a role=tool row with an sms_draft block).
func (x *draftHarness) newDraft(to, text string) *message.Message {
	x.t.Helper()
	mc := &messageContent{
		ToolCallID: "tc-draft",
		Result:     "success",
		Message:    "Displayed an SMS draft for review.",
		Blocks:     []CardBlock{{Type: message.BlockTypeSMSDraft, Title: "SMS draft", To: to, Text: text}},
	}
	content, _ := json.Marshal(mc)
	d, err := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(content), nil, "tc-draft")
	if err != nil {
		x.t.Fatalf("could not create the draft row. err: %v", err)
	}
	return d
}

func (x *draftHarness) rows() []*message.Message {
	x.t.Helper()
	res, err := x.msg.List(context.Background(), 100, "", map[message.Field]any{message.FieldAIcallID: x.c.ID})
	if err != nil {
		x.t.Fatalf("could not list rows. err: %v", err)
	}
	return res
}

func (x *draftHarness) countRole(role message.Role) int {
	n := 0
	for _, m := range x.rows() {
		if m.Role == role {
			n++
		}
	}
	return n
}

func reasonOf(err error) string {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		return ve.Reason
	}
	return ""
}

func Test_DraftDecision_approve(t *testing.T) {
	x := newDraftHarness(t)
	to, text := x.expectSend(nil)
	d := x.newDraft("+14155550100", "hello there")
	x.setNow(d.TMCreate.Add(30 * time.Second))

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}

	if got := atomic.LoadInt32(x.sends); got != 1 {
		t.Fatalf("send calls = %d, want 1", got)
	}
	if *to != "+14155550100" || *text != "hello there" {
		t.Errorf("sent to %q text %q, want the stored draft values", *to, *text)
	}

	wantID := "dd_" + strings.ReplaceAll(d.ID.String(), "-", "")
	if res.Role != message.RoleTool || res.ToolCallID != wantID {
		t.Errorf("result row role/tool_call_id = %s/%s, want tool/%s", res.Role, res.ToolCallID, wantID)
	}
	var mc messageContent
	if errUnmarshal := json.Unmarshal([]byte(res.Content), &mc); errUnmarshal != nil {
		t.Fatalf("result content: %v", errUnmarshal)
	}
	if mc.Result != "success" || mc.ResourceType != "message" {
		t.Errorf("result content = %+v, want success/message", mc)
	}

	// rows: draft tool row + claim notification + assistant tool call + tool result.
	if x.countRole(message.RoleNotification) != 1 || x.countRole(message.RoleAssistant) != 1 || x.countRole(message.RoleTool) != 2 {
		t.Errorf("unexpected row set: %d notification, %d assistant, %d tool", x.countRole(message.RoleNotification), x.countRole(message.RoleAssistant), x.countRole(message.RoleTool))
	}
	for _, m := range x.rows() {
		if m.Role == message.RoleAssistant {
			if len(m.ToolCalls) != 1 || m.ToolCalls[0].ID != wantID || m.ToolCalls[0].Function.Name != message.FunctionCallNameSendMessage {
				t.Errorf("assistant row tool calls = %+v", m.ToolCalls)
			}
			if len(m.ToolCalls[0].ID) != 35 {
				t.Errorf("synthetic tool call id length = %d, want 35", len(m.ToolCalls[0].ID))
			}
		}
		if m.Role == message.RoleNotification && m.ID != draftDecisionClaimID(d.ID) {
			t.Errorf("claim row id = %s, want %s", m.ID, draftDecisionClaimID(d.ID))
		}
	}
}

func Test_DraftDecision_reject(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(nil)
	d := x.newDraft("+14155550100", "hello there")
	x.setNow(d.TMCreate.Add(10 * time.Second))

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionReject)
	if err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if got := atomic.LoadInt32(x.sends); got != 0 {
		t.Fatalf("send calls = %d, want 0 for a rejection", got)
	}
	var mc messageContent
	_ = json.Unmarshal([]byte(res.Content), &mc)
	if mc.Result != "rejected" || mc.Message != draftRejectedMessage {
		t.Errorf("result content = %+v, want rejected", mc)
	}
	if x.countRole(message.RoleAssistant) != 1 || x.countRole(message.RoleNotification) != 1 {
		t.Errorf("a rejection must still write the claim and the paired rows")
	}
}

func Test_DraftDecision_secondDecisionRefused(t *testing.T) {
	tests := []struct {
		name        string
		first       message.DraftDecision
		second      message.DraftDecision
		wantSends   int32
		wantRejects int
	}{
		{"approve then reject", message.DraftDecisionApprove, message.DraftDecisionReject, 1, 0},
		{"reject then approve", message.DraftDecisionReject, message.DraftDecisionApprove, 0, 1},
		{"approve twice", message.DraftDecisionApprove, message.DraftDecisionApprove, 1, 0},
		{"reject twice", message.DraftDecisionReject, message.DraftDecisionReject, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			x.expectSend(nil)
			d := x.newDraft("+14155550100", "hi")
			x.setNow(d.TMCreate.Add(time.Second))

			if _, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, tt.first); err != nil {
				t.Fatalf("first decision err = %v", err)
			}
			rowsBefore := len(x.rows())
			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, tt.second)
			if reasonOf(err) != "DRAFT_ALREADY_DECIDED" {
				t.Fatalf("second decision err = %v, want DRAFT_ALREADY_DECIDED", err)
			}
			if got := atomic.LoadInt32(x.sends); got != tt.wantSends {
				t.Errorf("send calls = %d, want %d", got, tt.wantSends)
			}
			if len(x.rows()) != rowsBefore {
				t.Errorf("a refused second decision must not write rows")
			}
		})
	}
}

func Test_DraftDecision_concurrentApproveAndReject(t *testing.T) {
	for i := 0; i < 20; i++ {
		x := newDraftHarness(t)
		x.expectSend(nil)
		d := x.newDraft("+14155550100", "hi")
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
		if sends := atomic.LoadInt32(x.sends); sends > 1 {
			t.Fatalf("round %d: %d sends, want at most 1", i, sends)
		}
		if x.countRole(message.RoleNotification) != 1 || x.countRole(message.RoleAssistant) != 1 {
			t.Fatalf("round %d: the decision rows must exist exactly once", i)
		}
	}
}

func Test_DraftDecision_refusedBeforeClaim(t *testing.T) {
	infoContent, _ := json.Marshal(&messageContent{Result: "success", Blocks: []CardBlock{{Type: "info", Title: "t"}}})
	emptyToContent, _ := json.Marshal(&messageContent{Result: "success", Blocks: []CardBlock{{Type: message.BlockTypeSMSDraft, Title: "t", Text: "x"}}})
	badToContent, _ := json.Marshal(&messageContent{Result: "success", Blocks: []CardBlock{{Type: message.BlockTypeSMSDraft, Title: "t", To: "not-a-number", Text: "x"}}})

	otherCustomer := uuid.FromStringOrNil("d2000000-0000-11f0-0000-000000000001")
	otherAIcall := uuid.FromStringOrNil("d2000000-0000-11f0-0000-000000000002")

	tests := []struct {
		name       string
		prepare    func(x *draftHarness) (aicallID, draftID uuid.UUID)
		decision   message.DraftDecision
		wantReason string
	}{
		{
			name: "missing draft id",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				return x.c.ID, uuid.FromStringOrNil("d2000000-0000-11f0-0000-0000000000aa")
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "draft of another aicall",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newDraft("+14155550100", "hi")
				row, _ := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, otherAIcall, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, d.Content, nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "draft of another customer",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newDraft("+14155550100", "hi")
				row, _ := x.msg.Create(context.Background(), uuid.Nil, otherCustomer, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, d.Content, nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "not a tool row",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newDraft("+14155550100", "hi")
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
			name: "sms_draft block with an empty to",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				row, _ := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(emptyToContent), nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionReject, wantReason: "DRAFT_NOT_FOUND",
		},
		{
			name: "stored recipient fails the E.164 re-validation on approve",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				row, _ := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(badToContent), nil, "tc")
				return x.c.ID, row.ID
			},
			decision: message.DraftDecisionApprove, wantReason: "DRAFT_INVALID",
		},
		{
			name: "invalid decision value",
			prepare: func(x *draftHarness) (uuid.UUID, uuid.UUID) {
				d := x.newDraft("+14155550100", "hi")
				return x.c.ID, d.ID
			},
			decision: message.DraftDecision("maybe"), wantReason: "INVALID_DECISION",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			x.expectSend(nil)
			aicallID, draftID := tt.prepare(x)
			x.setNow(time.Now().Add(time.Second))

			_, err := x.h.DraftDecision(context.Background(), aicallID, draftID, tt.decision)
			if reasonOf(err) != tt.wantReason {
				t.Fatalf("err = %v, want reason %s", err, tt.wantReason)
			}
			if x.countRole(message.RoleNotification) != 0 {
				t.Errorf("a refused request must not write the claim row")
			}
			if atomic.LoadInt32(x.sends) != 0 {
				t.Errorf("a refused request must not send")
			}
		})
	}
}

// the missing id, a foreign aicall and a foreign customer answer with the same
// error so that draft ids cannot be probed.
func Test_DraftDecision_refusalsAreIdentical(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(nil)
	d := x.newDraft("+14155550100", "hi")
	foreign, _ := x.msg.Create(context.Background(), uuid.Nil, uuid.FromStringOrNil("d3000000-0000-11f0-0000-000000000001"), x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, d.Content, nil, "tc")
	x.setNow(time.Now().Add(time.Second))

	_, errMissing := x.h.DraftDecision(context.Background(), x.c.ID, uuid.FromStringOrNil("d3000000-0000-11f0-0000-0000000000aa"), message.DraftDecisionApprove)
	_, errForeign := x.h.DraftDecision(context.Background(), x.c.ID, foreign.ID, message.DraftDecisionApprove)

	var a, b *cerrors.VoipbinError
	if !stderrors.As(errMissing, &a) || !stderrors.As(errForeign, &b) {
		t.Fatalf("both refusals must be typed errors: %v / %v", errMissing, errForeign)
	}
	if a.Status != b.Status || a.Reason != b.Reason || a.Message != b.Message {
		t.Errorf("refusals differ: %+v vs %+v", a, b)
	}
}

func Test_DraftDecision_unknownAIcall(t *testing.T) {
	x := newDraftHarness(t)
	d := x.newDraft("+14155550100", "hi")
	_, err := x.h.DraftDecision(context.Background(), uuid.FromStringOrNil("d4000000-0000-11f0-0000-000000000001"), d.ID, message.DraftDecisionApprove)
	if err == nil {
		t.Fatalf("expected an error for an unknown aicall")
	}
}

func Test_DraftDecision_expiryBoundary(t *testing.T) {
	tests := []struct {
		name       string
		elapsed    time.Duration
		expirySec  int
		wantReason string
	}{
		{"599 seconds is accepted", 599 * time.Second, 600, ""},
		{"600 seconds is accepted", 600 * time.Second, 600, ""},
		{"601 seconds is refused", 601 * time.Second, 600, "DRAFT_EXPIRED"},
		{"a setting of zero falls back to 600 (599)", 599 * time.Second, 0, ""},
		{"a setting of zero falls back to 600 (601)", 601 * time.Second, 0, "DRAFT_EXPIRED"},
		{"a negative setting falls back to 600", 601 * time.Second, -5, "DRAFT_EXPIRED"},
		{"a configured shorter expiry applies", 31 * time.Second, 30, "DRAFT_EXPIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			x.expectSend(nil)
			config.SetAIcallDraftDecisionExpirySecondsForTest(tt.expirySec)
			d := x.newDraft("+14155550100", "hi")
			x.setNow(d.TMCreate.Add(tt.elapsed))

			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
			if reasonOf(err) != tt.wantReason || (tt.wantReason == "" && err != nil) {
				t.Fatalf("err = %v, want reason %q", err, tt.wantReason)
			}
			if tt.wantReason != "" {
				if x.countRole(message.RoleNotification) != 0 || atomic.LoadInt32(x.sends) != 0 {
					t.Errorf("an expired draft leaves no history row and sends nothing")
				}
			}
		})
	}
}

// an expired draft reports expired whether or not it was decided earlier.
func Test_DraftDecision_expiredBeforeAlreadyDecided(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(nil)
	d := x.newDraft("+14155550100", "hi")
	x.setNow(d.TMCreate.Add(5 * time.Second))
	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove); err != nil {
		t.Fatalf("first decision err = %v", err)
	}
	x.setNow(d.TMCreate.Add(700 * time.Second))
	_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionReject)
	if reasonOf(err) != "DRAFT_EXPIRED" {
		t.Fatalf("err = %v, want DRAFT_EXPIRED", err)
	}
}

func Test_DraftDecision_sendFailureIsRecorded(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(stderrors.New("rpc timeout"))
	d := x.newDraft("+14155550100", "hi")
	x.setNow(d.TMCreate.Add(time.Second))

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if err != nil {
		t.Fatalf("a send failure is a recorded result, not an error: %v", err)
	}
	var mc messageContent
	_ = json.Unmarshal([]byte(res.Content), &mc)
	if mc.Result != "failed" || !strings.Contains(mc.Message, "rpc timeout") || !strings.HasSuffix(mc.Message, draftDeliveryUnconfirmedSuffix) {
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
	if atomic.LoadInt32(x.sends) != 1 {
		t.Errorf("sends = %d, want 1", atomic.LoadInt32(x.sends))
	}
}

// a database error other than a duplicate key is an internal failure and must not look like a repeated decision.
func Test_DraftDecision_claimErrorKinds(t *testing.T) {
	tests := []struct {
		name        string
		claimErr    error
		wantReason  string
		wantTyped   bool
		wantDecided bool
	}{
		{"duplicate key", fmt.Errorf("MessageCreate: could not execute query. err: %w", stderrors.New("UNIQUE constraint failed: ai_messages.id")), "DRAFT_ALREADY_DECIDED", true, true},
		{"other database error", stderrors.New("connection reset"), "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			config.SetAIcallDraftDecisionExpirySecondsForTest(600)

			c := &aicall.AIcall{Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("d5000000-0000-11f0-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("d5000000-0000-11f0-0000-000000000002"),
			}}
			draftID := uuid.FromStringOrNil("d5000000-0000-11f0-0000-000000000003")
			tm := time.Now()
			content, _ := json.Marshal(&messageContent{Blocks: []CardBlock{{Type: message.BlockTypeSMSDraft, To: "+14155550100", Text: "x"}}})

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMsg := messagehandler.NewMockMessageHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &aicallHandler{db: mockDB, messageHandler: mockMsg, utilHandler: mockUtil}

			mockDB.EXPECT().AIcallGet(gomock.Any(), c.ID).Return(c, nil)
			mockMsg.EXPECT().Get(gomock.Any(), draftID).Return(&message.Message{
				Identity: commonidentity.Identity{ID: draftID, CustomerID: c.CustomerID}, AIcallID: c.ID, Role: message.RoleTool, Content: string(content), TMCreate: &tm,
			}, nil)
			mockUtil.EXPECT().TimeNow().Return(&tm)
			mockMsg.EXPECT().Create(gomock.Any(), draftDecisionClaimID(draftID), c.CustomerID, c.ID, gomock.Any(), message.DirectionOutgoing, message.RoleNotification, gomock.Any(), nil, "").Return(nil, tt.claimErr)

			_, err := h.DraftDecision(context.Background(), c.ID, draftID, message.DraftDecisionApprove)
			if err == nil {
				t.Fatalf("expected an error")
			}
			if reasonOf(err) != tt.wantReason {
				t.Errorf("reason = %q, want %q (err: %v)", reasonOf(err), tt.wantReason, err)
			}
		})
	}
}

func Test_draftDecisionClaimID(t *testing.T) {
	id := uuid.FromStringOrNil("d6000000-0000-11f0-0000-000000000001")
	other := uuid.FromStringOrNil("d6000000-0000-11f0-0000-000000000002")

	first := draftDecisionClaimID(id)
	if second := draftDecisionClaimID(id); first != second {
		t.Errorf("the claim id must be deterministic")
	}
	if draftDecisionClaimID(id) == draftDecisionClaimID(other) {
		t.Errorf("different drafts must have different claim ids")
	}
	if draftDecisionClaimID(id) == id {
		t.Errorf("the claim id must differ from the draft id")
	}
	// pinned value: changing the namespace would make earlier claims invisible.
	if got := draftDecisionClaimID(id).String(); got != uuid.NewV5(uuid.FromStringOrNil("2fcc8cd1-51ff-4100-886d-961edc8c1c54"), id.String()).String() {
		t.Errorf("claim id = %s does not match the pinned namespace", got)
	}
	if got, want := draftDecisionToolCallID(id), "dd_d6000000000011f00000000000000001"; got != want || len(got) != 35 {
		t.Errorf("tool call id = %q, want %q (35 characters)", got, want)
	}
}

// the claim row never reaches the LLM and the decision pair is adjacent in the replayed history.
func Test_DraftDecision_replay(t *testing.T) {
	x := newDraftHarness(t)
	x.expectSend(nil)
	d := x.newDraft("+14155550100", "hi")
	x.setNow(d.TMCreate.Add(time.Second))
	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove); err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}

	msgs, err := x.h.getPipecatcallMessages(context.Background(), x.c)
	if err != nil {
		t.Fatalf("getPipecatcallMessages() err = %v", err)
	}
	for _, m := range msgs {
		if m["role"] == string(message.RoleNotification) {
			t.Errorf("the claim row must not be replayed")
		}
		if content, _ := m["content"].(string); strings.Contains(content, "draft_decision") {
			t.Errorf("the claim content must not be replayed")
		}
		if content, _ := m["content"].(string); strings.Contains(content, "sms_draft") || strings.Contains(content, `"blocks"`) {
			t.Errorf("blocks must not be replayed: %s", content)
		}
	}

	wantID := "dd_" + strings.ReplaceAll(d.ID.String(), "-", "")
	callIdx, resIdx := -1, -1
	for i, m := range msgs {
		if tcs, ok := m["tool_calls"].([]message.ToolCall); ok && len(tcs) == 1 && tcs[0].ID == wantID {
			callIdx = i
		}
		if m["tool_call_id"] == wantID {
			resIdx = i
		}
	}
	if callIdx < 0 || resIdx != callIdx+1 {
		t.Errorf("the decision tool call (%d) and result (%d) must be adjacent in the replay", callIdx, resIdx)
	}
}

// when the pair cannot be written after the claim and the send, the caller gets an
// error, nothing is retried, and a repeated decision stays refused (the claim exists).
func Test_DraftDecision_pairWriteFailure(t *testing.T) {
	tests := []struct {
		name         string
		callRowErr   error
		resultRowErr error
	}{
		{"call row fails", stderrors.New("db down"), nil},
		{"result row fails", nil, stderrors.New("db down")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			x.expectSend(nil)
			d := x.newDraft("+14155550100", "hi")
			x.setNow(d.TMCreate.Add(time.Second))

			// wrap the real message handler: fail the chosen row of the pair.
			x.h.messageHandler = &failingPairMessageHandler{MessageHandler: x.msg, failRole: failRoleFor(tt.callRowErr), err: firstErr(tt.callRowErr, tt.resultRowErr)}

			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
			if err == nil {
				t.Fatalf("expected an error when the pair cannot be written")
			}
			if atomic.LoadInt32(x.sends) != 1 {
				t.Errorf("sends = %d, want 1 (no automatic resend)", atomic.LoadInt32(x.sends))
			}
			if x.countRole(message.RoleNotification) != 1 {
				t.Errorf("the claim row must remain")
			}

			// the draft stays decided.
			x.h.messageHandler = x.msg
			_, err = x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
			if reasonOf(err) != "DRAFT_ALREADY_DECIDED" {
				t.Errorf("repeat err = %v, want DRAFT_ALREADY_DECIDED", err)
			}
			if atomic.LoadInt32(x.sends) != 1 {
				t.Errorf("a repeat must not resend")
			}
		})
	}
}

func failRoleFor(callRowErr error) message.Role {
	if callRowErr != nil {
		return message.RoleAssistant
	}
	return message.RoleTool
}

func firstErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// failingPairMessageHandler fails Create for one role only.
type failingPairMessageHandler struct {
	messagehandler.MessageHandler
	failRole message.Role
	err      error
}

func (f *failingPairMessageHandler) Create(ctx context.Context, id uuid.UUID, customerID uuid.UUID, aicallID uuid.UUID, activeflowID uuid.UUID, direction message.Direction, role message.Role, content string, toolCalls []message.ToolCall, toolCallID string, opts ...messagehandler.CreateOption) (*message.Message, error) {
	if role == f.failRole {
		return nil, f.err
	}
	return f.MessageHandler.Create(ctx, id, customerID, aicallID, activeflowID, direction, role, content, toolCalls, toolCallID, opts...)
}

// a draft row without a creation time counts as expired (fail closed).
func Test_DraftDecision_missingCreateTimeIsExpired(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetAIcallDraftDecisionExpirySecondsForTest(600)

	c := &aicall.AIcall{Identity: commonidentity.Identity{
		ID:         uuid.FromStringOrNil("d5000000-0000-11f0-0000-000000000011"),
		CustomerID: uuid.FromStringOrNil("d5000000-0000-11f0-0000-000000000012"),
	}}
	draftID := uuid.FromStringOrNil("d5000000-0000-11f0-0000-000000000013")
	now := time.Now()
	content, _ := json.Marshal(&messageContent{Blocks: []CardBlock{{Type: message.BlockTypeSMSDraft, To: "+14155550100", Text: "x"}}})

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMsg := messagehandler.NewMockMessageHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &aicallHandler{db: mockDB, messageHandler: mockMsg, utilHandler: mockUtil}

	mockDB.EXPECT().AIcallGet(gomock.Any(), c.ID).Return(c, nil)
	mockMsg.EXPECT().Get(gomock.Any(), draftID).Return(&message.Message{
		Identity: commonidentity.Identity{ID: draftID, CustomerID: c.CustomerID}, AIcallID: c.ID, Role: message.RoleTool, Content: string(content), TMCreate: nil,
	}, nil)
	mockUtil.EXPECT().TimeNow().Return(&now)
	// no Create expectation: the claim must not be written.

	_, err := h.DraftDecision(context.Background(), c.ID, draftID, message.DraftDecisionApprove)
	if reasonOf(err) != "DRAFT_EXPIRED" {
		t.Errorf("err = %v, want DRAFT_EXPIRED", err)
	}
}
