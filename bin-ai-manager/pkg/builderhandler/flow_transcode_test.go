package builderhandler

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/flowbuilder"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

func strp(s string) *string { return &s }

func allAllowed(types ...fmaction.Type) map[fmaction.Type]bool {
	m := make(map[fmaction.Type]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

func Test_AssembleFlowDraft_LinearHappyPath(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "greet", Type: string(fmaction.TypeAnswer), Next: strp("talk1")},
		{Label: "talk1", Type: string(fmaction.TypeTalk), Option: map[string]any{"text": "hello"}, Next: strp("end")},
		{Label: "end", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer, fmaction.TypeTalk, fmaction.TypeHangup))
	if draft == nil {
		t.Fatalf("draft is nil, warnings=%v", warnings)
	}
	if len(draft.Actions) != 3 {
		t.Fatalf("len(Actions) = %d, want 3", len(draft.Actions))
	}
	if draft.Actions[0]["type"] != string(fmaction.TypeAnswer) {
		t.Errorf("Actions[0].type = %v, want answer (start must stay first)", draft.Actions[0]["type"])
	}
	for _, w := range warnings {
		t.Errorf("unexpected warning: %s", w)
	}
}

func Test_AssembleFlowDraft_DuplicateLabelKeepsFirst(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeAnswer)},
		{Label: "a", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer, fmaction.TypeHangup))
	if len(draft.Actions) != 1 {
		t.Fatalf("len(Actions) = %d, want 1", len(draft.Actions))
	}
	if draft.Actions[0]["type"] != string(fmaction.TypeAnswer) {
		t.Errorf("kept type = %v, want the first occurrence (answer)", draft.Actions[0]["type"])
	}
	if !containsPrefix(warnings, flowbuilder.WarningDuplicateLabel) {
		t.Errorf("warnings = %v, want a %s warning", warnings, flowbuilder.WarningDuplicateLabel)
	}
}

func Test_AssembleFlowDraft_UnsupportedTypeClearsReferenceWithoutStitching(t *testing.T) {
	// a -> (unsupported b) -> c. b is dropped and a.next is cleared (no
	// stitching to c). a is then an open end that is not last, so the
	// executor falls through to c: c is reachable, and the user is told
	// about the open end instead.
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeAnswer), Next: strp("b")},
		{Label: "b", Type: string(fmaction.TypeCall), Next: strp("c")},
		{Label: "c", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer, fmaction.TypeHangup))
	if len(draft.Actions) != 2 {
		t.Fatalf("len(Actions) = %d, want 2 (b dropped)", len(draft.Actions))
	}
	if nid, _ := draft.Actions[0]["next_id"].(string); nid != "" && nid != fmaction.IDEmpty.String() {
		t.Errorf("a.next_id = %v, want cleared", nid)
	}
	for _, want := range []string{flowbuilder.WarningUnsupportedAction, flowbuilder.WarningOpenEnd} {
		if !containsPrefix(warnings, want) {
			t.Errorf("warnings = %v, want a %s warning", warnings, want)
		}
	}
	if containsPrefix(warnings, flowbuilder.WarningInvalidLabelRef) {
		t.Errorf("warnings = %v, a reference to a removed node must not be reported as invalid_label_ref", warnings)
	}
}

func Test_AssembleFlowDraft_RemovedStartUsesItsNext(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "s", Type: string(fmaction.TypeCall), Next: strp("t")},
		{Label: "x", Type: string(fmaction.TypeHangup)},
		{Label: "t", Type: string(fmaction.TypeAnswer), Next: strp("x")},
	}}
	draft, _ := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer, fmaction.TypeHangup))
	if draft == nil || draft.Actions[0]["type"] != string(fmaction.TypeAnswer) {
		t.Fatalf("draft = %+v, want answer (t) as the new start", draft)
	}
}

func Test_AssembleFlowDraft_RemovedStartWithoutUsableNextIsEmpty(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "s", Type: string(fmaction.TypeCall)},
		{Label: "x", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeHangup))
	if draft != nil || !containsExact(warnings, flowbuilder.WarningEmptyDraft) {
		t.Errorf("draft = %+v warnings = %v, want empty draft", draft, warnings)
	}
}

