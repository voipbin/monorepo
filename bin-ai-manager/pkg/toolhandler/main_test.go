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
