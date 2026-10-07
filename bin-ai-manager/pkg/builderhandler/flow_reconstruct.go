package builderhandler

// flow_reconstruct.go rebuilds the label-addressed graph from the client's
// current_draft each turn (design doc 3.2 step 1, section 5). The server
// stores nothing and trusts nothing: ids, labels and references are
// re-derived and anything inconsistent is dropped, so the model never sees a
// UUID and a forged draft cannot smuggle a reference.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/flowbuilder"

	fmaction "monorepo/bin-flow-manager/models/action"
)

var autoLabelPattern = regexp.MustCompile(`^n[0-9]+$`)

// ReconstructGraph returns the symbolic graph for draft, or nil when there is
// no draft or nothing in it is usable.
func ReconstructGraph(draft *flowbuilder.Draft) *flowbuilder.SymbolicGraph {
	if draft == nil || len(draft.Actions) == 0 {
		return nil
	}

	// Decode actions, dropping undecodable ones and duplicate or empty ids.
	actions := make([]fmaction.Action, 0, len(draft.Actions))
	seen := map[string]bool{}
	for _, m := range draft.Actions {
		b, err := json.Marshal(m)
		if err != nil {
			continue
		}
		var a fmaction.Action
		if err := json.Unmarshal(b, &a); err != nil {
			continue
		}
		id := a.ID.String()
		if a.ID == fmaction.IDEmpty || a.Type == "" || seen[id] {
			continue
		}
		seen[id] = true
		actions = append(actions, a)
	}
	if len(actions) == 0 {
		return nil
	}

	// Labels: keep a client label only if it is sane and unique. Auto-style
	// labels (n1, n2, ...) are reserved for server-assigned names, so a
	// client label that looks like one is re-assigned.
	idToLabel := make(map[string]string, len(actions))
	used := map[string]bool{}
	for _, a := range actions {
		l := draft.Labels[a.ID.String()]
		if l == "" || utf8.RuneCountInString(l) > flowbuilder.MaxLabelRunes || autoLabelPattern.MatchString(l) || used[l] {
			continue
		}
		used[l] = true
		idToLabel[a.ID.String()] = l
	}
	next := 1
	for _, a := range actions {
		if _, ok := idToLabel[a.ID.String()]; ok {
			continue
		}
		for {
			l := fmt.Sprintf("n%d", next)
			next++
			if !used[l] {
				used[l] = true
				idToLabel[a.ID.String()] = l
				break
			}
		}
	}

	nodes := make([]flowbuilder.SymbolicNode, 0, len(actions))
	for _, a := range actions {
		n := flowbuilder.SymbolicNode{Label: idToLabel[a.ID.String()], Type: string(a.Type)}

		// Only exact option keys: a case variant such as "Queue_ID" would
		// slip past the resource hiding below (see exactOptionKeys).
		opt, _ := exactOptionKeys(a.Type, a.Option)
		if !fmaction.IsBuilderExposable(a.Type) {
			// The model cannot use this type (unknown, internal or excluded
			// like call), so its option is not sent to the provider either:
			// it could carry ids the model must never see.
			opt = map[string]any{}
		}
		for _, f := range fmaction.RefFieldsOf(a.Type) {
			switch f.Kind {
			case fmaction.RefKindResource:
				delete(opt, f.JSONName) // resources are never shown to the model
			case fmaction.RefKindAction:
				opt = labelizeActionRef(opt, f, idToLabel)
			case fmaction.RefKindAddress:
				opt, _ = sanitizeAddressField(opt, f, "", false)
			}
		}
		if len(opt) > 0 {
			n.Option = opt
		}

		meta, ok := fmaction.MetaByType[a.Type]
		if (!ok || meta.Flow == fmaction.FlowKindContinue) && a.NextID != fmaction.IDEmpty {
			if l, ok := idToLabel[a.NextID.String()]; ok {
				n.Next = &l
			}
		}
		nodes = append(nodes, n)
	}
	return &flowbuilder.SymbolicGraph{Nodes: nodes}
}

// labelizeActionRef rewrites a ref:"action" field from UUID strings to labels.
// A value that is not the id of an action in this draft is dropped.
func labelizeActionRef(opt map[string]any, f fmaction.RefField, idToLabel map[string]string) map[string]any {
	if f.IsMap {
		raw, ok := opt[f.JSONName].(map[string]any)
		if !ok {
			delete(opt, f.JSONName)
			return opt
		}
		out := make(map[string]any, len(raw))
		for k, v := range raw {
			if _, err := uuid.FromString(k); err == nil {
				continue // a key is a value to match, never an id
			}
			if s, ok := v.(string); ok {
				if l, ok := idToLabel[s]; ok {
					out[k] = l
				}
			}
		}
		opt[f.JSONName] = out
		return opt
	}
	s, ok := opt[f.JSONName].(string)
	if !ok {
		delete(opt, f.JSONName)
		return opt
	}
	if l, ok := idToLabel[s]; ok {
		opt[f.JSONName] = l
	} else {
		delete(opt, f.JSONName)
	}
	return opt
}