func Test_AssembleFlowDraft_UnknownOptionKeyDroppedAndReported(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "t", Type: string(fmaction.TypeTalk), Option: map[string]any{"text": "hi", "bogus": 1}},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeTalk))
	opt := draft.Actions[0]["option"].(map[string]any)
	if _, ok := opt["bogus"]; ok {
		t.Errorf("option = %v, want bogus dropped", opt)
	}
	if opt["text"] != "hi" {
		t.Errorf("option = %v, want text kept", opt)
	}
	if !containsExact(warnings, flowbuilder.WarningInvalidOption+": t.bogus") {
		t.Errorf("warnings = %v, want invalid_option: t.bogus", warnings)
	}
}

func Test_AssembleFlowDraft_BranchWithoutDefaultReportsEmptyActionRef(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "b", Type: string(fmaction.TypeBranch), Option: map[string]any{"variable": "v"}},
	}}
	_, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeBranch))
	if !containsPrefix(warnings, flowbuilder.WarningEmptyActionRef) {
		t.Errorf("warnings = %v, want empty_action_ref", warnings)
	}
}

func Test_AssembleFlowDraft_AllUnsupportedIsEmptyDraft(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeCall)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer))
	if draft != nil {
		t.Fatalf("draft = %+v, want nil (empty draft)", draft)
	}
	if !containsExact(warnings, flowbuilder.WarningEmptyDraft) {
		t.Errorf("warnings = %v, want %s", warnings, flowbuilder.WarningEmptyDraft)
	}
}

func Test_AssembleFlowDraft_ResourceFieldAlwaysClearedEvenIfSupplied(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "q", Type: string(fmaction.TypeQueueJoin), Option: map[string]any{
			"queue_id": "11111111-1111-1111-1111-111111111111", // the LLM must never supply a real resource id
		}},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeQueueJoin))
	opt, _ := draft.Actions[0]["option"].(map[string]any)
	if v, ok := opt["queue_id"]; ok && v != "" && v != nil {
		t.Errorf("option.queue_id = %v, want cleared", v)
	}
	if !containsPrefix(warnings, flowbuilder.WarningSelectResource) {
		t.Errorf("warnings = %v, want a %s warning (queue_id is Required)", warnings, flowbuilder.WarningSelectResource)
	}
}

func Test_AssembleFlowDraft_ActionRefResolvesLabelToUUID(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "g", Type: string(fmaction.TypeGoto), Option: map[string]any{
			"target_id":  "g", // self-loop by label
			"loop_count": float64(3),
		}},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeGoto))
	opt := draft.Actions[0]["option"].(map[string]any)
	gID := draft.Actions[0]["id"].(string)
	if opt["target_id"] != gID {
		t.Errorf("target_id = %v, want resolved to self id %s", opt["target_id"], gID)
	}
	for _, w := range warnings {
		t.Errorf("unexpected warning: %s", w)
	}
}

func Test_AssembleFlowDraft_InvalidLabelRefClearsAndWarns(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "g", Type: string(fmaction.TypeGoto), Option: map[string]any{
			"target_id": "does_not_exist",
		}},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeGoto))
	opt, _ := draft.Actions[0]["option"].(map[string]any) // option omitempty: absent when goto's target_id was its only (now-cleared) field
	if v, ok := opt["target_id"]; ok && v != "" {
		t.Errorf("target_id = %v, want cleared", v)
	}
	if !containsPrefix(warnings, flowbuilder.WarningInvalidLabelRef) {
		t.Errorf("warnings = %v, want a %s warning", warnings, flowbuilder.WarningInvalidLabelRef)
	}
}

func Test_AssembleFlowDraft_JumpTypeIgnoresNext(t *testing.T) {
	// branch is FlowKindJump: `next` must be ignored (and warned if set),
	// not transcoded to next_id (round-2/3 review findings).
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "b", Type: string(fmaction.TypeBranch), Next: strp("ignored"), Option: map[string]any{
			"variable": "${voipbin.call.digits}",
			"target_ids": map[string]any{
				"1": "ignored",
			},
		}},
		{Label: "ignored", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeBranch, fmaction.TypeHangup))
	bID := draft.Actions[0]["id"].(string)
	if draft.Actions[0]["next_id"] == bID {
		t.Fatalf("sanity: next_id should not equal self id")
	}
	if nid, _ := draft.Actions[0]["next_id"].(string); nid != "" && nid != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("branch next_id = %v, want cleared (jump type ignores next)", nid)
	}
	if !containsPrefix(warnings, flowbuilder.WarningNextIgnored) {
		t.Errorf("warnings = %v, want a %s warning", warnings, flowbuilder.WarningNextIgnored)
	}
	// but the map ref:"action" field must still resolve to the real id.
	opt := draft.Actions[0]["option"].(map[string]any)
	targetIDs := opt["target_ids"].(map[string]any)
	if targetIDs["1"] != draft.Actions[1]["id"] {
		t.Errorf("target_ids[1] = %v, want resolved to ignored's id %v", targetIDs["1"], draft.Actions[1]["id"])
	}
}

