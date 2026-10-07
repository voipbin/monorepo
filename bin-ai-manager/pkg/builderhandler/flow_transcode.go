package builderhandler

// flow_transcode.go converts the LLM's label-addressed SymbolicGraph into a
// confirmed Draft (VOIP-1573 design doc §3.2). Every step here is a pure,
// deterministic function: the LLM never sees or produces a UUID, a position,
// or a decision about which warning fires. This file has no dependency on
// the LLM client or any RPC; it is unit-tested directly.

import (
	fmaction "monorepo/bin-flow-manager/models/action"
	"monorepo/bin-ai-manager/models/flowbuilder"

	"github.com/gofrs/uuid"
)

// AssembleFlowDraft runs design doc §3.2 steps 1-9 and returns the confirmed
// draft plus every warning collected along the way. allowedTypes is the
// already-computed intersection of builder-exposable types and the
// client's supported_action_types (design doc §2.5/§2.6); it is NOT
// recomputed here.
//
// AssembleFlowDraft never returns an error: a malformed or partially invalid
// graph degrades to warnings and, in the worst case, an empty draft (step 8)
// rather than failing the whole turn, matching the Assistant Builder's
// "an almost-right draft beats no draft" philosophy (design doc §3.2 step 4).
func AssembleFlowDraft(graph flowbuilder.SymbolicGraph, allowedTypes map[fmaction.Type]bool) (*flowbuilder.Draft, []string) {
	var warnings []string

	// Step 1: label normalization. Keep the first node per label; drop
	// later duplicates.
	nodes, w := dedupeLabels(graph.Nodes)
	warnings = append(warnings, w...)

	// Step 2: type filter. Remove nodes whose type is not in
	// allowedTypes, rewiring any reference to the removed node onto its
	// own `next` (bypass chain, cycle-safe).
	nodes, w = filterUnsupportedTypes(nodes, allowedTypes)
	warnings = append(warnings, w...)

	if len(nodes) == 0 {
		return nil, append(warnings, flowbuilder.WarningEmptyDraft)
	}

	// Step 3: assign a new UUID per surviving label.
	labelToID := make(map[string]uuid.UUID, len(nodes))
	for _, n := range nodes {
		labelToID[n.Label] = mustNewV4()
	}

	// Steps 4-5: resolve ref:"resource" (cleared first) and ref:"action"
	// fields, and next -> next_id, per node.
	actions := make([]fmaction.Action, 0, len(nodes))
	idToLabel := make(map[string]string, len(nodes))
	for _, n := range nodes {
		id := labelToID[n.Label]
		idToLabel[id.String()] = n.Label

		opt, w := resolveOption(n, labelToID)
		warnings = append(warnings, w...)

		var nextID uuid.UUID
		flow, hasMeta := fmaction.MetaByType[fmaction.Type(n.Type)]
		if hasMeta && flow.Flow != fmaction.FlowKindContinue {
			// jump/terminate types never consult `next`.
			if n.Next != nil {
				warnings = append(warnings, flowbuilder.WarningNextIgnored+": "+n.Label)
			}
			nextID = fmaction.IDEmpty
		} else if n.Next == nil {
			nextID = fmaction.IDEmpty
		} else if target, ok := labelToID[*n.Next]; ok {
			nextID = target
		} else {
			nextID = fmaction.IDEmpty
			warnings = append(warnings, flowbuilder.WarningInvalidLabelRef+": "+n.Label)
		}

		actions = append(actions, fmaction.Action{
			ID:     id,
			NextID: nextID,
			Type:   fmaction.Type(n.Type),
			Option: opt,
		})
	}

	// Step 6: order by BFS from the start node (first surviving node,
	// §3.2 step 2 may have changed what that is), push every "open end"
	// (a FlowKindContinue node with no outgoing next_id) to the back so
	// array-adjacency fall-through is a deliberate, late choice rather
	// than an accident of input order. Every continuing node's next_id
	// that does have a target is left untouched; only a genuinely open
	// end (next_id == IDEmpty and FlowKind == continue) relies on
	// adjacency, and §3.3 flags every such node that is not last.
	actions = reorderStartFirstOpenEndsLast(actions)

	// Step 7: layout.
	positions := computeLayout(actions)

	// Step 9: assemble the wire draft. (Step 8, the empty-draft case, is
	// handled above and after filterUnsupportedTypes/dedupe can make the
	// graph empty; a start-not-resolvable case cannot occur here because
	// filterUnsupportedTypes always leaves a valid first element when
	// len(nodes) > 0 -- see its own doc comment.)
	draft := &flowbuilder.Draft{
		Actions:   make([]map[string]any, 0, len(actions)),
		Positions: make(map[string]flowbuilder.Position, len(actions)),
		Labels:    make(map[string]string, len(actions)),
	}
	for i, a := range actions {
		m, _ := actionToMap(a)
		draft.Actions = append(draft.Actions, m)
		draft.Positions[a.ID.String()] = positions[i]
		draft.Labels[a.ID.String()] = idToLabel[a.ID.String()]
	}

	return draft, warnings
}

