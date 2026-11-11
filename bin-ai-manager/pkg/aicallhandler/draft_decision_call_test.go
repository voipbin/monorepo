package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
)

// callCreateRecord is one recorded CallV1CallsCreate request. Nothing is ever dialed: the
// request handler is a mock.
type callCreateRecord struct {
	customerID   uuid.UUID
	flowID       uuid.UUID
	masterCallID uuid.UUID
	source       *commonaddress.Address
	destinations []commonaddress.Address
	anonymous    string
	variables    map[string]string
}

func (x *draftHarness) expectCallCreate(result error) *[]callCreateRecord {
	var mu sync.Mutex
	calls := []callCreateRecord{}
	x.req.EXPECT().CallV1CallsCreate(gomock.Any(), x.c.CustomerID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false, false, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, customerID, flowID, master uuid.UUID, src *commonaddress.Address, dests []commonaddress.Address, _ bool, _ bool, anonymous string, _ map[string]interface{}, vars map[string]string) ([]*cmcall.Call, []*cmgroupcall.Groupcall, error) {
			mu.Lock()
			calls = append(calls, callCreateRecord{customerID: customerID, flowID: flowID, masterCallID: master, source: src, destinations: dests, anonymous: anonymous, variables: vars})
			mu.Unlock()
			if result != nil {
				return nil, nil, result
			}
			return []*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("d9000000-0000-11f0-0000-0000000000c1")}}}, nil, nil
		}).AnyTimes()
	return &calls
}

// flowState is the flow the mock returns; a test changes it between decisions.
type flowState struct {
	mu   sync.Mutex
	flow *fmflow.Flow
	err  error
	gets int
}

func (s *flowState) set(f *fmflow.Flow, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flow, s.err = f, err
}

func (s *flowState) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gets
}

func (x *draftHarness) expectFlowGet(f *fmflow.Flow) *flowState {
	s := &flowState{flow: f}
	x.req.EXPECT().FlowV1FlowGet(gomock.Any(), uuid.FromStringOrNil(testCallFlowID)).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*fmflow.Flow, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.gets++
			return s.flow, s.err
		}).AnyTimes()
	return s
}

func (x *draftHarness) newCallDraft(d *CallDraft) *message.Message {
	x.t.Helper()
	return x.newEmailDraftBlock(CardBlock{Type: message.BlockTypeCallDraft, Title: "Call draft", Call: d})
}

func flowCallDraft(tmUpdate string) *CallDraft {
	persistent := true
	return &CallDraft{
		Destinations:    []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550101", TargetName: "John"}},
		Source:          &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550199"},
		Anonymous:       "no",
		Variables:       map[string]string{"campaign": "123"},
		FlowID:          testCallFlowID,
		FlowName:        "Reminder flow",
		FlowPersistent:  &persistent,
		FlowActionTypes: []string{"talk"},
		FlowTMUpdate:    tmUpdate,
	}
}

func decisionFlow(x *draftHarness, tm *time.Time) *fmflow.Flow {
	return &fmflow.Flow{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(testCallFlowID), CustomerID: x.c.CustomerID},
		Persist:  true,
		TMUpdate: tm,
	}
}