func Test_AssembleFlowDraft_OpenEndsMoveToBackExceptStart(t *testing.T) {
	// start has no next (open end) but must stay first; a true mid-chain
	// open end must move to the back.
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "start", Type: string(fmaction.TypeAnswer)},                                   // open end, but must stay actions[0]
		{Label: "mid", Type: string(fmaction.TypeTalk), Option: map[string]any{"text": "hi"}}, // open end
		{Label: "last", Type: string(fmaction.TypeHangup)},
	}}
	draft, _ := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer, fmaction.TypeTalk, fmaction.TypeHangup))
	if draft.Actions[0]["type"] != string(fmaction.TypeAnswer) {
		t.Fatalf("Actions[0] = %v, want start to stay first", draft.Actions[0]["type"])
	}
	if draft.Actions[len(draft.Actions)-1]["type"] != string(fmaction.TypeTalk) {
		t.Errorf("last element = %v, want the open-end 'mid' pushed to the back", draft.Actions[len(draft.Actions)-1]["type"])
	}
}

func Test_ValidateDraft_MissingRequiredField(t *testing.T) {
	a := fmaction.Action{ID: fmaction.IDEmpty, Type: fmaction.TypeTalk, Option: map[string]any{}}
	warnings := ValidateDraft([]fmaction.Action{a}, nil)
	if !containsPrefix(warnings, flowbuilder.WarningMissingRequired) {
		t.Errorf("warnings = %v, want a %s warning (talk.text is required)", warnings, flowbuilder.WarningMissingRequired)
	}
}

func Test_ValidateDraft_OpenEndExceptLast(t *testing.T) {
	first := fmaction.Action{ID: mustID(t, "1"), Type: fmaction.TypeAnswer, NextID: fmaction.IDEmpty}
	last := fmaction.Action{ID: mustID(t, "2"), Type: fmaction.TypeHangup}
	warnings := ValidateDraft([]fmaction.Action{first, last}, nil)
	if !containsPrefix(warnings, flowbuilder.WarningOpenEnd) {
		t.Errorf("warnings = %v, want a %s warning for the non-last open end", warnings, flowbuilder.WarningOpenEnd)
	}
}

func Test_ValidateDraft_MediaMixed(t *testing.T) {
	talk := fmaction.Action{ID: mustID(t, "1"), Type: fmaction.TypeTalk, Option: map[string]any{"text": "hi"}}
	aitask := fmaction.Action{ID: mustID(t, "2"), Type: fmaction.TypeAITask, Option: map[string]any{}}
	warnings := ValidateDraft([]fmaction.Action{talk, aitask}, nil)
	if !containsExact(warnings, flowbuilder.WarningMediaMixed) {
		t.Errorf("warnings = %v, want %s (talk is RTC-only, ai_task is non-RTC-only)", warnings, flowbuilder.WarningMediaMixed)
	}
}

