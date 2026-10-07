package builderhandler

import (
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

func TestAssembleFlowDraft_LinearHappyPath(t *testing.T) {
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

func TestAssembleFlowDraft_DuplicateLabelKeepsFirst(t *testing.T) {
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

func TestAssembleFlowDraft_UnsupportedTypeRewiresChain(t *testing.T) {
	// a -> (unsupported b) -> c. b must be dropped and a's next rewired to c.
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "a", Type: string(fmaction.TypeAnswer), Next: strp("b")},
		{Label: "b", Type: string(fmaction.TypeCall), Next: strp("c")}, // call: structurally excluded
		{Label: "c", Type: string(fmaction.TypeHangup)},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeAnswer, fmaction.TypeHangup))
	if len(draft.Actions) != 2 {
		t.Fatalf("len(Actions) = %d, want 2 (b dropped)", len(draft.Actions))
	}
	aID := draft.Actions[0]["id"].(string)
	cID := draft.Actions[1]["id"].(string)
	if draft.Actions[0]["next_id"] != cID {
		t.Errorf("a.next_id = %v, want it rewired to c (%s)", draft.Actions[0]["next_id"], cID)
	}
	_ = aID
	if !containsPrefix(warnings, flowbuilder.WarningUnsupportedAction) {
		t.Errorf("warnings = %v, want a %s warning", warnings, flowbuilder.WarningUnsupportedAction)
	}
}

func TestAssembleFlowDraft_AllUnsupportedIsEmptyDraft(t *testing.T) {
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

func TestAssembleFlowDraft_ResourceFieldAlwaysClearedEvenIfSupplied(t *testing.T) {
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

func TestAssembleFlowDraft_ActionRefResolvesLabelToUUID(t *testing.T) {
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

func TestAssembleFlowDraft_InvalidLabelRefClearsAndWarns(t *testing.T) {
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

func TestAssembleFlowDraft_JumpTypeIgnoresNext(t *testing.T) {
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

func TestAssembleFlowDraft_OpenEndsMoveToBackExceptStart(t *testing.T) {
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

func TestValidateDraft_MissingRequiredField(t *testing.T) {
	a := fmaction.Action{ID: fmaction.IDEmpty, Type: fmaction.TypeTalk, Option: map[string]any{}}
	warnings := ValidateDraft([]fmaction.Action{a})
	if !containsPrefix(warnings, flowbuilder.WarningMissingRequired) {
		t.Errorf("warnings = %v, want a %s warning (talk.text is required)", warnings, flowbuilder.WarningMissingRequired)
	}
}

func TestValidateDraft_OpenEndExceptLast(t *testing.T) {
	first := fmaction.Action{ID: mustID(t, "1"), Type: fmaction.TypeAnswer, NextID: fmaction.IDEmpty}
	last := fmaction.Action{ID: mustID(t, "2"), Type: fmaction.TypeHangup}
	warnings := ValidateDraft([]fmaction.Action{first, last})
	if !containsPrefix(warnings, flowbuilder.WarningOpenEnd) {
		t.Errorf("warnings = %v, want a %s warning for the non-last open end", warnings, flowbuilder.WarningOpenEnd)
	}
}

func TestValidateDraft_MediaMixed(t *testing.T) {
	talk := fmaction.Action{ID: mustID(t, "1"), Type: fmaction.TypeTalk, Option: map[string]any{"text": "hi"}}
	aitask := fmaction.Action{ID: mustID(t, "2"), Type: fmaction.TypeAITask, Option: map[string]any{}}
	warnings := ValidateDraft([]fmaction.Action{talk, aitask})
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
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("uuid.NewV4() error = %v", err)
	}
	return id
}