func Test_DraftDecision_callApproveFlow(t *testing.T) {
	x := newDraftHarness(t)
	calls := x.expectCallCreate(nil)
	tm := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	flows := x.expectFlowGet(decisionFlow(x, &tm))
	d := x.newCallDraft(flowCallDraft("2026-11-11T10:00:00Z"))
	x.setNow(d.TMCreate.Add(30 * time.Second))

	before := testutil.ToFloat64(promAIcallToolExecuteTotal.WithLabelValues("create_call"))
	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if after := testutil.ToFloat64(promAIcallToolExecuteTotal.WithLabelValues("create_call")); after != before+1 {
		t.Errorf("create_call execute metric went from %v to %v, want +1", before, after)
	}
	if len(*calls) != 1 {
		t.Fatalf("CallV1CallsCreate calls = %d, want exactly 1", len(*calls))
	}
	call := (*calls)[0]
	if call.flowID != uuid.FromStringOrNil(testCallFlowID) || call.masterCallID != uuid.Nil {
		t.Errorf("flow/master = %s/%s, want the stored flow and no master call", call.flowID, call.masterCallID)
	}
	if len(call.destinations) != 1 || call.destinations[0].Target != "+15550101" || call.destinations[0].TargetName != "John" {
		t.Errorf("destinations = %+v", call.destinations)
	}
	if call.source == nil || call.source.Target != "+15550199" || call.anonymous != "no" || call.variables["campaign"] != "123" {
		t.Errorf("source/anonymous/variables = %+v/%q/%v", call.source, call.anonymous, call.variables)
	}
	if flows.count() != 2 {
		t.Errorf("FlowV1FlowGet calls = %d, want 2 (the binding read and the one inside create_call)", flows.count())
	}

	wantID := "dd_" + strings.ReplaceAll(d.ID.String(), "-", "")
	if res.Role != message.RoleTool || res.ToolCallID != wantID {
		t.Errorf("result row role/tool_call_id = %s/%s, want tool/%s", res.Role, res.ToolCallID, wantID)
	}
	mc := decodeContent(t, res)
	if mc.Result != "success" || mc.ResourceType != "call" || mc.ResourceID != "d9000000-0000-11f0-0000-0000000000c1" {
		t.Errorf("result content = %+v, want success/call with the call id", mc)
	}
	if !strings.Contains(mc.Message, `"requested":1`) || !strings.Contains(mc.Message, `"created":1`) || strings.Contains(mc.Message, "unconfirmed") {
		t.Errorf("result message = %q, want the handler JSON unchanged", mc.Message)
	}

	// the same draft cannot be decided twice.
	_, err = x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if messageOf(err) != "The draft was already decided." {
		t.Errorf("second decision err = %v, want DRAFT_ALREADY_DECIDED", err)
	}
	if len(*calls) != 1 {
		t.Errorf("calls after a second decision = %d, want still 1", len(*calls))
	}
}

func Test_DraftDecision_callApproveActions(t *testing.T) {
	x := newDraftHarness(t)
	calls := x.expectCallCreate(nil)

	ephemeralID := uuid.FromStringOrNil("d9000000-0000-11f0-0000-0000000000e9")
	var gotActions []fmaction.Action
	x.req.EXPECT().FlowV1FlowCreate(gomock.Any(), x.c.CustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ fmflow.Type, _, _ string, actions []fmaction.Action, _ uuid.UUID, _ bool) (*fmflow.Flow, error) {
			gotActions = actions
			return &fmflow.Flow{Identity: commonidentity.Identity{ID: ephemeralID}}, nil
		}).Times(1)

	// the stored row holds a number above 2^53 and a decimal; both must reach flow-manager unchanged.
	content := `{"tool_call_id":"tc-draft","result":"success","message":"Displayed a call draft for review.","blocks":[{"type":"call_draft","title":"Call draft","call":{` +
		`"destinations":[{"type":"tel","target":"+15550101"}],"actions":[{"type":"talk","option":{"text":"Hi","n":9007199254740993,"f":1.0}},{"type":"hangup"}]}}]}`
	row, err := x.msg.Create(context.Background(), uuid.Nil, x.c.CustomerID, x.c.ID, x.c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, content, nil, "tc-draft")
	if err != nil {
		t.Fatal(err)
	}
	x.setNow(row.TMCreate.Add(time.Second))

	if _, err := x.h.DraftDecision(context.Background(), x.c.ID, row.ID, message.DraftDecisionApprove); err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	if len(*calls) != 1 || (*calls)[0].flowID != ephemeralID {
		t.Fatalf("calls = %+v, want one call on the ephemeral flow", *calls)
	}
	b, _ := json.Marshal(gotActions)
	if !strings.Contains(string(b), `"n":9007199254740993`) || !strings.Contains(string(b), `"f":1.0`) {
		t.Errorf("actions sent to flow-manager = %s, want the digits of the stored numbers", b)
	}
}

