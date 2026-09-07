package toolhandler

import (
	"sort"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
)

// TestGetResourceEnumMatchesFetchers locks the get_resource JSON-schema enum
// in definitions.go to the aicallhandler fetcher table so the two cannot
// drift when Phase 2 adds resource types.
func TestGetResourceEnumMatchesFetchers(t *testing.T) {
	// find the get_resource definition
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameGetResource {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("get_resource tool definition not found in definitions.go")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource parameters has no properties map")
	}
	rt, ok := props["resource_type"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource has no resource_type property")
	}
	enum, ok := rt["enum"].([]string)
	if !ok {
		t.Fatalf("resource_type enum is not []string: %T", rt["enum"])
	}

	gotEnum := make([]string, len(enum))
	copy(gotEnum, enum)
	sort.Strings(gotEnum)

	want := aicallhandler.SupportedResourceTypes()

	if len(gotEnum) != len(want) {
		t.Fatalf("enum length %d != fetcher table length %d. enum: %v, fetchers: %v", len(gotEnum), len(want), gotEnum, want)
	}
	for i := range want {
		if gotEnum[i] != want[i] {
			t.Errorf("enum[%d] = %s, want %s (enum: %v, fetchers: %v)", i, gotEnum[i], want[i], gotEnum, want)
		}
	}
}

// TestGetResourceIncludeConfigSchema locks the include_config parameter shape
// in the get_resource JSON schema (design 2026-06-12 test 13): it must exist
// as a boolean property and must NOT be required (opt-in, default off).
func TestGetResourceIncludeConfigSchema(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameGetResource {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("get_resource tool definition not found in definitions.go")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource parameters has no properties map")
	}

	ic, ok := props["include_config"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource has no include_config property")
	}
	if got, _ := ic["type"].(string); got != "boolean" {
		t.Errorf("include_config type = %q, want \"boolean\"", got)
	}

	req, ok := def.Parameters["required"].([]string)
	if !ok {
		t.Fatalf("get_resource required is not []string: %T", def.Parameters["required"])
	}
	wantReq := []string{"resource_type", "resource_id"}
	if len(req) != len(wantReq) {
		t.Fatalf("required = %v, want exactly %v (include_config must stay optional)", req, wantReq)
	}
	for i := range wantReq {
		if req[i] != wantReq[i] {
			t.Errorf("required[%d] = %s, want %s", i, req[i], wantReq[i])
		}
	}
}

// TestCreateCallActionsEnumMatchesTypeListAll locks the create_call inline-actions
// 'type' JSON-schema enum to flow-manager's authoritative action.TypeListAll, so the
// two cannot drift when a new action type is added to flow-manager. The enum is what
// the LLM is offered; TypeListAll is what flow-manager ValidateActions accepts. They
// must stay identical (every offered type is accepted; no accepted type is hidden).
func TestCreateCallActionsEnumMatchesTypeListAll(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameCreateCall {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("create_call tool definition not found in definitions.go")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("create_call parameters has no properties map")
	}
	actions, ok := props["actions"].(map[string]any)
	if !ok {
		t.Fatalf("create_call has no actions property")
	}
	items, ok := actions["items"].(map[string]any)
	if !ok {
		t.Fatalf("create_call actions has no items map")
	}
	itemProps, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("create_call actions.items has no properties map")
	}
	typeProp, ok := itemProps["type"].(map[string]any)
	if !ok {
		t.Fatalf("create_call actions.items has no type property")
	}
	enum, ok := typeProp["enum"].([]string)
	if !ok {
		t.Fatalf("actions.items.type enum is not []string: %T", typeProp["enum"])
	}

	gotEnum := make([]string, len(enum))
	copy(gotEnum, enum)
	sort.Strings(gotEnum)

	want := make([]string, len(fmaction.TypeListAll))
	for i, ty := range fmaction.TypeListAll {
		want[i] = string(ty)
	}
	sort.Strings(want)

	if len(gotEnum) != len(want) {
		t.Fatalf("enum length %d != TypeListAll length %d. enum: %v, TypeListAll: %v", len(gotEnum), len(want), gotEnum, want)
	}
	for i := range want {
		if gotEnum[i] != want[i] {
			t.Errorf("enum[%d] = %s, want %s (enum: %v, TypeListAll: %v)", i, gotEnum[i], want[i], gotEnum, want)
		}
	}
}

// TestGetConversationContentSchema locks the get_conversation_content JSON
// schema after VOIP-1475: the tool takes an OPTIONAL conversation_id (the old
// required reference_id was a message id the backend has no route to resolve),
// so no parameter may be required. It also pins the two Description contracts
// the LLM relies on after VOIP-1479: the no-argument call is the DEFAULT (the
// production failure was an LLM that concluded the tool needed an id at all),
// and an unusable conversation_id falls back rather than dead-ending.
func TestGetConversationContentSchema(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameGetConversationContent {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("get_conversation_content tool definition not found in definitions.go")
	}

	const wantPrefix = "Returns the message thread of a conversation, oldest first. Call it with NO arguments to read the conversation this Case was created from; that is the default and the common case."
	if !strings.HasPrefix(def.Description, wantPrefix) {
		t.Errorf("get_conversation_content description must start with the no-argument sentence.\nwant prefix: %s\ngot: %s", wantPrefix, def.Description)
	}
	if !strings.Contains(def.Description, "falls back") {
		t.Errorf("get_conversation_content description must state the fallback. got: %s", def.Description)
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("get_conversation_content parameters has no properties map")
	}

	for _, name := range []string{"conversation_id", "limit", "run_llm"} {
		if _, ok := props[name].(map[string]any); !ok {
			t.Errorf("get_conversation_content has no %s property", name)
		}
	}
	if ct, _ := props["conversation_id"].(map[string]any); ct != nil {
		if got, _ := ct["type"].(string); got != "string" {
			t.Errorf("conversation_id type = %q, want \"string\"", got)
		}
	}
	if _, exists := props["reference_id"]; exists {
		t.Errorf("get_conversation_content must not advertise reference_id anymore (props: %v)", props)
	}

	if req, exists := def.Parameters["required"]; exists {
		got, ok := req.([]string)
		if !ok {
			t.Fatalf("get_conversation_content required is not []string: %T", req)
		}
		if len(got) != 0 {
			t.Errorf("required = %v, want absent or empty (conversation_id is optional)", got)
		}
	}
}

