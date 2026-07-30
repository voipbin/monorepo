package ai

import (
	"fmt"

	"monorepo/bin-ai-manager/models/tool"

	"github.com/sirupsen/logrus"
)

// toSet converts a slice of tool names into a lookup set.
func toSet(names []tool.ToolName) map[tool.ToolName]bool {
	set := make(map[tool.ToolName]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

// AllowedToolNames returns the tool whitelist for a given AI type -- the
// single source of truth consumed by both ValidateToolNames (save-time
// validation, this package) and bin-pipecat-manager's runtime tool
// expansion (GetByNames). Unknown/future types deny-by-default (empty set)
// rather than falling back to the widest (Normal, write-capable) set.
//
// tool.ToolNameAll ("all") is a SELECTOR, not a concrete tool name --
// tool.AllToolNames / tool.AllInsightToolNames correctly do not contain it.
// AllowedToolNames only ever answers "is this concrete tool name allowed
// for type t"; it is never asked whether "all" itself is a member.
//
// See docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md
// §2.2 for the full rationale, including why this is keyed on the Type enum
// (in models/ai) rather than a bool in models/tool: a bool collapses any
// future third AIType into "Normal" (the write-capable set) by
// construction, which is the opposite of deny-by-default.
func AllowedToolNames(t Type) map[tool.ToolName]bool {
	switch t {
	case TypeNormal:
		return toSet(tool.AllToolNames)
	case TypeInsight:
		return toSet(tool.AllInsightToolNames)
	default:
		logrus.WithField("ai_type", t).Warnf("unknown AI type encountered in tool whitelist resolution, denying all tools")
		UnknownAITypeToolDenialTotal.Inc()
		return map[tool.ToolName]bool{}
	}
}

// ValidateToolNames returns an error if any name in toolNames is not
// permitted for the given (already-resolved) Type. See
// docs/plans/2026-07-14-insight-ai-tool-name-validation-design.md for the
// original rationale (VOIP-1257) and
// docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §2.2
// for the AllowedToolNames-based refactor that also grants Insight AI the
// ability to store tool_names=["all"].
//
// Rules:
//   - tool.ToolNameAll is always structurally valid input, for either Type --
//     it is a selector expanded at runtime by bin-pipecat-manager's
//     GetByNames against AllowedToolNames(t), never a concrete tool name
//     checked for set membership here.
//   - Every other name must be a member of AllowedToolNames(t) for the
//     given Type. A name that is neither a known tool for this Type nor
//     ToolNameAll is rejected (closes a pre-existing "unknown tool name
//     silently accepted" gap).
//   - nil/empty toolNames is always valid for either Type.
//
// The whole toolNames slice is checked (not short-circuited on the first
// valid or invalid element), so a mixed list such as
// ["all", "bogus_tool"] is rejected because "bogus_tool" is invalid even
// though "all" is valid.
func ValidateToolNames(t Type, toolNames []tool.ToolName) error {
	if len(toolNames) == 0 {
		return nil
	}

	allowed := AllowedToolNames(t)

	var invalid []tool.ToolName
	for _, name := range toolNames {
		if name == tool.ToolNameAll {
			continue
		}
		if !allowed[name] {
			invalid = append(invalid, name)
		}
	}

	if len(invalid) == 0 {
		return nil
	}

	return fmt.Errorf("invalid tool_names for type=%s: %v is not allowed", t, invalid)
}
