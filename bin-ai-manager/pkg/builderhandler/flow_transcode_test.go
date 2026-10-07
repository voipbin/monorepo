package builderhandler

import (
	"reflect"
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