func Test_DraftDecision_callReject(t *testing.T) {
	x := newDraftHarness(t)
	// no expectation for CallV1CallsCreate, FlowV1FlowGet or FlowV1FlowCreate: a reject places and reads nothing.
	d := x.newCallDraft(flowCallDraft(""))
	x.setNow(d.TMCreate.Add(time.Second))

	res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionReject)
	if err != nil {
		t.Fatalf("DraftDecision() err = %v", err)
	}
	mc := decodeContent(t, res)
	if mc.Result != "rejected" || mc.Message != "The agent rejected this draft. The call was not placed." {
		t.Errorf("result = %q %q", mc.Result, mc.Message)
	}
}

func Test_DraftDecision_callExpiredReadsNoFlow(t *testing.T) {
	x := newDraftHarness(t)
	// no FlowV1FlowGet expectation: an expired draft must not cost a flow read.
	d := x.newCallDraft(flowCallDraft(""))
	x.setNow(d.TMCreate.Add(601 * time.Second))

	_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
	if messageOf(err) != "The draft expired and can no longer be decided." {
		t.Errorf("err = %v, want DRAFT_EXPIRED", err)
	}
}

func Test_DraftDecision_callFlowBinding(t *testing.T) {
	tm := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	changed := tm.Add(time.Second)

	tests := []struct {
		name       string
		draftTM    string
		flow       func(x *draftHarness) *fmflow.Flow
		flowErr    error
		wantReason string // empty means the approval goes through
		wantTrans  bool   // a transient read error: not DRAFT_INVALID
	}{
		{"same tm_update", "2026-11-11T10:00:00Z", func(x *draftHarness) *fmflow.Flow { return decisionFlow(x, &tm) }, nil, "", false},
		{"temporary flow nil against nil", "", func(x *draftHarness) *fmflow.Flow { return decisionFlow(x, nil) }, nil, "", false},
		{"tm_update changed", "2026-11-11T10:00:00Z", func(x *draftHarness) *fmflow.Flow { return decisionFlow(x, &changed) }, nil, draftCallInvalidMessage, false},
		{"nil against non-nil", "", func(x *draftHarness) *fmflow.Flow { return decisionFlow(x, &tm) }, nil, draftCallInvalidMessage, false},
		{"non-nil against nil", "2026-11-11T10:00:00Z", func(x *draftHarness) *fmflow.Flow { return decisionFlow(x, nil) }, nil, draftCallInvalidMessage, false},
		{"unparsable stored tm_update", "yesterday", func(x *draftHarness) *fmflow.Flow { return decisionFlow(x, &tm) }, nil, draftCallInvalidMessage, false},
		{"flow of another customer", "", func(x *draftHarness) *fmflow.Flow {
			f := decisionFlow(x, nil)
			f.CustomerID = uuid.FromStringOrNil("d7000000-0000-11f0-0000-0000000000aa")
			return f
		}, nil, draftCallInvalidMessage, false},
		{"flow not found", "", func(x *draftHarness) *fmflow.Flow { return nil }, cerrors.NotFound("flow-manager", "FLOW_NOT_FOUND", "The flow was not found."), draftCallInvalidMessage, false},
		{"transient read error", "", func(x *draftHarness) *fmflow.Flow { return nil }, stderrors.New("timeout"), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			calls := x.expectCallCreate(nil)
			flows := x.expectFlowGet(tt.flow(x))
			flows.set(tt.flow(x), tt.flowErr)
			d := x.newCallDraft(flowCallDraft(tt.draftTM))
			x.setNow(d.TMCreate.Add(time.Second))

			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
			switch {
			case tt.wantReason == "" && !tt.wantTrans:
				if err != nil || len(*calls) != 1 {
					t.Fatalf("err = %v, calls = %d, want an approved call", err, len(*calls))
				}
				return
			case tt.wantTrans:
				if err == nil || messageOf(err) != "" {
					t.Fatalf("err = %v, want a plain error and not a VoipbinError refusal", err)
				}
			default:
				if messageOf(err) != tt.wantReason {
					t.Fatalf("err = %v, want DRAFT_INVALID %q", err, tt.wantReason)
				}
			}
			if len(*calls) != 0 {
				t.Errorf("calls = %d, want none", len(*calls))
			}

			// the draft was not consumed: with the flow as shown the same draft can still be approved.
			good := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
			ok := decisionFlow(x, &good)
			if tt.draftTM == "" {
				ok.TMUpdate = nil
			}
			flows.set(ok, nil)
			if tt.draftTM == "yesterday" {
				return // the stored value itself is bad; it can never match.
			}
			if _, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove); err != nil {
				t.Errorf("approve after the refusal: err = %v, want the draft not to be consumed", err)
			}
		})
	}
}

