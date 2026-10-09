package builderhandler

// flow_validate.go runs the structural checks of VOIP-1573 design doc §3.3
// over an already-transcoded []fmaction.Action. It never trusts anything
// the LLM said about itself (design doc §3.2 step 4's "clients lie" note
// extends to §3.3): every check re-derives its answer from the action list
// and the flow-manager metadata registries.

import (
	"sort"
	"strconv"

	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/actioncatalog"
	fmaction "monorepo/bin-flow-manager/models/action"
)

// ValidateDraft returns every warning design doc 3.3 defines for actions,
// which must already be in the order AssembleFlowDraft produces (start
// first, open ends last). labels maps action id to label and is used only to
// make warning details readable; a missing entry falls back to the id.
// invalid_option is not reported here: it is produced while transcoding,
// where the offending key is dropped.
func ValidateDraft(actions []fmaction.Action, labels map[string]string) []string {
	var warnings []string

	warnings = append(warnings, checkEmptyActionRefs(actions, labels)...)
	warnings = append(warnings, checkRequiredFields(actions, labels)...)
	warnings = append(warnings, checkAddressFields(actions, labels)...)
	warnings = append(warnings, checkOpenEnds(actions, labels)...)
	warnings = append(warnings, checkUnreachable(actions, labels)...)
	warnings = append(warnings, checkMediaMixed(actions)...)

	return warnings
}

func labelOf(a fmaction.Action, labels map[string]string) string {
	if l, ok := labels[a.ID.String()]; ok && l != "" {
		return l
	}
	return a.ID.String()
}

// checkEmptyActionRefs flags every ref:"action" field that has no target
// (design doc 3.3 item 1). A scalar field warns when empty. A map field
// (branch.target_ids) warns only when the map is empty AND the node has no
// non-empty scalar action ref (branch.default_target_id covers the miss).
func checkEmptyActionRefs(actions []fmaction.Action, labels map[string]string) []string {
	var warnings []string
	for _, a := range actions {
		var scalarSet bool
		var scalarFields, mapFields []fmaction.RefField
		for _, f := range fmaction.RefFieldsOf(a.Type) {
			if f.Kind != fmaction.RefKindAction {
				continue
			}
			if f.IsMap {
				mapFields = append(mapFields, f)
			} else {
				scalarFields = append(scalarFields, f)
			}
		}
		for _, f := range scalarFields {
			if len(extractActionRefTargets(a.Option, f)) > 0 {
				scalarSet = true
				continue
			}
			warnings = append(warnings, flowbuilder.WarningEmptyActionRef+": "+labelOf(a, labels)+"."+f.JSONName)
		}
		for _, f := range mapFields {
			if len(extractActionRefTargets(a.Option, f)) > 0 || scalarSet {
				continue
			}
			warnings = append(warnings, flowbuilder.WarningEmptyActionRef+": "+labelOf(a, labels)+"."+f.JSONName)
		}
	}
	return warnings
}

// checkRequiredFields flags any action whose option is missing a field
// actioncatalog.RequiredFields marks Required, except ref-tagged fields:
// those are reported as select_resource (resource) or empty_action_ref
// (action), never twice. The Required value is the catalog's own; the
// builder has no second "required" source.
func checkRequiredFields(actions []fmaction.Action, labels map[string]string) []string {
	var warnings []string
	for _, a := range actions {
		refNames := map[string]bool{}
		for _, f := range fmaction.RefFieldsOf(a.Type) {
			refNames[f.JSONName] = true
		}
		fields := make([]string, 0)
		for field := range actioncatalog.RequiredFields(a.Type) {
			if !refNames[field] {
				fields = append(fields, field)
			}
		}
		sort.Strings(fields)
		for _, field := range fields {
			v, ok := a.Option[field]
			if !ok || v == nil || v == "" {
				warnings = append(warnings, flowbuilder.WarningMissingRequired+": "+labelOf(a, labels)+"."+field)
			}
		}
	}
	return warnings
}