func containsPrefix(warnings []string, prefix string) bool {
	for _, w := range warnings {
		if len(w) >= len(prefix) && w[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func containsExact(warnings []string, want string) bool {
	for _, w := range warnings {
		if w == want {
			return true
		}
	}
	return false
}

func mustID(t *testing.T, seed string) uuid.UUID {
	t.Helper()
	ids := map[string]string{
		"1": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071",
		"2": "841c5fa2-f0c2-11ee-834f-53b2b00ec88d",
	}
	id := uuid.FromStringOrNil(ids[seed])
	if id == uuid.Nil {
		t.Fatalf("no fixed uuid for seed %q", seed)
	}
	return id
}

func Test_ValidateDraft_unreachable(t *testing.T) {
	idA := mustID(t, "1")
	idB := mustID(t, "2")
	idC := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	tests := []struct {
		name      string
		actions   []fmaction.Action
		wantUnrch []string
	}{
		{
			name: "a node reached by next_id is reachable",
			actions: []fmaction.Action{
				{ID: idA, Type: fmaction.TypeAnswer, NextID: idB},
				{ID: idB, Type: fmaction.TypeHangup},
			},
		},
		{
			name: "array fall-through of a non-last open end reaches the next element",
			actions: []fmaction.Action{
				{ID: idA, Type: fmaction.TypeAnswer},
				{ID: idB, Type: fmaction.TypeHangup},
			},
		},
		{
			name: "a terminate node never falls through",
			actions: []fmaction.Action{
				{ID: idA, Type: fmaction.TypeStop},
				{ID: idB, Type: fmaction.TypeHangup},
			},
			wantUnrch: []string{flowbuilder.WarningUnreachable + ": b"},
		},
		{
			name: "a ref:action target is reachable and a node nothing points at is not",
			actions: []fmaction.Action{
				{ID: idA, Type: fmaction.TypeGoto, Option: map[string]any{"target_id": idB.String()}},
				{ID: idB, Type: fmaction.TypeStop},
				{ID: idC, Type: fmaction.TypeStop},
			},
			wantUnrch: []string{flowbuilder.WarningUnreachable + ": c"},
		},
	}
	labels := map[string]string{idA.String(): "a", idB.String(): "b", idC.String(): "c"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, w := range ValidateDraft(tt.actions, labels) {
				if strings.HasPrefix(w, flowbuilder.WarningUnreachable) {
					got = append(got, w)
				}
			}
			if !reflect.DeepEqual(got, tt.wantUnrch) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantUnrch, got)
			}
		})
	}
}

func Test_computeLayout(t *testing.T) {
	idA := mustID(t, "1")
	idB := mustID(t, "2")
	idC := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	// a (next b) -> b (branch: 1 -> c, default -> a) ; c unreachable? no: c is a branch target.
	actions := []fmaction.Action{
		{ID: idA, Type: fmaction.TypeAnswer, NextID: idB},
		{ID: idB, Type: fmaction.TypeBranch, Option: map[string]any{
			"target_ids":        map[string]any{"1": idC.String()},
			"default_target_id": idA.String(),
		}},
		{ID: idC, Type: fmaction.TypeStop},
		{ID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"), Type: fmaction.TypeStop},
	}
	got := computeLayout(actions)
	want := []flowbuilder.Position{
		{X: 0, Y: 100},  // start
		{X: 0, Y: 600},  // depth 1
		{X: 0, Y: 1100}, // depth 2 (branch target)
		{X: 0, Y: 1600}, // unreachable: one row below the deepest
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", want, got)
	}

	if got := computeLayout(nil); got != nil {
		t.Errorf("Wrong match. expect: nil for no actions, got: %v", got)
	}
}

// encoding/json matches keys case-insensitively and flow-manager decodes an
// option with it, so a case variant of a resource or reference key must not
// get past the clearing and resolution (review round 2, H1).
func Test_AssembleFlowDraft_caseVariantKeysCannotCarryReferences(t *testing.T) {
	const foreign = "11111111-2222-4333-8444-555555555555"
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "q", Type: string(fmaction.TypeQueueJoin), Option: map[string]any{"Queue_ID": foreign, "QUEUE_ID": foreign}},
		{Label: "b", Type: string(fmaction.TypeBranch), Option: map[string]any{
			"Variable":          "v",
			"Target_IDs":        map[string]any{"1": foreign},
			"DEFAULT_TARGET_ID": foreign,
		}},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeQueueJoin, fmaction.TypeBranch))

	raw, _ := json.Marshal(draft)
	if strings.Contains(string(raw), foreign) {
		t.Errorf("Wrong match. a foreign id reached the draft: %s", raw)
	}
	for _, key := range []string{"q.Queue_ID", "q.QUEUE_ID", "b.Variable", "b.Target_IDs", "b.DEFAULT_TARGET_ID"} {
		if !containsExact(warnings, flowbuilder.WarningInvalidOption+": "+key) {
			t.Errorf("Wrong match. expect invalid_option for %s in %v", key, warnings)
		}
	}
	if !containsPrefix(warnings, flowbuilder.WarningSelectResource) {
		t.Errorf("Wrong match. expect select_resource in %v", warnings)
	}
}

