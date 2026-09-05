package ai

import (
	"testing"

	"monorepo/bin-ai-manager/models/tool"
)

func TestAllowedToolNames(t *testing.T) {
	tests := []struct {
		name   string
		aiType Type

		wantAllowed    []tool.ToolName
		wantDisallowed []tool.ToolName
	}{
		{
			name:           "normal_allows_normal_tools_only",
			aiType:         TypeNormal,
			wantAllowed:    tool.AllToolNames,
			wantDisallowed: tool.AllInsightToolNames,
		},
		{
			name:           "insight_allows_insight_tools_only",
			aiType:         TypeInsight,
			wantAllowed:    tool.AllInsightToolNames,
			wantDisallowed: tool.AllToolNames,
		},
		{
			name:           "unknown_type_denies_everything",
			aiType:         Type("some_future_type"),
			wantDisallowed: append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...),
		},
		{
			name:           "type_none_denies_everything",
			aiType:         TypeNone,
			wantDisallowed: append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := AllowedToolNames(tt.aiType)

			for _, n := range tt.wantAllowed {
				if !allowed[n] {
					t.Errorf("AllowedToolNames(%v)[%v] = false, want true", tt.aiType, n)
				}
			}
			for _, n := range tt.wantDisallowed {
				if allowed[n] {
					t.Errorf("AllowedToolNames(%v)[%v] = true, want false", tt.aiType, n)
				}
			}

			// "all" is a selector, never itself a member of the whitelist.
			if allowed[tool.ToolNameAll] {
				t.Errorf("AllowedToolNames(%v)[ToolNameAll] = true, want false (all is a selector, not a concrete tool name)", tt.aiType)
			}
		})
	}
}

// TestAllInsightToolNamesAreReadOnly is the recommended (design §2.6) cheap
// regression guard: every entry in tool.AllInsightToolNames must have no
// side effects outside the session's own message/expression surface, since
// the "all" selector automatically grants any future addition to every
// Insight AI already storing tool_names=["all"], with no re-consent step.
// (Reworded from "must be a read-only tool" by docs/plans/
// 2026-09-04-insight-assistant-emit-info-card-design.md §1.4: a literal
// read-only reading would incorrectly exclude emit_info_card, which writes
// a message -- but only into its own session's message stream, the same
// surface a plain assistant-text turn already writes to.) This is a
// hardcoded allowlist of known-safe names (not a structural Tool metadata
// flag -- deferred per design §2.6/non-goals) so adding an unsafe tool to
// AllInsightToolNames without updating this test fails loudly here rather
// than silently shipping.
func TestAllInsightToolNamesAreReadOnly(t *testing.T) {
	knownReadOnly := map[tool.ToolName]bool{
		tool.ToolNameGetContactInteractions: true,
		tool.ToolNameGetConversationContent: true,
		tool.ToolNameGetRelatedCases:        true,
		tool.ToolNameGetCaseNotes:           true,
		tool.ToolNameGetContactProfile:      true,
		tool.ToolNameGetCallTranscript:      true,
		tool.ToolNameEmitInfoCard:           true,
	}

	// Sanctioned write exceptions -- a SEPARATE map, deliberately, so this test
	// keeps failing loudly for any other write tool added to
	// AllInsightToolNames. Adding a name here requires the same explicit
	// design-level justification notify_agent got (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.5.2): its only effect
	// must be on the AIcall's own conversation thread -- no external action, no
	// customer-data mutation, no spend.
	knownSanctionedWrite := map[tool.ToolName]bool{
		tool.ToolNameNotifyAgent: true,
	}

	for _, n := range tool.AllInsightToolNames {
		if !knownReadOnly[n] && !knownSanctionedWrite[n] {
			t.Errorf("tool.AllInsightToolNames contains %q, which is in neither this test's known-read-only allowlist nor its sanctioned-write allowlist -- "+
				"verify it has no side effects outside the session's own message/expression surface, then add it to the right map", n)
		}
	}
}

// TestValidateToolNames_WriteToolNeverAllowedForInsight guards against a
// write-capable Normal-only tool ever being smuggled into
// tool.AllInsightToolNames: every tool.AllToolNames entry that is NOT also
// in tool.AllInsightToolNames must be rejected for TypeInsight.
func TestValidateToolNames_WriteToolNeverAllowedForInsight(t *testing.T) {
	insightAllowed := toSet(tool.AllInsightToolNames)

	for _, writeTool := range tool.AllToolNames {
		if insightAllowed[writeTool] {
			continue
		}
		t.Run(string(writeTool), func(t *testing.T) {
			if err := ValidateToolNames(TypeInsight, []tool.ToolName{writeTool}); err == nil {
				t.Errorf("ValidateToolNames(TypeInsight, [%q]) = nil error, want rejection -- "+
					"write-capable tool must never be allowed for Insight AI", writeTool)
			}
		})
	}
}
