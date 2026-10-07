package builderhandler

// flow_validate.go runs the structural checks of VOIP-1573 design doc §3.3
// over an already-transcoded []fmaction.Action. It never trusts anything
// the LLM said about itself (design doc §3.2 step 4's "clients lie" note
// extends to §3.3): every check re-derives its answer from the action list
// and the flow-manager metadata registries.

import (
	"bytes"
	"encoding/json"
	"reflect"

	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/actioncatalog"
	fmaction "monorepo/bin-flow-manager/models/action"
)

// ValidateDraft returns every warning §3.3 defines for actions, which must
// already be in the order AssembleFlowDraft produces (start first, open
// ends last).
func ValidateDraft(actions []fmaction.Action) []string {
	var warnings []string

	warnings = append(warnings, checkRequiredFields(actions)...)
	warnings = append(warnings, checkInvalidOptions(actions)...)
	warnings = append(warnings, checkOpenEnds(actions)...)
	warnings = append(warnings, checkUnreachable(actions)...)
	warnings = append(warnings, checkMediaMixed(actions)...)

	return warnings
}

// checkRequiredFields flags any action whose option is missing a field
// actioncatalog.RequiredFields marks Required (this is the catalog's own
// Required value, not a second builder-owned tag; see RequiredFields' doc
// comment). A ref:"resource" field already warns select_resource in
// resolveOption when required and absent; this check also covers
// non-ref required fields (e.g. talk's "text").
func checkRequiredFields(actions []fmaction.Action) []string {
	var warnings []string
	for _, a := range actions {
		for field := range actioncatalog.RequiredFields(a.Type) {
			v, ok := a.Option[field]
			if !ok || v == nil || v == "" {
				warnings = append(warnings, flowbuilder.WarningMissingRequired+": "+a.ID.String()+"."+field)
			}
		}
	}
	return warnings
}

// checkInvalidOptions strict-decodes each action's option map into its real
// OptionXxx struct (DisallowUnknownFields), catching a key the LLM invented
// that does not exist on the type, or a value of the wrong shape (e.g. a
// string where a number is expected). A disallowed/malformed option is
// still left in the draft (so the user can see and fix it in the editor);
// this only adds a warning.
func checkInvalidOptions(actions []fmaction.Action) []string {
	var warnings []string
	for _, a := range actions {
		optAny, ok := fmaction.OptionStructByType[a.Type]
		if !ok {
			continue
		}
		b, err := json.Marshal(a.Option)
		if err != nil {
			warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+a.ID.String())
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		target := reflect.New(reflect.TypeOf(optAny)).Interface()
		if err := dec.Decode(target); err != nil {
			warnings = append(warnings, flowbuilder.WarningInvalidOption+": "+a.ID.String())
		}
	}
	return warnings
}

// checkOpenEnds flags every FlowKindContinue action with no outgoing
// next_id that is not the last element of the array (design doc §3.3,
// round-7 fix: the last element is always allowed to be open; the flow
// simply finishes there).
func checkOpenEnds(actions []fmaction.Action) []string {
	var warnings []string
	for i, a := range actions {
		if i == len(actions)-1 {
			continue
		}
		if isOpenEnd(a) {
			warnings = append(warnings, flowbuilder.WarningOpenEnd+": "+a.ID.String())
		}
	}
	return warnings
}

// checkUnreachable flags every action no edge (next_id or a ref:"action"
// field) reaches from actions[0], using the same edge definition as the
// layout BFS (computeLayout), so "unreachable" and "placed in the
// rightmost layout column" always agree.
func checkUnreachable(actions []fmaction.Action) []string {
	if len(actions) == 0 {
		return nil
	}
	reached := make([]bool, len(actions))
	byID := make(map[string]int, len(actions))
	for i, a := range actions {
		byID[a.ID.String()] = i
	}
	reached[0] = true
	queue := []int{0}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		a := actions[cur]
		var targets []string
		if a.NextID != fmaction.IDEmpty {
			targets = append(targets, a.NextID.String())
		}
		for _, f := range fmaction.RefFieldsOf(a.Type) {
			if f.Kind == fmaction.RefKindAction {
				targets = append(targets, extractActionRefTargets(a.Option, f)...)
			}
		}
		for _, t := range targets {
			if j, ok := byID[t]; ok && !reached[j] {
				reached[j] = true
				queue = append(queue, j)
			}
		}
	}

	var warnings []string
	for i, a := range actions {
		if !reached[i] {
			warnings = append(warnings, flowbuilder.WarningUnreachable+": "+a.ID.String())
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
