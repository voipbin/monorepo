package builderhandler

// flow_transcode.go converts the LLM's label-addressed SymbolicGraph into a
// confirmed Draft (VOIP-1573 design doc §3.2). Every step here is a pure,
// deterministic function: the LLM never sees or produces a UUID, a position,
// or a decision about which warning fires. This file has no dependency on
// the LLM client or any RPC; it is unit-tested directly.

import (
	"monorepo/bin-ai-manager/models/flowbuilder"
	fmaction "monorepo/bin-flow-manager/models/action"

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
	// allowedTypes. References to a removed node are cleared, not stitched
	// to its successor (see filterUnsupportedTypes).
	nodes, removed, w, startOK := filterUnsupportedTypes(nodes, allowedTypes)
	warnings = append(warnings, w...)

	if len(nodes) == 0 || !startOK {
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

		opt, w := resolveOption(n, labelToID, removed)
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
			if !removed[*n.Next] {
				warnings = append(warnings, flowbuilder.WarningInvalidLabelRef+": "+n.Label)
			}
		}

		actions = append(actions, fmaction.Action{
			ID:     id,
			NextID: nextID,
			Type:   fmaction.Type(n.Type),
			Option: opt,
		})
	}

	// Step 6: keep the start node (first surviving node, step 2 may have
	// changed what that is) at index 0 and push every "open end"
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

	warnings = append(warnings, ValidateDraft(actions, draft.Labels)...)

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
// allowedTypes (design doc 3.2 step 2). References that pointed at a removed
// node are NOT stitched to the removed node's successor: they are cleared by
// the later steps (resolveOption / next handling) and surface as open_end or
// empty_action_ref, because LLM output outside the catalog is rare and
// stitching is deferred until measured.
//
// The one exception is the start node. If the first node was removed, the
// node its next pointed at becomes the new start (moved to index 0). If that
// cannot be determined (no next, or next also removed), startOK is false and
// the caller returns an empty draft: a graph without a start is meaningless.
//
// removed is the set of labels dropped here, so later steps can clear
// references to them silently instead of reporting invalid_label_ref.
func filterUnsupportedTypes(nodes []flowbuilder.SymbolicNode, allowedTypes map[fmaction.Type]bool) (out []flowbuilder.SymbolicNode, removed map[string]bool, warnings []string, startOK bool) {
	removed = make(map[string]bool)
	out = make([]flowbuilder.SymbolicNode, 0, len(nodes))
	for _, n := range nodes {
		if !allowedTypes[fmaction.Type(n.Type)] {
			removed[n.Label] = true
			warnings = append(warnings, flowbuilder.WarningUnsupportedAction+": "+n.Label)
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return out, removed, warnings, false
	}
	if len(nodes) > 0 && removed[nodes[0].Label] {
		if nodes[0].Next == nil {
			return out, removed, warnings, false
		}
		idx := -1
		for i, n := range out {
			if n.Label == *nodes[0].Next {
				idx = i
				break
			}
		}
		if idx < 0 {
			return out, removed, warnings, false
		}
		start := out[idx]
		rest := make([]flowbuilder.SymbolicNode, 0, len(out))
		rest = append(rest, start)
		for i, n := range out {
			if i != idx {
				rest = append(rest, n)
			}
		}
		out = rest
	}
	return out, removed, warnings, true
}