// TestInsightReadToolsRunLLMDescription pins the run_llm wording of the six
// Insight read tools after VOIP-1485. The production failure was an LLM that
// set run_llm=false on the first tool of a two-step chain because the old
// description only described the true branch ("reason about the retrieved X"),
// which reads as optional reasoning; the runner honoured false and the turn
// ended with no second tool call and no answer. Both texts must therefore keep
// stating what false costs. notify_agent and emit_info_card are deliberately
// out of scope and are asserted unchanged here so a future bulk edit cannot
// sweep them in.
//
// Coverage is derived from tool.AllInsightToolNames rather than hardcoded: the
// three groups below must partition that list exactly, so an Insight tool added
// there without a decision about which run_llm contract it carries fails this
// test instead of shipping with un-pinned wording.
func TestInsightReadToolsRunLLMDescription(t *testing.T) {
	const (
		wantDescriptionSubstring = "false ends the turn with no answer"
		wantParameterSubstring   = "False ends the turn with no further tool call and no answer"
	)

	insightReadTools := []tool.ToolName{
		tool.ToolNameGetContactInteractions,
		tool.ToolNameGetConversationContent,
		tool.ToolNameGetRelatedCases,
		tool.ToolNameGetCaseNotes,
		tool.ToolNameGetContactProfile,
		tool.ToolNameGetCallTranscript,
	}

	t.Run("covers_all_insight_tools", func(t *testing.T) {
		asserted := make(map[tool.ToolName]bool, len(insightReadTools)+2)
		for _, name := range insightReadTools {
			asserted[name] = true
		}
		asserted[tool.ToolNameEmitInfoCard] = true
		asserted[tool.ToolNameNotifyAgent] = true

		declared := make(map[tool.ToolName]bool, len(tool.AllInsightToolNames))
		for _, name := range tool.AllInsightToolNames {
			declared[name] = true
			if !asserted[name] {
				t.Errorf("%s is in tool.AllInsightToolNames but no group in this test pins its run_llm contract; add it to a group", name)
			}
		}
		for name := range asserted {
			if !declared[name] {
				t.Errorf("%s is asserted here but is not in tool.AllInsightToolNames; the groups have drifted from the source list", name)
			}
		}
	})

	for _, name := range insightReadTools {
		t.Run(string(name), func(t *testing.T) {
			def := findToolDefinition(t, name)

			if !def.RunLLM {
				t.Errorf("%s RunLLM = false, want true (the platform default must keep the turn alive)", name)
			}
			if !strings.Contains(def.Description, wantDescriptionSubstring) {
				t.Errorf("%s description must state what run_llm=false costs.\nwant substring: %s\ngot: %s", name, wantDescriptionSubstring, def.Description)
			}

			props, ok := def.Parameters["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s parameters has no properties map", name)
			}
			runLLM, ok := props["run_llm"].(map[string]any)
			if !ok {
				t.Fatalf("%s has no run_llm property", name)
			}
			got, _ := runLLM["description"].(string)
			if !strings.Contains(got, wantParameterSubstring) {
				t.Errorf("%s run_llm parameter description must state what false costs.\nwant substring: %s\ngot: %s", name, wantParameterSubstring, got)
			}
			if dflt, ok := runLLM["default"].(bool); !ok || !dflt {
				t.Errorf("%s run_llm schema default = %v, want true", name, runLLM["default"])
			}
		})
	}

	t.Run(string(tool.ToolNameNotifyAgent), func(t *testing.T) {
		def := findToolDefinition(t, tool.ToolNameNotifyAgent)

		if def.RunLLM {
			t.Errorf("notify_agent RunLLM = true, want false (the notification IS the output)")
		}
		if strings.Contains(def.Description, "ends the turn") {
			t.Errorf("notify_agent description must not carry the Insight run_llm wording. got: %s", def.Description)
		}
		props, ok := def.Parameters["properties"].(map[string]any)
		if !ok {
			t.Fatalf("notify_agent parameters has no properties map")
		}
		if _, exists := props["run_llm"]; exists {
			t.Errorf("notify_agent must not declare a run_llm parameter (props: %v)", props)
		}
	})

	t.Run(string(tool.ToolNameEmitInfoCard), func(t *testing.T) {
		def := findToolDefinition(t, tool.ToolNameEmitInfoCard)

		if !def.RunLLM {
			t.Errorf("emit_info_card RunLLM = false, want true")
		}
		if !strings.Contains(strings.ToLower(def.Description), "must not restate") {
			t.Errorf("emit_info_card description must keep its do-not-restate contract. got: %s", def.Description)
		}
	})
}

// findToolDefinition returns the definition registered in definitions.go for
// name, failing the test when it is absent.
func findToolDefinition(t *testing.T, name tool.ToolName) *tool.Tool {
	t.Helper()

	for i := range toolDefinitions {
		if toolDefinitions[i].Name == name {
			return &toolDefinitions[i]
		}
	}

	t.Fatalf("%s tool definition not found in definitions.go", name)
	return nil
}
