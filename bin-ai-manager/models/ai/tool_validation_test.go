package ai

import (
	"testing"

	"monorepo/bin-ai-manager/models/tool"
)

func TestValidateToolNames(t *testing.T) {
	tests := []struct {
		name      string
		aiType    Type
		toolNames []tool.ToolName
		wantError bool
	}{
		{
			name:      "normal_with_normal_tools_ok",
			aiType:    TypeNormal,
			toolNames: []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameSendEmail},
			wantError: false,
		},
		{
			name:      "normal_with_all_ok",
			aiType:    TypeNormal,
			toolNames: []tool.ToolName{tool.ToolNameAll},
			wantError: false,
		},
		{
			name:      "normal_with_insight_tool_rejected",
			aiType:    TypeNormal,
			toolNames: []tool.ToolName{tool.ToolNameGetContactInteractions},
			wantError: true,
		},
		{
			name:      "normal_with_unknown_name_rejected",
			aiType:    TypeNormal,
			toolNames: []tool.ToolName{"bogus_tool"},
			wantError: true,
		},
		{
			name:      "insight_with_insight_tools_ok",
			aiType:    TypeInsight,
			toolNames: []tool.ToolName{tool.ToolNameGetContactInteractions, tool.ToolNameGetConversationContent},
			wantError: false,
		},
		{
			name:      "insight_with_all_rejected",
			aiType:    TypeInsight,
			toolNames: []tool.ToolName{tool.ToolNameAll},
			wantError: true,
		},
		{
			name:      "insight_with_normal_tool_rejected",
			aiType:    TypeInsight,
			toolNames: []tool.ToolName{tool.ToolNameSendEmail},
			wantError: true,
		},
		{
			name:      "insight_with_unknown_name_rejected",
			aiType:    TypeInsight,
			toolNames: []tool.ToolName{"bogus_tool"},
			wantError: true,
		},
		{
			name:      "nil_toolnames_ok_for_normal",
			aiType:    TypeNormal,
			toolNames: nil,
			wantError: false,
		},
		{
			name:      "empty_toolnames_ok_for_insight",
			aiType:    TypeInsight,
			toolNames: []tool.ToolName{},
			wantError: false,
		},
		{
			name:      "normal_duplicate_names_ok",
			aiType:    TypeNormal,
			toolNames: []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameConnectCall},
			wantError: false,
		},
		{
			name:      "insight_all_mixed_with_insight_tool_rejected",
			aiType:    TypeInsight,
			toolNames: []tool.ToolName{tool.ToolNameAll, tool.ToolNameGetContactInteractions},
			wantError: true,
		},
		{
			name:      "normal_all_mixed_with_unknown_rejected",
			aiType:    TypeNormal,
			toolNames: []tool.ToolName{tool.ToolNameAll, "bogus_tool"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolNames(tt.aiType, tt.toolNames)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateToolNames(%v, %v) error = %v, wantError %v", tt.aiType, tt.toolNames, err, tt.wantError)
			}
		})
	}
}