func mustNewV4() uuid.UUID {
	id, err := uuid.NewV4()
	if err != nil {
		// uuid.NewV4 only fails if the system entropy source is broken;
		// there is no sane fallback, and every caller already treats a
		// panic path as unreachable in practice (same assumption the
		// rest of the monorepo makes about gofrs/uuid).
		panic(err)
	}
	return id
}

// dedupeLabels keeps the first node per label and drops later duplicates
// (design doc §3.2 step 1), warning once per dropped duplicate.
func dedupeLabels(nodes []flowbuilder.SymbolicNode) ([]flowbuilder.SymbolicNode, []string) {
	seen := make(map[string]bool, len(nodes))
	out := make([]flowbuilder.SymbolicNode, 0, len(nodes))
	var warnings []string
	for _, n := range nodes {
		if seen[n.Label] {
			warnings = append(warnings, flowbuilder.WarningDuplicateLabel+": "+n.Label)
			continue
		}
		seen[n.Label] = true
		out = append(out, n)
	}
	return out, warnings
}

// filterUnsupportedTypes removes every node whose type is not in
// allowedTypes, rewiring next/ref:"action" references that pointed at a
// removed node onto the removed node's own `next`, following the chain with
// a visited set so a cycle of removed nodes degrades to "no target" instead
// of looping. If the removed node's own chain also runs out before hitting
// a surviving node, the reference is cleared.
//
// If a removed node happened to be the symbolic "start" (nodes[0] before
// filtering), the new start is whichever surviving node its own `next`
// chain resolves to; if that chain never reaches a surviving node, the
// result is the same as removing everything (step 8's empty-draft case).
func filterUnsupportedTypes(nodes []flowbuilder.SymbolicNode, allowedTypes map[fmaction.Type]bool) ([]flowbuilder.SymbolicNode, []string) {
	byLabel := make(map[string]flowbuilder.SymbolicNode, len(nodes))
	for _, n := range nodes {
		byLabel[n.Label] = n
	}

	survives := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if allowedTypes[fmaction.Type(n.Type)] {
			survives[n.Label] = true
		}
	}

	// resolve(label) walks the removed-node chain to the first surviving
	// label (or "" if none), guarding against cycles.
	resolve := func(label *string) *string {
		if label == nil {
			return nil
		}
		visited := make(map[string]bool)
		cur := *label
		for {
			if survives[cur] {
				return &cur
			}
			if visited[cur] {
				return nil // cycle of removed nodes
			}
			visited[cur] = true
			n, ok := byLabel[cur]
			if !ok || n.Next == nil {
				return nil
			}
			cur = *n.Next
		}
	}

	var warnings []string
	out := make([]flowbuilder.SymbolicNode, 0, len(nodes))
	for _, n := range nodes {
		if !survives[n.Label] {
			warnings = append(warnings, flowbuilder.WarningUnsupportedAction+": "+n.Label)
			continue
		}
		n.Next = resolve(n.Next)
		out = append(out, n)
	}
	return out, warnings
}
