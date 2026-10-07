package builderhandler

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/flowbuilder"
)

// FlowParsed is the validated result of one Flow Builder model turn.
type FlowParsed struct {
	Message     string
	Graph       *flowbuilder.SymbolicGraph
	Assumptions []string
	Warnings    []string
}

// FlowParse extracts and validates the model's JSON answer. It follows the
// same tolerance as Parse (code fence or prose around the object, fields
// decoded independently), but the draft is a label-addressed node list.
// A draft that is not an object, has no usable node, or has more than
// flowbuilder.MaxFlowNodes nodes is discarded with draft_discarded; the
// message is still returned. Nothing here carries model output into an error.
func FlowParse(raw string) (*FlowParsed, error) {
	fields, ok := firstUsableObject(raw)
	if !ok {
		return nil, ErrInvalidResponse
	}

	out := &FlowParsed{}
	_ = json.Unmarshal(fields["message"], &out.Message)

	draftRaw, hasDraft := fields["draft"]
	if !hasDraft || isJSONNull(draftRaw) {
		return out, nil
	}

	graph, warnings := decodeFlowDraft(draftRaw)
	out.Warnings = append(out.Warnings, warnings...)
	if graph == nil {
		out.Warnings = append(out.Warnings, WarnDraftDiscarded)
		return out, nil
	}
	out.Graph = graph

	out.Assumptions = []string{}
	if ar, ok := fields["assumptions"]; ok {
		var a []string
		if err := json.Unmarshal(ar, &a); err == nil && a != nil {
			out.Assumptions = a
		}
	}
	return out, nil
}

// decodeFlowDraft returns nil when the draft must be discarded.
func decodeFlowDraft(raw json.RawMessage) (*flowbuilder.SymbolicGraph, []string) {
	var f map[string]json.RawMessage
	if err := json.Unmarshal(raw, &f); err != nil || f == nil {
		return nil, nil
	}
	var rawNodes []json.RawMessage
	if err := json.Unmarshal(f["nodes"], &rawNodes); err != nil {
		return nil, nil
	}
	if len(rawNodes) > flowbuilder.MaxFlowNodes {
		return nil, nil
	}

	var warnings []string
	nodes := make([]flowbuilder.SymbolicNode, 0, len(rawNodes))
	for _, rn := range rawNodes {
		var nf map[string]json.RawMessage
		if err := json.Unmarshal(rn, &nf); err != nil || nf == nil {
			continue
		}
		var n flowbuilder.SymbolicNode
		_ = json.Unmarshal(nf["label"], &n.Label)
		_ = json.Unmarshal(nf["type"], &n.Type)
		n.Label = strings.TrimSpace(n.Label)
		if n.Label == "" || n.Type == "" || utf8.RuneCountInString(n.Label) > flowbuilder.MaxLabelRunes || strings.IndexFunc(n.Label, unicode.IsControl) >= 0 {
			continue // a node without a usable address cannot be referenced or placed
		}

		if or, ok := nf["option"]; ok && !isJSONNull(or) {
			var opt map[string]any
			if err := json.Unmarshal(or, &opt); err == nil && len(or) <= flowbuilder.MaxOptionBytes {
				n.Option = opt
			} else {
				warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+n.Label)
			}
		}
		if nr, ok := nf["next"]; ok && !isJSONNull(nr) {
			var next string
			if err := json.Unmarshal(nr, &next); err == nil && next != "" {
				n.Next = &next
			}
		}
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		return nil, warnings
	}
	return &flowbuilder.SymbolicGraph{Nodes: nodes}, warnings
}
