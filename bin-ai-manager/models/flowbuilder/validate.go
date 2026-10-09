package flowbuilder

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"monorepo/bin-ai-manager/models/builder"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// invalid builds the only error type this package returns. The message names
// the violated rule but never echoes caller-supplied text.
func invalid(format string, args ...any) error {
	return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, ReasonInvalidArgument, fmt.Sprintf(format, args...))
}

// ValidateRequest checks the request against the shared conversation limits
// (builder.ValidateRequest, which counts message text only) and the
// draft-side limits of design doc 5. Failures are InvalidArgument.
func ValidateRequest(req *ChatRequest) error {
	if req == nil {
		return invalid("request is required")
	}
	if err := builder.ValidateRequest(&builder.ChatRequest{Messages: req.Messages}); err != nil {
		return err
	}

	if len(req.SupportedActionTypes) == 0 {
		return invalid("supported_action_types must not be empty")
	}
	if len(req.SupportedActionTypes) > MaxSupportedTypes {
		return invalid("supported_action_types exceeds the limit of %d", MaxSupportedTypes)
	}
	for i, t := range req.SupportedActionTypes {
		if t == "" || utf8.RuneCountInString(t) > MaxSupportedTypeRunes {
			return invalid("supported_action_types[%d] is empty or exceeds %d characters", i, MaxSupportedTypeRunes)
		}
	}

	d := req.CurrentDraft
	if d == nil {
		return nil
	}
	if len(d.Actions) > MaxFlowNodes {
		return invalid("current_draft.actions exceeds the limit of %d", MaxFlowNodes)
	}
	if len(d.Labels) > len(d.Actions) {
		return invalid("current_draft.labels has more entries than actions")
	}
	for _, l := range d.Labels {
		if utf8.RuneCountInString(l) > MaxLabelRunes {
			return invalid("current_draft.labels contains a label over %d characters", MaxLabelRunes)
		}
	}

	total := 0
	for i, a := range d.Actions {
		if opt, ok := a["option"]; ok && opt != nil {
			if depth(opt) > MaxOptionDepth {
				return invalid("current_draft.actions[%d].option is nested deeper than %d", i, MaxOptionDepth)
			}
			b, err := json.Marshal(opt)
			if err != nil {
				return invalid("current_draft.actions[%d].option could not be read", i)
			}
			if len(b) > MaxOptionBytes {
				return invalid("current_draft.actions[%d].option exceeds %d bytes", i, MaxOptionBytes)
			}
		}
		b, err := json.Marshal(a)
		if err != nil {
			return invalid("current_draft.actions[%d] could not be read", i)
		}
		total += len(b)
	}
	if total > MaxDraftBytes {
		return invalid("current_draft exceeds %d bytes", MaxDraftBytes)
	}
	return nil
}

// depth returns the nesting depth of a decoded JSON value (scalars are 0).
func depth(v any) int {
	switch x := v.(type) {
	case map[string]any:
		max := 0
		for _, c := range x {
			if d := depth(c); d > max {
				max = d
			}
		}
		return max + 1
	case []any:
		max := 0
		for _, c := range x {
			if d := depth(c); d > max {
				max = d
			}
		}
		return max + 1
	default:
		return 0
	}
}