// checkOpenEnds flags every FlowKindContinue action with no outgoing
// next_id that is not the last element of the array (design doc 3.2 step 6;
// the last element is a normal finish).
func checkOpenEnds(actions []fmaction.Action, labels map[string]string) []string {
	var warnings []string
	for i, a := range actions {
		if i == len(actions)-1 {
			continue
		}
		if isOpenEnd(a) {
			warnings = append(warnings, flowbuilder.WarningOpenEnd+": "+labelOf(a, labels))
		}
	}
	return warnings
}

// checkUnreachable flags every action the executor cannot reach from
// actions[0], using the edges from successors: next_id,
// ref:"action" targets and the array fall-through of a non-last open end.
func checkUnreachable(actions []fmaction.Action, labels map[string]string) []string {
	if len(actions) == 0 {
		return nil
	}
	adj := successors(actions)
	reached := make([]bool, len(actions))
	reached[0] = true
	queue := []int{0}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, j := range adj[cur] {
			if !reached[j] {
				reached[j] = true
				queue = append(queue, j)
			}
		}
	}

	var warnings []string
	for i, a := range actions {
		if !reached[i] {
			warnings = append(warnings, flowbuilder.WarningUnreachable+": "+labelOf(a, labels))
		}
	}
	return warnings
}

// checkMediaMixed warns once if the draft contains an action that only
// applies to a real-time (call) activeflow and an action that only applies
// to a non-real-time (message/email/chat) activeflow, per
// fmaction.MapRequiredMediasByType -- such a draft can only ever half-run,
// since a single activeflow has one media type for its whole lifetime
// (pkg/activeflowhandler/execute.go's verifyActionType skips the other
// half silently at runtime; the builder surfaces that as a warning instead
// of letting it be a silent runtime surprise).
func checkMediaMixed(actions []fmaction.Action) []string {
	hasRTCOnly := false
	hasNonRTCOnly := false
	for _, a := range actions {
		medias, ok := fmaction.MapRequiredMediasByType[a.Type]
		if !ok || len(medias) != 1 {
			continue
		}
		switch medias[0] {
		case fmaction.MediaTypeRealTimeCommunication:
			hasRTCOnly = true
		case fmaction.MediaTypeNonRealTimeCommunication:
			hasNonRTCOnly = true
		}
	}
	if hasRTCOnly && hasNonRTCOnly {
		return []string{flowbuilder.WarningMediaMixed}
	}
	return nil
}

// checkAddressFields reports the address fields (ref:"address") the user still
// has to fill: a required one that is absent or empty is missing_required, and
// an address whose target is empty (cleared because it named a platform
// resource such as an agent, or never given) is select_resource. Reporting it
// here, not only when the target is cleared, keeps the hint on every later turn.
func checkAddressFields(actions []fmaction.Action, labels map[string]string) []string {
	var warnings []string
	for _, a := range actions {
		required := actioncatalog.RequiredFields(a.Type)
		for _, f := range fmaction.RefFieldsOf(a.Type) {
			if f.Kind != fmaction.RefKindAddress {
				continue
			}
			where := labelOf(a, labels) + "." + f.JSONName
			raw := a.Option[f.JSONName]

			if !f.IsList {
				obj, ok := raw.(map[string]any)
				if !ok || len(obj) == 0 {
					if required[f.JSONName] {
						warnings = append(warnings, flowbuilder.WarningMissingRequired+": "+where)
					}
					continue
				}
				if t, _ := obj["target"].(string); t == "" {
					warnings = append(warnings, flowbuilder.WarningSelectResource+": "+where)
				}
				continue
			}

			list, _ := raw.([]any)
			if len(list) == 0 {
				if required[f.JSONName] {
					warnings = append(warnings, flowbuilder.WarningMissingRequired+": "+where)
				}
				continue
			}
			for i, item := range list {
				obj, _ := item.(map[string]any)
				if t, _ := obj["target"].(string); t == "" {
					warnings = append(warnings, flowbuilder.WarningSelectResource+": "+where+"["+strconv.Itoa(i)+"]")
				}
			}
		}
	}
	return warnings
}
