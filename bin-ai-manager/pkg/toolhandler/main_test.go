package toolhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
)

func TestToolHandler_GetAll(t *testing.T) {
	tests := []struct {
		name string
		want []tool.Tool
	}{
		{
			name: "returns all tools",
			want: toolDefinitions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewToolHandler()
			got := h.GetAll()

			if len(got) != len(tt.want) {
				t.Errorf("GetAll() returned %d tools, want %d", len(got), len(tt.want))
				return
			}

			// Verify all expected tools are present
			for _, wantTool := range tt.want {
				found := false
				for _, gotTool := range got {
					if gotTool.Name == wantTool.Name {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetAll() missing tool %s", wantTool.Name)
				}
			}
		})
	}
}

func TestToolHandler_GetByNames(t *testing.T) {
	tests := []struct {
		name      string
		aiType    ai.Type
		names     []tool.ToolName
		wantCount int
		wantNames []tool.ToolName
	}{
		{
			name:      "empty names returns nil",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{},
			wantCount: 0,
			wantNames: nil,
		},
		{
			name:      "nil names returns nil",
			aiType:    ai.TypeNormal,
			names:     nil,
			wantCount: 0,
			wantNames: nil,
		},
		{
			name:      "all returns all normal tools",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{tool.ToolNameAll},
			wantCount: len(tool.AllToolNames),
			wantNames: nil, // Don't check specific names, just count
		},
		{
			name:      "single tool name",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{tool.ToolNameConnectCall},
			wantCount: 1,
			wantNames: []tool.ToolName{tool.ToolNameConnectCall},
		},
		{
			name:      "multiple tool names",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameSendEmail, tool.ToolNameSendMessage},
			wantCount: 3,
			wantNames: []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameSendEmail, tool.ToolNameSendMessage},
		},
		{
			name:      "all with other names returns all",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameAll},
			wantCount: len(tool.AllToolNames),
			wantNames: nil, // Don't check specific names, just count
		},
		{
			name:      "non-existent tool name returns empty",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{"non_existent_tool"},
			wantCount: 0,
			wantNames: nil,
		},
		{
			// B4: an Insight AI storing tool_names:["all"] must never resolve
			// Normal-only built-ins (e.g. connect_call) -- AllowedToolNames(aiType)
			// is re-applied regardless of what names claims.
			name:      "insight ai with all does not leak normal-only tools",
			aiType:    ai.TypeInsight,
			names:     []tool.ToolName{tool.ToolNameAll},
			wantCount: -1, // checked via wantNames below, not an exact count
			wantNames: nil,
		},
		{
			name:      "insight ai explicit normal-only tool name is denied",
			aiType:    ai.TypeInsight,
			names:     []tool.ToolName{tool.ToolNameConnectCall},
			wantCount: 0,
			wantNames: nil,
		},
		{
			name:      "unknown ai type denies all",
			aiType:    ai.Type("unknown"),
			names:     []tool.ToolName{tool.ToolNameAll},
			wantCount: 0,
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewToolHandler()
			got := h.GetByNames(tt.aiType, tt.names)

			if tt.name == "insight ai with all does not leak normal-only tools" {
				for _, gt := range got {
					if gt.Name == tool.ToolNameConnectCall {
						t.Errorf("GetByNames() leaked normal-only tool %s to an insight AI", gt.Name)
					}
				}
				return
			}

			if tt.wantCount >= 0 && len(got) != tt.wantCount {
				t.Errorf("GetByNames() returned %d tools, want %d", len(got), tt.wantCount)
				return
			}

			if tt.wantNames != nil {
				gotNames := make([]tool.ToolName, len(got))
				for i, t := range got {
					gotNames[i] = t.Name
				}

				if !reflect.DeepEqual(sortToolNames(gotNames), sortToolNames(tt.wantNames)) {
					t.Errorf("GetByNames() = %v, want %v", gotNames, tt.wantNames)
				}
			}
		})
	}
}