func Test_ReconstructGraph_caseVariantKeysAreNotShownToTheModel(t *testing.T) {
	draft := &flowbuilder.Draft{Actions: []map[string]any{
		{"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071", "type": "queue_join", "option": map[string]any{"Queue_ID": "11111111-2222-4333-8444-555555555555"}},
	}}
	g := ReconstructGraph(draft)
	raw, _ := json.Marshal(g)
	if strings.Contains(string(raw), "11111111") {
		t.Errorf("Wrong match. a resource id reached the model graph: %s", raw)
	}
}

func Test_FlowParse_labelWithControlCharactersIsSkipped(t *testing.T) {
	got, err := FlowParse(`{"message":"hi","draft":{"nodes":[{"label":"a\u0000b","type":"stop"},{"label":"ok","type":"stop"}]}}`)
	if err != nil || got.Graph == nil || len(got.Graph.Nodes) != 1 || got.Graph.Nodes[0].Label != "ok" {
		t.Errorf("Wrong match. expect: only ok, got: %+v %v", got, err)
	}
}

// Review round 3 (F): a nested address carries a resource id in target when
// its type is a platform resource. Only a value the user owns may stay.
func Test_AssembleFlowDraft_addressTargetOfAResourceTypeIsCleared(t *testing.T) {
	const victim = "11111111-2222-4333-8444-555555555555"
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "c", Type: string(fmaction.TypeConnect), Option: map[string]any{
			"source": map[string]any{"type": "tel", "target": "+15551230000"},
			"destinations": []any{
				map[string]any{"type": "agent", "target": victim},
				map[string]any{"type": "conference", "target": victim},
				map[string]any{"type": "tel", "target": "+15551230001"},
				map[string]any{"type": "Agent", "Target": victim},
				map[string]any{"type": "a_type_added_later", "target": victim},
			},
		}},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeConnect))

	raw, _ := json.Marshal(draft)
	if strings.Contains(string(raw), victim) {
		t.Errorf("Wrong match. a foreign id reached the draft: %s", raw)
	}
	for _, keep := range []string{"+15551230000", "+15551230001"} {
		if !strings.Contains(string(raw), keep) {
			t.Errorf("Wrong match. the user's own endpoint %s was lost: %s", keep, raw)
		}
	}
	if !containsPrefix(warnings, flowbuilder.WarningSelectResource+": c.destinations[0]") {
		t.Errorf("Wrong match. expect select_resource for destinations[0] in %v", warnings)
	}
}

func Test_AssembleFlowDraft_uuidShapedMapKeyIsDropped(t *testing.T) {
	const victim = "11111111-2222-4333-8444-555555555555"
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "b", Type: string(fmaction.TypeBranch), Option: map[string]any{
			"variable":   "v",
			"target_ids": map[string]any{victim: "e", "1": "e"},
		}},
		{Label: "e", Type: string(fmaction.TypeHangup)},
	}}
	draft, _ := AssembleFlowDraft(graph, allAllowed(fmaction.TypeBranch, fmaction.TypeHangup))
	raw, _ := json.Marshal(draft)
	if strings.Contains(string(raw), victim) {
		t.Errorf("Wrong match. a uuid-shaped key reached the draft: %s", raw)
	}
	if !strings.Contains(string(raw), `"1"`) {
		t.Errorf("Wrong match. the digit key was lost: %s", raw)
	}
}

// A value of the wrong shape is dropped key by key and reported (this is the
// behaviour dropInvalidOptionKeys exists for; review round 3 found no test).
func Test_AssembleFlowDraft_wrongShapedOptionValueIsDroppedAndReported(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "w", Type: string(fmaction.TypeSleep), Option: map[string]any{"duration": "abc"}},
		{Label: "e", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeSleep, fmaction.TypeHangup))
	raw, _ := json.Marshal(draft)
	if strings.Contains(string(raw), "abc") {
		t.Errorf("Wrong match. the wrong-shaped value stayed: %s", raw)
	}
	if !containsExact(warnings, flowbuilder.WarningInvalidOption+": w.duration") {
		t.Errorf("Wrong match. expect invalid_option for w.duration in %v", warnings)
	}
}

