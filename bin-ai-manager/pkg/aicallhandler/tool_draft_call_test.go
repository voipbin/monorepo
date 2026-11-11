package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
)

const (
	testCallFlowID = "d7000000-0000-11f0-0000-0000000000f1"
)

func draftCallTool(args string) *message.ToolCall {
	return &message.ToolCall{
		ID:   "d8000000-0000-11f0-0000-000000000003",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameDraftCall,
			Arguments: args,
		},
	}
}

func draftCallAIcall() *aicall.AIcall {
	return &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d7000000-0000-11f0-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("d7000000-0000-11f0-0000-000000000002"),
		},
	}
}

func newDraftCallHandler(t *testing.T) (*aicallHandler, *requesthandler.MockRequestHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	req := requesthandler.NewMockRequestHandler(mc)
	return &aicallHandler{reqHandler: req}, req
}

func testCallFlow(c *aicall.AIcall, tmUpdate *time.Time, persist bool, actions []fmaction.Action) *fmflow.Flow {
	return &fmflow.Flow{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(testCallFlowID), CustomerID: c.CustomerID},
		Name:     "Reminder flow",
		Persist:  persist,
		Actions:  actions,
		TMUpdate: tmUpdate,
	}
}

func Test_toolHandleDraftCall_flowDraft(t *testing.T) {
	c := draftCallAIcall()
	h, req := newDraftCallHandler(t)

	tm := time.Date(2026, 11, 11, 10, 0, 0, 123456789, time.UTC)
	flow := testCallFlow(c, &tm, true, []fmaction.Action{
		{Type: fmaction.TypeTalk},
		{Type: fmaction.TypeConnect, Option: map[string]any{"destinations": []any{map[string]any{"type": "tel", "target": "+15550100"}}}},
		{Type: fmaction.TypeMessageSend},
	})
	// the draft reads the flow once; CallV1CallsCreate has no expectation, so a call fails the test.
	req.EXPECT().FlowV1FlowGet(gomock.Any(), uuid.FromStringOrNil(testCallFlowID)).Return(flow, nil).Times(1)

	args := jsonString(t, map[string]any{
		"flow_id":      testCallFlowID,
		"destinations": []map[string]any{{"type": "tel", "target": "+15550101", "target_name": "John"}},
		"source":       map[string]any{"type": "tel", "target": "+15550199"},
		"anonymous":    "no",
		"variables":    map[string]any{"campaign": 123, "ok": true},
		"title":        "  Remind John  ",
	})
	res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(args))
	if res.Result != "success" || res.Message != draftCallDisplayedMessage {
		t.Fatalf("result = %q/%q, want success/%q", res.Result, res.Message, draftCallDisplayedMessage)
	}
	if len(res.Blocks) != 1 || res.Blocks[0].Type != message.BlockTypeCallDraft || res.Blocks[0].Title != "Remind John" {
		t.Fatalf("blocks = %+v", res.Blocks)
	}
	d := res.Blocks[0].Call
	if d == nil || d.FlowID != testCallFlowID || len(d.Actions) != 0 {
		t.Fatalf("call = %+v", d)
	}
	if len(d.Destinations) != 1 || d.Destinations[0].Target != "+15550101" || d.Destinations[0].TargetName != "John" {
		t.Errorf("destinations = %+v, want the whole address kept", d.Destinations)
	}
	if d.Source == nil || d.Source.Target != "+15550199" || d.Anonymous != "no" {
		t.Errorf("source/anonymous = %+v/%q", d.Source, d.Anonymous)
	}
	if d.Variables["campaign"] != "123" || d.Variables["ok"] != "true" {
		t.Errorf("variables = %v, want strings", d.Variables)
	}
	if d.FlowName != "Reminder flow" || d.FlowPersistent == nil || !*d.FlowPersistent {
		t.Errorf("flow labels = %q/%v", d.FlowName, d.FlowPersistent)
	}
	if got := strings.Join(d.FlowActionTypes, ","); got != "talk,connect,message_send" {
		t.Errorf("flow_action_types = %s", got)
	}
	if len(d.FlowDestinations) != 1 || d.FlowDestinations[0].Target != "+15550100" {
		t.Errorf("flow_destinations = %+v, want the connect destination", d.FlowDestinations)
	}
	if d.FlowTMUpdate != "2026-11-11T10:00:00.123456789Z" {
		t.Errorf("flow_tm_update = %q", d.FlowTMUpdate)
	}
	if strings.Contains(res.Message, "+1555") {
		t.Errorf("the result message must not carry the draft body")
	}
}

