package ai

import (
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/tool"
)

// ValidateToolNames returns an error if any name in toolNames is not
// permitted for the given (already-resolved) Type. See
// docs/plans/2026-07-14-insight-ai-tool-name-validation-design.md for the
// full rationale (VOIP-1257).
//
// Rules:
//   - Type = TypeInsight: every name must be a member of
//     tool.AllInsightToolNames. tool.ToolNameAll is rejected (it currently
//     means "all Normal tools", which has no sensible meaning for an
//     Insight AI).
//   - Type = TypeNormal (or resolved default): every name must be a member
//     of tool.AllToolNames, or be tool.ToolNameAll.
//   - Any name that is neither a known AllToolNames member, a known
//     AllInsightToolNames member, nor ToolNameAll is rejected regardless of
//     Type (closes a pre-existing "unknown tool name silently accepted"
//     gap).
//   - nil/empty toolNames is always valid for either Type.
//
// The whole toolNames slice is checked (not short-circuited on the first
// valid or invalid element), so a mixed list such as
// ["all", "get_contact_interactions"] for an Insight AI is rejected because
// "all" alone is invalid for Insight, and ["all", "bogus_tool"] for a
// Normal AI is rejected because "bogus_tool" is invalid even though "all"
// is valid.
func ValidateToolNames(t Type, toolNames []tool.ToolName) error {
	if len(toolNames) == 0 {
		return nil
	}

	allowedNormal := make(map[tool.ToolName]bool, len(tool.AllToolNames))
	for _, n := range tool.AllToolNames {
		allowedNormal[n] = true
	}
	allowedInsight := make(map[tool.ToolName]bool, len(tool.AllInsightToolNames))
	for _, n := range tool.AllInsightToolNames {
		allowedInsight[n] = true
	}

	var invalid []string
	for _, name := range toolNames {
		switch t {
		case TypeInsight:
			if !allowedInsight[name] {
				invalid = append(invalid, string(name))
			}
		default: // TypeNormal and any other resolved value
			if name != tool.ToolNameAll && !allowedNormal[name] {
				invalid = append(invalid, string(name))
			}
		}
	}

	if len(invalid) == 0 {
		return nil
	}

	if t == TypeInsight {
		valid := make([]string, 0, len(tool.AllInsightToolNames))
		for _, n := range tool.AllInsightToolNames {
			valid = append(valid, string(n))
		}
		return fmt.Errorf(
			"invalid tool_names for type=insight: %q is not an Insight tool (valid: %s)",
			strings.Join(invalid, ", "), strings.Join(valid, ", "),
		)
	}

	valid := make([]string, 0, len(tool.AllToolNames)+1)
	valid = append(valid, string(tool.ToolNameAll))
	for _, n := range tool.AllToolNames {
		valid = append(valid, string(n))
	}
	return fmt.Errorf(
		"invalid tool_names for type=normal: %q is not a Normal tool (valid: %s)",
		strings.Join(invalid, ", "), strings.Join(valid, ", "),
	)
}