// The draft the server returns must pass the validation of the next request.
func Test_AssembleFlowDraft_optionStaysWithinTheRequestLimit(t *testing.T) {
	targets := map[string]any{}
	for i := 0; i < 400; i++ {
		targets[strconv.Itoa(i)] = "b" // 400 * ~8B = fits 4096B as labels; 400 * ~45B as ids does not
	}
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeBranch), Option: map[string]any{"variable": "v", "target_ids": targets}},
		{Label: "b", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeBranch, fmaction.TypeHangup))

	next := &flowbuilder.ChatRequest{
		Messages:             []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: "go on"}},
		SupportedActionTypes: []string{string(fmaction.TypeBranch)},
		CurrentDraft:         draft,
	}
	if err := flowbuilder.ValidateRequest(next); err != nil {
		t.Errorf("Wrong match. the draft does not pass the next request's validation: %v", err)
	}
	if !containsPrefix(warnings, flowbuilder.WarningInvalidOption) {
		t.Errorf("Wrong match. expect invalid_option for the cut entries, got %v", warnings)
	}
}

func Test_AssembleFlowDraft_warningsAreCapped(t *testing.T) {
	var nodes []flowbuilder.SymbolicNode
	for i := 0; i < 60; i++ {
		opt := map[string]any{}
		for k := 0; k < 100; k++ {
			opt["bad"+strconv.Itoa(k)] = 1
		}
		nodes = append(nodes, flowbuilder.SymbolicNode{Label: "n" + strconv.Itoa(i) + "x", Type: string(fmaction.TypeAnswer), Option: opt})
	}
	_, warnings := AssembleFlowDraft(flowbuilder.SymbolicGraph{Nodes: nodes}, allAllowed(fmaction.TypeAnswer))
	if len(warnings) > maxDraftWarnings {
		t.Errorf("Wrong match. expect at most %d warnings, got %d", maxDraftWarnings, len(warnings))
	}
}

// The model must not see ids of types it cannot use (unknown, internal,
// excluded): their option is not sent.
func Test_ReconstructGraph_optionOfAnUnusableTypeIsNotShownToTheModel(t *testing.T) {
	const victim = "11111111-2222-4333-8444-555555555555"
	draft := &flowbuilder.Draft{Actions: []map[string]any{
		{"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071", "type": "bogus", "option": map[string]any{"queue_id": victim}},
		{"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4072", "type": "call", "option": map[string]any{"actions": []any{map[string]any{"id": victim}}}},
		{"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4073", "type": "connect", "option": map[string]any{
			"destinations": []any{map[string]any{"type": "agent", "target": victim}},
		}},
	}}
	raw, _ := json.Marshal(ReconstructGraph(draft))
	if strings.Contains(string(raw), victim) {
		t.Errorf("Wrong match. an id reached the model graph: %s", raw)
	}
}

func Test_ReconstructGraph_unknownScalarActionRefIsNotShownToTheModel(t *testing.T) {
	const stray = "11111111-2222-4333-8444-555555555555"
	draft := &flowbuilder.Draft{Actions: []map[string]any{
		{"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071", "type": "branch", "option": map[string]any{"variable": "v", "default_target_id": stray}},
	}}
	raw, _ := json.Marshal(ReconstructGraph(draft))
	if strings.Contains(string(raw), stray) {
		t.Errorf("Wrong match. an id that is not in the draft reached the model graph: %s", raw)
	}
}

func Test_AssembleFlowDraft_longOptionKeyIsShortenedInTheWarning(t *testing.T) {
	long := strings.Repeat("k", 4000)
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeAnswer), Option: map[string]any{long: 1}},
	}}
	_, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer))
	for _, w := range warnings {
		if len(w) > 200 {
			t.Errorf("Wrong match. expect a short warning, got %d bytes", len(w))
		}
	}
}

// Review round 5 (J): the catalog used to say the address fields are always
// null, and a missing address raised no warning at all.
func Test_FlowCatalog_addressFieldsAreNotTaughtAsAlwaysNull(t *testing.T) {
	cat := FlowCatalog([]fmaction.Type{fmaction.TypeConnect})
	if !strings.Contains(cat, "address fields (source, destinations)") {
		t.Errorf("Wrong match. expect a separate address-fields line, got:\n%s", cat)
	}
	for _, line := range strings.Split(cat, "\n") {
		if strings.Contains(line, "resource fields") && strings.Contains(line, "destinations") {
			t.Errorf("Wrong match. destinations must not be listed as an always-null resource field: %s", line)
		}
	}
}