func Test_toolHandleDraftCall_temporaryFlowHasNoTMUpdate(t *testing.T) {
	c := draftCallAIcall()
	h, req := newDraftCallHandler(t)
	req.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(testCallFlow(c, nil, false, nil), nil)

	res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(jsonString(t, map[string]any{
		"flow_id": testCallFlowID, "destinations": []map[string]any{{"type": "tel", "target": "+15550101"}},
	})))
	if res.Result != "success" {
		t.Fatalf("result = %q %q", res.Result, res.Message)
	}
	d := res.Blocks[0].Call
	if d.FlowTMUpdate != "" || d.FlowPersistent == nil || *d.FlowPersistent {
		t.Errorf("tm_update/persistent = %q/%v, want empty/false", d.FlowTMUpdate, d.FlowPersistent)
	}
}

func Test_toolHandleDraftCall_actionsDraft(t *testing.T) {
	c := draftCallAIcall()
	h, _ := newDraftCallHandler(t) // no request is expected for an actions draft.

	args := `{"actions":[{"type":"talk","option":{"text":"Hi","n":9007199254740993,"f":1.0}},{"type":"hangup"}],` +
		`"destinations":[{"type":"sip","target":"sip:a@example.com"},{"type":"extension","target":"1001"}]}`
	res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(args))
	if res.Result != "success" {
		t.Fatalf("result = %q %q", res.Result, res.Message)
	}
	d := res.Blocks[0].Call
	if d.FlowID != "" || len(d.Actions) != 2 || len(d.Destinations) != 2 {
		t.Fatalf("call = %+v", d)
	}
	if res.Blocks[0].Title != draftCallTitleDefault {
		t.Errorf("title = %q", res.Blocks[0].Title)
	}

	// the numbers survive draft arguments -> stored row -> approval decoding, digit for digit.
	stored, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var back messageContent
	dec := json.NewDecoder(strings.NewReader(string(stored)))
	dec.UseNumber()
	if err := dec.Decode(&back); err != nil {
		t.Fatal(err)
	}
	opt := back.Blocks[0].Call.Actions[0].Option
	if fmt.Sprint(opt["n"]) != "9007199254740993" || fmt.Sprint(opt["f"]) != "1.0" {
		t.Errorf("option numbers = %v / %v, want 9007199254740993 / 1.0", opt["n"], opt["f"])
	}
	rebuilt, err := callDraftCreateArguments(back.Blocks[0].Call)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rebuilt), `"n":9007199254740993`) || !strings.Contains(string(rebuilt), `"f":1.0`) {
		t.Errorf("rebuilt create_call arguments lost the digits: %s", rebuilt)
	}
}