// sortToolNames sorts tool names for comparison
func sortToolNames(names []tool.ToolName) []tool.ToolName {
	result := make([]tool.ToolName, len(names))
	copy(result, names)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func TestAllToolNames(t *testing.T) {
	// Verify AllToolNames contains all expected tool names
	expectedNames := []tool.ToolName{
		tool.ToolNameConnectCall,
		tool.ToolNameCreateCall,
		tool.ToolNameGetVariables,
		tool.ToolNameGetAIcallMessages,
		tool.ToolNameSendEmail,
		tool.ToolNameSendMessage,
		tool.ToolNameSetVariables,
		tool.ToolNameStopFlow,
		tool.ToolNameStopMedia,
		tool.ToolNameStopService,
		tool.ToolNameSearchKnowledge,
		tool.ToolNameGetCorrelation,
		tool.ToolNameGetResource,
		tool.ToolNameDescribeAction,
		tool.ToolNameCaseCreate,
		tool.ToolNameListQueues,
		tool.ToolNameJoinQueue,
	}

	if len(tool.AllToolNames) != len(expectedNames) {
		t.Errorf("AllToolNames has %d items, want %d", len(tool.AllToolNames), len(expectedNames))
	}

	for _, name := range expectedNames {
		found := false
		for _, allName := range tool.AllToolNames {
			if allName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllToolNames missing %s", name)
		}
	}
}

func nameSet(tools []tool.Tool) map[tool.ToolName]bool {
	res := map[tool.ToolName]bool{}
	for _, t := range tools {
		res[t.Name] = true
	}
	return res
}

// Test_ToolHandler_GetByNames_InsightMessageTools pins the contract of the
// Insight SMS tools (VOIP-1704, VOIP-1708): the "all" selector returns them
// together with the other Insight tools (10 in total), an explicit name still
// returns just that tool, and a mixed list returns no duplicates. A Normal AI
// never gets draft_message.
func Test_ToolHandler_GetByNames_InsightMessageTools(t *testing.T) {
	h := NewToolHandler()

	tests := []struct {
		name      string
		aiType    ai.Type
		names     []tool.ToolName
		want      []tool.ToolName
		wantNot   []tool.ToolName
		wantCount int
	}{
		{
			name:      "insight all includes the message tools",
			aiType:    ai.TypeInsight,
			names:     []tool.ToolName{tool.ToolNameAll},
			want:      append(append([]tool.ToolName{}, tool.AllInsightToolNames...), tool.InsightMessageToolNames...),
			wantCount: 10,
		},
		{
			name:      "insight explicit draft_message",
			aiType:    ai.TypeInsight,
			names:     []tool.ToolName{tool.ToolNameDraftMessage},
			want:      []tool.ToolName{tool.ToolNameDraftMessage},
			wantNot:   []tool.ToolName{tool.ToolNameSendMessage},
			wantCount: 1,
		},
		{
			name:      "insight explicit send_message",
			aiType:    ai.TypeInsight,
			names:     []tool.ToolName{tool.ToolNameSendMessage},
			want:      []tool.ToolName{tool.ToolNameSendMessage},
			wantNot:   []tool.ToolName{tool.ToolNameDraftMessage},
			wantCount: 1,
		},
		{
			name:      "insight all plus send_message",
			aiType:    ai.TypeInsight,
			names:     []tool.ToolName{tool.ToolNameAll, tool.ToolNameSendMessage},
			want:      append(append([]tool.ToolName{}, tool.AllInsightToolNames...), tool.InsightMessageToolNames...),
			wantCount: 10,
		},
		{
			name:      "normal explicit draft_message is not allowed",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{tool.ToolNameDraftMessage},
			wantNot:   []tool.ToolName{tool.ToolNameDraftMessage},
			wantCount: 0,
		},
		{
			name:      "normal all has no draft_message",
			aiType:    ai.TypeNormal,
			names:     []tool.ToolName{tool.ToolNameAll},
			want:      tool.AllToolNames,
			wantNot:   []tool.ToolName{tool.ToolNameDraftMessage},
			wantCount: len(tool.AllToolNames),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.GetByNames(tt.aiType, tt.names)
			set := nameSet(got)
			if len(got) != tt.wantCount {
				t.Errorf("GetByNames() returned %d tools, want %d", len(got), tt.wantCount)
			}
			for _, n := range tt.want {
				if !set[n] {
					t.Errorf("GetByNames() missing %s", n)
				}
			}
			for _, n := range tt.wantNot {
				if set[n] {
					t.Errorf("GetByNames() must not return %s", n)
				}
			}
		})
	}
}

// TestDraftMessageDefinition pins the draft_message definition contract.
func Test_DraftMessageDefinition(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameDraftMessage {
			def = &toolDefinitions[i]
		}
	}
	if def == nil {
		t.Fatalf("draft_message definition not found")
	}
	if !def.RunLLM {
		t.Errorf("draft_message RunLLM = false, want true")
	}
	required, _ := def.Parameters["required"].([]string)
	if len(required) != 2 || required[0] != "to" || required[1] != "text" {
		t.Errorf("draft_message required = %v, want [to text]", required)
	}
	props, _ := def.Parameters["properties"].(map[string]any)
	for _, k := range []string{"to", "text", "title", "run_llm"} {
		if _, ok := props[k]; !ok {
			t.Errorf("draft_message property %q missing", k)
		}
	}
}