func Test_AssembleFlowDraft_missingOrClearedAddressIsAlwaysReported(t *testing.T) {
	const victim = "11111111-2222-4333-8444-555555555555"
	tests := []struct {
		name   string
		option map[string]any
		want   string
		absent []string
	}{
		{"no destinations", map[string]any{}, flowbuilder.WarningMissingRequired + ": c.destinations", nil},
		{"null destinations", map[string]any{"destinations": nil}, flowbuilder.WarningMissingRequired + ": c.destinations", nil},
		{"empty list", map[string]any{"destinations": []any{}}, flowbuilder.WarningMissingRequired + ": c.destinations", nil},
		{"an address without a target", map[string]any{"destinations": []any{map[string]any{"type": "agent"}}}, flowbuilder.WarningSelectResource + ": c.destinations[0]", nil},
		{"an agent target is cleared", map[string]any{"destinations": []any{map[string]any{"type": "agent", "target": victim}}}, flowbuilder.WarningSelectResource + ": c.destinations[0]", []string{victim}},
		{"a phone number with no type is cleared and reported", map[string]any{"destinations": []any{map[string]any{"target": "+15551230000"}}}, flowbuilder.WarningSelectResource + ": c.destinations[0]", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{{Label: "c", Type: string(fmaction.TypeConnect), Option: tt.option}}}
			draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeConnect))
			if !containsExact(warnings, tt.want) {
				t.Errorf("Wrong match. expect %q in %v", tt.want, warnings)
			}
			raw, _ := json.Marshal(draft)
			for _, a := range tt.absent {
				if strings.Contains(string(raw), a) {
					t.Errorf("Wrong match. %s stayed in the draft: %s", a, raw)
				}
			}
		})
	}
}

func Test_AssembleFlowDraft_addressTypeIsAcceptedInAnyCase(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{{Label: "c", Type: string(fmaction.TypeConnect), Option: map[string]any{
		"destinations": []any{map[string]any{"type": " TEL ", "target": "+15551230000"}},
	}}}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeConnect))
	raw, _ := json.Marshal(draft)
	if !strings.Contains(string(raw), `"type":"tel"`) || !strings.Contains(string(raw), "+15551230000") {
		t.Errorf("Wrong match. expect the number kept with type tel, got %s", raw)
	}
	if containsPrefix(warnings, flowbuilder.WarningSelectResource) {
		t.Errorf("Wrong match. unexpected select_resource in %v", warnings)
	}
}

// The hint must not vanish on the second turn: the draft is rebuilt from the
// client's current_draft, where the cleared target is simply absent.
func Test_ValidateDraft_addressHintSurvivesTheNextTurn(t *testing.T) {
	id := mustID(t, "1")
	actions := []fmaction.Action{{ID: id, Type: fmaction.TypeConnect, Option: map[string]any{
		"destinations": []any{map[string]any{"type": "agent"}},
	}}}
	got := ValidateDraft(actions, map[string]string{id.String(): "c"})
	if !containsExact(got, flowbuilder.WarningSelectResource+": c.destinations[0]") {
		t.Errorf("Wrong match. expect select_resource in %v", got)
	}
}

func Test_AssembleFlowDraft_nextPointingAtItselfIsDropped(t *testing.T) {
	b := "b"
	a := "a"
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeAnswer), Next: &b},
		{Label: "b", Type: string(fmaction.TypeAnswer), Next: &b},
	}}
	draft, _ := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer))

	byLabel := map[string]map[string]any{}
	for _, act := range draft.Actions {
		byLabel[draft.Labels[act["id"].(string)]] = act
	}
	if v, ok := byLabel["b"]["next_id"]; ok && v != "" && v != fmaction.IDEmpty.String() {
		t.Errorf("Wrong match. expect no next_id for the self reference, got %v", v)
	}
	if byLabel["a"]["next_id"] == nil {
		t.Errorf("Wrong match. a keeps its next step to b")
	}

	// A loop through two nodes is kept: the executor bounds it (design OQ9).
	graph = flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeAnswer), Next: &b},
		{Label: "b", Type: string(fmaction.TypeAnswer), Next: &a},
	}}
	draft, _ = AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer))
	for _, act := range draft.Actions {
		if act["next_id"] == nil {
			t.Errorf("Wrong match. a two-node loop keeps both next steps, got %v", draft.Actions)
		}
	}
}