func Test_toolHandleDraftCall_refusals(t *testing.T) {
	c := draftCallAIcall()
	dest := `"destinations":[{"type":"tel","target":"+15550101"}]`
	tests := []struct {
		name string
		args string
		want string
	}{
		{"not json", `{`, ""},
		{"neither flow nor actions", `{` + dest + `}`, "either flow_id or actions is required"},
		{"both flow and actions", `{"flow_id":"` + testCallFlowID + `","actions":[{"type":"hangup"}],` + dest + `}`, "not both"},
		{"empty actions array", `{"actions":[],` + dest + `}`, "either flow_id or actions is required"},
		{"bad flow id", `{"flow_id":"nope",` + dest + `}`, ""},
		{"no destinations", `{"actions":[{"type":"hangup"}]}`, "at least one destination is required"},
		{"empty destination target", `{"actions":[{"type":"hangup"}],"destinations":[{"type":"tel","target":""}]}`, "target must not be empty"},
		{"unknown action type", `{"actions":[{"type":"launch_missiles"}],` + dest + `}`, "unknown type"},
		{"reserved variable key", `{"actions":[{"type":"hangup"}],` + dest + `,"variables":{"voipbin.x":"1"}}`, "must not start with"},
		{"reserved variable key upper case", `{"actions":[{"type":"hangup"}],` + dest + `,"variables":{"Voipbin.x":"1"}}`, "must not start with"},
		{"variable key with a space", `{"actions":[{"type":"hangup"}],` + dest + `,"variables":{" voipbin.x":"1"}}`, "start or end with a space"},
		{"empty variable key", `{"actions":[{"type":"hangup"}],` + dest + `,"variables":{"":"1"}}`, "must not be empty"},
		{"variable value too large", `{"actions":[{"type":"hangup"}],` + dest + `,"variables":{"a":"` + strings.Repeat("x", 32*1024+1) + `"}}`, "above 32768 bytes"},
		{"too large to store", `{"actions":[{"type":"talk","option":{"text":"` + strings.Repeat("y", draftStoredMaxBytes) + `"}}],` + dest + `}`, "too large to store"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newDraftCallHandler(t) // strict mock: any request fails the test.
			res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(tt.args))
			if res.Result != "failed" {
				t.Fatalf("result = %q, want failed", res.Result)
			}
			if len(res.Blocks) != 0 {
				t.Errorf("a refused draft must carry no card, blocks = %+v", res.Blocks)
			}
			if tt.want != "" && !strings.Contains(res.Message, tt.want) {
				t.Errorf("message = %q, want it to contain %q", res.Message, tt.want)
			}
		})
	}
}

func Test_toolHandleDraftCall_variablesLimits(t *testing.T) {
	c := draftCallAIcall()
	dest := `"destinations":[{"type":"tel","target":"+15550101"}]`

	many := map[string]string{}
	for i := 0; i < draftCallVariablesMaxKeys+1; i++ {
		many[fmt.Sprintf("k%d", i)] = "v"
	}
	h, _ := newDraftCallHandler(t)
	res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(`{"actions":[{"type":"hangup"}],`+dest+`,"variables":`+jsonString(t, many)+`}`))
	if res.Result != "failed" || !strings.Contains(res.Message, "keys") {
		t.Errorf("101 keys: %q %q, want a keys refusal", res.Result, res.Message)
	}

	delete(many, "k100")
	res = h.toolHandleDraftCall(context.Background(), c, draftCallTool(`{"actions":[{"type":"hangup"}],`+dest+`,"variables":`+jsonString(t, many)+`}`))
	if res.Result != "success" {
		t.Errorf("100 keys: %q %q, want success", res.Result, res.Message)
	}

	// total above 64 KiB with every value under 32 KiB.
	big := map[string]string{"a": strings.Repeat("x", 30*1024), "b": strings.Repeat("x", 30*1024), "c": strings.Repeat("x", 30*1024)}
	if err := validateCallDraftVariables(big); err == nil || !strings.Contains(err.Error(), "in total") {
		t.Errorf("total over the limit: err = %v", err)
	}
}

