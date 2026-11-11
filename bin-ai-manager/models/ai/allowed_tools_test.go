package ai

import (
	"testing"

	"monorepo/bin-ai-manager/models/tool"
)

// insightExtraToolNames returns the Insight tools that send or propose a
// message: the SMS tools and the email tools.
func insightExtraToolNames() []tool.ToolName {
	return append(append(append([]tool.ToolName{}, tool.InsightMessageToolNames...), tool.InsightEmailToolNames...), tool.InsightCallToolNames...)
}

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
			wantDisallowed: append(withoutNames(tool.AllInsightToolNames, tool.AllToolNames), tool.ToolNameDraftMessage, tool.ToolNameDraftEmail, tool.ToolNameDraftCall),
		},
		{
			name:        "insight_allows_insight_tools_and_message_tools_only",
			aiType:      TypeInsight,
			wantAllowed: append(append([]tool.ToolName{}, tool.AllInsightToolNames...), insightExtraToolNames()...),
			// every Normal tool except the message, email and call ones stays denied.
			wantDisallowed: withoutNames(withoutNames(tool.AllToolNames, insightExtraToolNames()), tool.AllInsightToolNames),
		},
		{
			name:           "unknown_type_denies_everything",
			aiType:         Type("some_future_type"),
			wantDisallowed: append(append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...), tool.ToolNameDraftMessage, tool.ToolNameDraftEmail, tool.ToolNameDraftCall),
		},
		{
			name:           "type_none_denies_everything",
			aiType:         TypeNone,
			wantDisallowed: append(append(append([]tool.ToolName{}, tool.AllToolNames...), tool.AllInsightToolNames...), tool.ToolNameDraftMessage, tool.ToolNameDraftEmail, tool.ToolNameDraftCall),
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
// The SMS tools are NOT in this list and so are not covered by this guard;
// they are a separate list pinned by Test_InsightMessageToolNames.
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
		tool.ToolNameSearchKnowledge:        true,
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
// message tools (VOIP-1704, tool.InsightMessageToolNames) and the email tools
// (VOIP-1710, tool.InsightEmailToolNames) and the call tools (VOIP-1717,
// tool.InsightCallToolNames), which are accepted and pinned by
// Test_InsightMessageToolNames, Test_InsightEmailToolNames and
// Test_InsightCallToolNames.
func TestValidateToolNames_WriteToolNeverAllowedForInsight(t *testing.T) {
	insightAllowed := toSet(tool.AllInsightToolNames)
	for _, n := range insightExtraToolNames() {
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

// Test_InsightMessageToolNames pins the message list: adding a tool to it
// grants a side-effecting capability to every Insight AI stored with
// tool_names=["all"] (VOIP-1708) and bypasses TestAllInsightToolNamesAreReadOnly
// and TestValidateToolNames_WriteToolNeverAllowedForInsight, so it must be a
// deliberate change that updates this test.
func Test_InsightMessageToolNames(t *testing.T) {
	want := map[tool.ToolName]bool{
		tool.ToolNameSendMessage:  true,
		tool.ToolNameDraftMessage: true,
	}
	if len(tool.InsightMessageToolNames) != len(want) {
		t.Fatalf("tool.InsightMessageToolNames = %v, want exactly %d known names", tool.InsightMessageToolNames, len(want))
	}

	// The message tools are a separate list: never in AllInsightToolNames
	// (which only holds tools without outside side effects), always part of
	// the "all" selector and accepted by name.
	insightAll := toSet(tool.AllInsightToolNames)
	expanded := ExpandAllToolNames(TypeInsight)
	for _, n := range tool.InsightMessageToolNames {
		if !want[n] {
			t.Errorf("tool.InsightMessageToolNames contains unexpected %q", n)
		}
		if insightAll[n] {
			t.Errorf("message tool %q must not be in tool.AllInsightToolNames", n)
		}
		if !expanded[n] {
			t.Errorf("ExpandAllToolNames(TypeInsight) must contain message tool %q", n)
		}
		if err := ValidateToolNames(TypeInsight, []tool.ToolName{n}); err != nil {
			t.Errorf("ValidateToolNames(TypeInsight, [%q]) = %v, want nil", n, err)
		}
	}
}

// Test_InsightCallToolNames pins the call list (VOIP-1717): adding to it is a
// deliberate change. create_call has outside effects, so the list is separate
// from AllInsightToolNames; draft_call is not a Normal AI tool.
func Test_InsightCallToolNames(t *testing.T) {
	want := map[tool.ToolName]bool{
		tool.ToolNameCreateCall:     true,
		tool.ToolNameDraftCall:      true,
		tool.ToolNameDescribeAction: true,
	}
	if len(tool.InsightCallToolNames) != len(want) {
		t.Fatalf("tool.InsightCallToolNames = %v, want exactly %d known names", tool.InsightCallToolNames, len(want))
	}
	for _, n := range tool.InsightCallToolNames {
		if !want[n] {
			t.Errorf("tool.InsightCallToolNames contains unexpected %q", n)
		}
	}
	for _, n := range tool.AllInsightToolNames {
		if want[n] {
			t.Errorf("%q must not be in AllInsightToolNames", n)
		}
	}
}

// Test_InsightEmailToolNames pins the email list the same way
// Test_InsightMessageToolNames pins the message list (VOIP-1710): adding to it
// grants a side-effecting capability to every Insight AI stored with
// tool_names=["all"] and bypasses the read-only guards, so it must be a
// deliberate change that updates this test. draft_email is Insight only:
// it is not in tool.AllToolNames, and a Normal AI rejects it by name.
func Test_InsightEmailToolNames(t *testing.T) {
	want := map[tool.ToolName]bool{
		tool.ToolNameSendEmail:  true,
		tool.ToolNameDraftEmail: true,
	}
	if len(tool.InsightEmailToolNames) != len(want) {
		t.Fatalf("tool.InsightEmailToolNames = %v, want exactly %d known names", tool.InsightEmailToolNames, len(want))
	}

	insightAll := toSet(tool.AllInsightToolNames)
	expanded := ExpandAllToolNames(TypeInsight)
	for _, n := range tool.InsightEmailToolNames {
		if !want[n] {
			t.Errorf("tool.InsightEmailToolNames contains unexpected %q", n)
		}
		if insightAll[n] {
			t.Errorf("email tool %q must not be in tool.AllInsightToolNames", n)
		}
		if !expanded[n] {
			t.Errorf("ExpandAllToolNames(TypeInsight) must contain email tool %q", n)
		}
		if err := ValidateToolNames(TypeInsight, []tool.ToolName{n}); err != nil {
			t.Errorf("ValidateToolNames(TypeInsight, [%q]) = %v, want nil", n, err)
		}
	}

	// draft_email is not a Normal tool; send_email is (unchanged).
	if err := ValidateToolNames(TypeNormal, []tool.ToolName{tool.ToolNameDraftEmail}); err == nil {
		t.Errorf("draft_email must be rejected for a Normal AI")
	}
	if err := ValidateToolNames(TypeNormal, []tool.ToolName{tool.ToolNameSendEmail}); err != nil {
		t.Errorf("send_email must stay valid for a Normal AI, got %v", err)
	}
	for _, n := range tool.AllToolNames {
		if n == tool.ToolNameDraftEmail {
			t.Errorf("tool.AllToolNames must not contain draft_email")
		}
	}
	if ExpandAllToolNames(TypeNormal)[tool.ToolNameDraftEmail] {
		t.Errorf("the all selector of a Normal AI must not expand to draft_email")
	}
}

// Test_ExpandAllToolNames_InsightIsAllowedSet: for an Insight AI "all" means
// every tool it may use, so the selector set and the whitelist are equal, and
// together they are the 8 AllInsightToolNames plus the 2 message tools and the
// 2 email tools.
func Test_ExpandAllToolNames_InsightIsAllowedSet(t *testing.T) {
	expanded := ExpandAllToolNames(TypeInsight)
	allowed := AllowedToolNames(TypeInsight)

	if len(expanded) != len(allowed) {
		t.Fatalf("ExpandAllToolNames(TypeInsight) has %d names, AllowedToolNames(TypeInsight) has %d, want equal", len(expanded), len(allowed))
	}
	for n := range allowed {
		if !expanded[n] {
			t.Errorf("AllowedToolNames(TypeInsight) has %q but ExpandAllToolNames(TypeInsight) does not", n)
		}
	}
	if want := len(tool.AllInsightToolNames) + len(tool.InsightMessageToolNames) + len(tool.InsightEmailToolNames) + len(tool.InsightCallToolNames); len(expanded) != want || want != 16 {
		t.Errorf("ExpandAllToolNames(TypeInsight) has %d names, want %d (== 16)", len(expanded), want)
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
		{"normal", TypeNormal, tool.AllToolNames, []tool.ToolName{tool.ToolNameDraftMessage, tool.ToolNameDraftEmail}, true},
		{"insight", TypeInsight, append(append([]tool.ToolName{}, tool.AllInsightToolNames...), insightExtraToolNames()...), withoutNames(withoutNames(tool.AllToolNames, insightExtraToolNames()), tool.AllInsightToolNames), true},
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
