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
// regression guard: every entry in tool.AllInsightToolNames must be a
// read-only tool, since the "all" selector automatically grants any future
// addition to every Insight AI already storing tool_names=["all"], with no
// re-consent step. This is a hardcoded allowlist of known-read-only names
// (not a structural Tool metadata flag -- deferred per design §2.6/non-goals)
// so adding a write-capable tool to AllInsightToolNames without updating
// this test fails loudly here rather than silently shipping.
func TestAllInsightToolNamesAreReadOnly(t *testing.T) {
	knownReadOnly := map[tool.ToolName]bool{
		tool.ToolNameGetContactInteractions: true,
		tool.ToolNameGetConversationContent: true,
		tool.ToolNameGetRelatedCases:        true,
		tool.ToolNameGetCaseNotes:           true,
	}

	for _, n := range tool.AllInsightToolNames {
		if !knownReadOnly[n] {
			t.Errorf("tool.AllInsightToolNames contains %q, which is not in this test's known-read-only allowlist -- "+
				"verify it has no side effects, then add it to knownReadOnly in this test", n)
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
