package ai

import (
	"testing"

	"monorepo/bin-ai-manager/models/tool"
)

// withoutNames returns names minus the entries of exclude.
func withoutNames(names, exclude []tool.ToolName) []tool.ToolName {
	ex := toSet(exclude)
	res := []tool.ToolName{}
	for _, n := range names {
		if !ex[n] {
			res = append(res, n)
		}
	}
	return res
}

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
			name:        "insight_allows_insight_tools_and_opt_in_tools_only",
			aiType:      TypeInsight,
			wantAllowed: append(append([]tool.ToolName{}, tool.AllInsightToolNames...), tool.InsightOptInToolNames...),
			// every Normal tool except the opt-in ones stays denied.
			wantDisallowed: withoutNames(tool.AllToolNames, tool.InsightOptInToolNames),
		},
		{
			name:           "unknown_type_denies_everything",
			aiType:         Type("some_future_type"),
			wantDisallowed: append(append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...), tool.ToolNameDraftMessage),
		},
		{
			name:           "type_none_denies_everything",
			aiType:         TypeNone,
			wantDisallowed: append(append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...), tool.ToolNameDraftMessage),
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
// in tool.AllInsightToolNames must be rejected for TypeInsight, except the
// explicit opt-in tools (VOIP-1704), which are accepted by name and pinned by
// Test_InsightOptInToolNames.
func TestValidateToolNames_WriteToolNeverAllowedForInsight(t *testing.T) {
	insightAllowed := toSet(tool.AllInsightToolNames)
	for _, n := range tool.InsightOptInToolNames {
		insightAllowed[n] = true
	}

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

// Test_InsightOptInToolNames pins the opt-in list: adding a tool to it grants
// a side-effecting capability on an explicit tool_names entry, so it must be
// a deliberate change that updates this test.
func Test_InsightOptInToolNames(t *testing.T) {
	want := map[tool.ToolName]bool{
		tool.ToolNameSendMessage:  true,
		tool.ToolNameDraftMessage: true,
	}
	if len(tool.InsightOptInToolNames) != len(want) {
		t.Fatalf("tool.InsightOptInToolNames = %v, want exactly %d known names", tool.InsightOptInToolNames, len(want))
	}
	for _, n := range tool.InsightOptInToolNames {
		if !want[n] {
			t.Errorf("tool.InsightOptInToolNames contains unexpected %q", n)
		}
	}

	// opt-in tools are never part of the "all" selector or of AllInsightToolNames.
	insightAll := toSet(tool.AllInsightToolNames)
	for _, n := range tool.InsightOptInToolNames {
		if insightAll[n] {
			t.Errorf("opt-in tool %q must not be in tool.AllInsightToolNames", n)
		}
		if ExpandAllToolNames(TypeInsight)[n] {
			t.Errorf("ExpandAllToolNames(TypeInsight) must not contain opt-in tool %q", n)
		}
		if err := ValidateToolNames(TypeInsight, []tool.ToolName{n}); err != nil {
			t.Errorf("ValidateToolNames(TypeInsight, [%q]) = %v, want nil", n, err)
		}
	}
}

func Test_ExpandAllToolNames(t *testing.T) {
	tests := []struct {
		name         string
		aiType       Type
		want         []tool.ToolName
		wantNot      []tool.ToolName
		wantSizeSame bool
	}{
		{"normal", TypeNormal, tool.AllToolNames, []tool.ToolName{tool.ToolNameDraftMessage}, true},
		{"insight", TypeInsight, tool.AllInsightToolNames, tool.InsightOptInToolNames, true},
		{"unknown", Type("future"), nil, append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandAllToolNames(tt.aiType)
			for _, n := range tt.want {
				if !got[n] {
					t.Errorf("ExpandAllToolNames(%v)[%v] = false, want true", tt.aiType, n)
				}
			}
			for _, n := range tt.wantNot {
				if got[n] {
					t.Errorf("ExpandAllToolNames(%v)[%v] = true, want false", tt.aiType, n)
				}
			}
			if len(got) != len(tt.want) {
				t.Errorf("ExpandAllToolNames(%v) size = %d, want %d", tt.aiType, len(got), len(tt.want))
			}
		})
	}

	// Normal "all" still includes send_message (it is a Normal tool), but not draft_message.
	if !ExpandAllToolNames(TypeNormal)[tool.ToolNameSendMessage] {
		t.Errorf("Normal all must include send_message")
	}
	// draft_message is not allowed for Normal by name either.
	if err := ValidateToolNames(TypeNormal, []tool.ToolName{tool.ToolNameDraftMessage}); err == nil {
		t.Errorf("draft_message must be rejected for a Normal AI")
	}
}