func Test_toolHandleDraftCall_flowRefusals(t *testing.T) {
	c := draftCallAIcall()
	args := jsonString(t, map[string]any{"flow_id": testCallFlowID, "destinations": []map[string]any{{"type": "tel", "target": "+15550101"}}})

	t.Run("flow read error", func(t *testing.T) {
		h, req := newDraftCallHandler(t)
		req.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("boom"))
		res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(args))
		if res.Result != "failed" || res.Message != errCouldNotResolveFlow.Error() {
			t.Errorf("result = %q %q, want the masked flow error", res.Result, res.Message)
		}
	})
	t.Run("flow of another customer", func(t *testing.T) {
		h, req := newDraftCallHandler(t)
		other := testCallFlow(c, nil, true, nil)
		other.CustomerID = uuid.FromStringOrNil("d7000000-0000-11f0-0000-0000000000aa")
		req.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(other, nil)
		res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(args))
		if res.Result != "failed" || res.Message != errCouldNotResolveFlow.Error() {
			t.Errorf("result = %q %q, want the same masked flow error", res.Result, res.Message)
		}
	})
}

func Test_toolHandleDraftCall_titleCut(t *testing.T) {
	c := draftCallAIcall()
	h, _ := newDraftCallHandler(t)
	res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(`{"actions":[{"type":"hangup"}],"destinations":[{"type":"tel","target":"+15550101"}],"title":"`+strings.Repeat("t", 300)+`"}`))
	if res.Result != "success" || len([]rune(res.Blocks[0].Title)) > cardTitleMaxLen {
		t.Errorf("title length = %d, want at most %d", len([]rune(res.Blocks[0].Title)), cardTitleMaxLen)
	}
}

func Test_callDraftFlowSummary(t *testing.T) {
	types, dests := callDraftFlowSummary([]fmaction.Action{
		{Type: fmaction.TypeConnect, Option: map[string]any{"destinations": []any{map[string]any{"type": "tel", "target": "+1"}}}},
		{Type: fmaction.TypeCall, Option: map[string]any{"destinations": "not a list"}}, // unreadable option: type only
		{Type: fmaction.TypeEmailSend, Option: map[string]any{"destinations": []any{map[string]any{"type": "email", "target": "a@b.c"}}}},
		{Type: fmaction.TypeCall, Option: map[string]any{"destinations": []any{map[string]any{"type": "tel", "target": "+2"}}}},
	})
	if got := strings.Join(types, ","); got != "connect,call,email_send,call" {
		t.Errorf("types = %s", got)
	}
	if len(dests) != 2 || dests[0].Target != "+1" || dests[1].Target != "+2" {
		t.Errorf("destinations = %+v, want only the connect and call destinations that could be read", dests)
	}
}

func Test_callDraftFlowBindingValid_noFlow(t *testing.T) {
	h, _ := newDraftCallHandler(t)
	ok, err := h.callDraftFlowBindingValid(context.Background(), draftCallAIcall(), &CallDraft{})
	if err != nil || !ok {
		t.Errorf("an actions draft has nothing to bind: %v %v", ok, err)
	}
}

// A flow read that answers (nil, nil) must be refused, not dereferenced.
func Test_toolHandleDraftCall_flowNilResult(t *testing.T) {
	c := draftCallAIcall()
	h, req := newDraftCallHandler(t)
	req.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(nil, nil)
	res := h.toolHandleDraftCall(context.Background(), c, draftCallTool(jsonString(t, map[string]any{
		"flow_id": testCallFlowID, "destinations": []map[string]any{{"type": "tel", "target": "+15550101"}},
	})))
	if res.Result != "failed" || res.Message != errCouldNotResolveFlow.Error() {
		t.Errorf("result = %q %q, want the masked flow error", res.Result, res.Message)
	}
}

func Test_callDraftFlowBindingValid_nilResult(t *testing.T) {
	h, req := newDraftCallHandler(t)
	req.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(nil, nil)
	ok, err := h.callDraftFlowBindingValid(context.Background(), draftCallAIcall(), &CallDraft{FlowID: testCallFlowID})
	if err != nil || ok {
		t.Errorf("ok/err = %v/%v, want false/nil for a nil flow", ok, err)
	}
}