func Test_DraftDecision_callTamperedRow(t *testing.T) {
	tests := []struct {
		name string
		d    *CallDraft
	}{
		{"flow and actions together", func() *CallDraft {
			d := flowCallDraft("")
			d.Actions = []fmaction.Action{{Type: fmaction.TypeHangup}}
			return d
		}()},
		{"no destination", func() *CallDraft { d := flowCallDraft(""); d.Destinations = nil; return d }()},
		{"unknown action type", &CallDraft{Destinations: flowCallDraft("").Destinations, Actions: []fmaction.Action{{Type: "launch_missiles"}}}},
		{"reserved variable", func() *CallDraft { d := flowCallDraft(""); d.Variables = map[string]string{"voipbin.x": "1"}; return d }()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newDraftHarness(t)
			// no request expectation: a tampered row is refused before any read or call.
			d := x.newCallDraft(tt.d)
			x.setNow(d.TMCreate.Add(time.Second))
			_, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
			if messageOf(err) != draftCallInvalidMessage {
				t.Errorf("err = %v, want DRAFT_INVALID", err)
			}
		})
	}
}

func Test_DraftDecision_callFailures(t *testing.T) {
	t.Run("every destination failed", func(t *testing.T) {
		x := newDraftHarness(t)
		calls := x.expectCallCreate(stderrors.New("all destinations failed to create outgoing calls"))
		x.expectFlowGet(decisionFlow(x, nil))
		d := x.newCallDraft(flowCallDraft(""))
		x.setNow(d.TMCreate.Add(time.Second))

		res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		mc := decodeContent(t, res)
		if mc.Result != "failed" || strings.Contains(mc.Message, "unconfirmed") {
			t.Errorf("result = %q %q, want failed without an unconfirmed suffix", mc.Result, mc.Message)
		}
		if len(*calls) != 1 {
			t.Errorf("calls = %d", len(*calls))
		}
		// the draft is spent.
		_, err = x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
		if messageOf(err) != "The draft was already decided." {
			t.Errorf("second decision err = %v, want DRAFT_ALREADY_DECIDED", err)
		}
	})

	t.Run("ephemeral flow creation fails after the claim", func(t *testing.T) {
		x := newDraftHarness(t)
		calls := x.expectCallCreate(nil)
		x.req.EXPECT().FlowV1FlowCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, stderrors.New("rejected action")).Times(1)
		d := x.newCallDraft(&CallDraft{Destinations: flowCallDraft("").Destinations, Actions: []fmaction.Action{{Type: fmaction.TypeHangup}}})
		x.setNow(d.TMCreate.Add(time.Second))

		res, err := x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if mc := decodeContent(t, res); mc.Result != "failed" {
			t.Errorf("result = %q, want failed", mc.Result)
		}
		if len(*calls) != 0 {
			t.Errorf("calls = %d, want none", len(*calls))
		}
		_, err = x.h.DraftDecision(context.Background(), x.c.ID, d.ID, message.DraftDecisionApprove)
		if messageOf(err) != "The draft was already decided." {
			t.Errorf("the draft must be spent, err = %v", err)
		}
	})
}

func Test_draftDecisionFindBlock(t *testing.T) {
	if draftDecisionFindBlock(nil) != nil {
		t.Error("no blocks must give nil")
	}
	if draftDecisionFindBlock([]CardBlock{{Type: message.BlockTypeCallDraft}}) != nil {
		t.Error("a call_draft without a call is not decidable")
	}
	if draftDecisionFindBlock([]CardBlock{{Type: "info"}, {Type: message.BlockTypeCallDraft, Call: &CallDraft{}}}) == nil {
		t.Error("a call_draft with a call must be found after other blocks")
	}
}
