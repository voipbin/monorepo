package toolhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	aitool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"
)

func TestToolHandler_FetchTools(t *testing.T) {
	tests := []struct {
		name string

		responseTools []aitool.Tool
		responseErr   error

		expectErr bool
	}{
		{
			name: "normal - fetches all tools",

			responseTools: []aitool.Tool{
				{
					Name:        aitool.ToolNameConnectCall,
					Description: "Connects to another endpoint",
					Parameters:  map[string]any{"type": "object"},
				},
				{
					Name:        aitool.ToolNameSendEmail,
					Description: "Sends an email",
					Parameters:  map[string]any{"type": "object"},
				},
			},
			responseErr: nil,

			expectErr: false,
		},
		{
			name: "empty tools",

			responseTools: []aitool.Tool{},
			responseErr:   nil,

			expectErr: false,
		},
		{
			name: "error from ai-manager",

			responseTools: nil,
			responseErr:   errors.New("connection error"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRequest := requesthandler.NewMockRequestHandler(mc)
			h := NewToolHandler(mockRequest)
			ctx := context.Background()

			mockRequest.EXPECT().AIV1ToolList(ctx).Return(tt.responseTools, tt.responseErr)

			err := h.FetchTools(ctx)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify tools were cached: GetByNames(TypeNormal, ["all"]) returns
			// everything cached that is also Normal-whitelisted. Both fixture
			// tools here are in tool.AllToolNames, so this is equivalent to the
			// old GetAll() check.
			got := h.GetByNames(amai.TypeNormal, []aitool.ToolName{aitool.ToolNameAll})
			if len(got) != len(tt.responseTools) {
				t.Errorf("GetByNames(TypeNormal, [\"all\"]) returned %d tools, want %d", len(got), len(tt.responseTools))
			}
		})
	}
}

func TestToolHandler_GetByNames(t *testing.T) {
	cachedTools := []aitool.Tool{
		{Name: aitool.ToolNameConnectCall, Description: "Connect call"},
		{Name: aitool.ToolNameSendEmail, Description: "Send email"},
		{Name: aitool.ToolNameSendMessage, Description: "Send message"},
		{Name: aitool.ToolNameSetVariables, Description: "Set variables"},
		{Name: aitool.ToolNameGetContactInteractions, Description: "Get contact interactions"},
		{Name: aitool.ToolNameGetConversationContent, Description: "Get conversation content"},
	}

	tests := []struct {
		name string

		cachedTools []aitool.Tool
		aiType      amai.Type
		names       []aitool.ToolName

		wantCount int
		wantNames []aitool.ToolName
	}{
		{
			name: "empty names returns empty slice",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{},

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "nil names returns empty slice",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       nil,

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "normal all returns all normal tools, not insight tools",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{aitool.ToolNameAll},

			wantCount: 4,
			wantNames: []aitool.ToolName{
				aitool.ToolNameConnectCall, aitool.ToolNameSendEmail,
				aitool.ToolNameSendMessage, aitool.ToolNameSetVariables,
			},
		},
		{
			name: "insight all returns only insight tools, never normal tools (leak regression guard)",

			cachedTools: cachedTools,
			aiType:      amai.TypeInsight,
			names:       []aitool.ToolName{aitool.ToolNameAll},

			wantCount: 2,
			wantNames: []aitool.ToolName{
				aitool.ToolNameGetContactInteractions, aitool.ToolNameGetConversationContent,
			},
		},
		{
			name: "single tool name",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall},

			wantCount: 1,
			wantNames: []aitool.ToolName{aitool.ToolNameConnectCall},
		},
		{
			name: "multiple tool names",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameSendEmail},

			wantCount: 2,
			wantNames: []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameSendEmail},
		},
		{
			name: "all with other names returns all (of the allowed type)",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameAll},

			wantCount: 4,
			wantNames: []aitool.ToolName{
				aitool.ToolNameConnectCall, aitool.ToolNameSendEmail,
				aitool.ToolNameSendMessage, aitool.ToolNameSetVariables,
			},
		},
		{
			name: "non-existent tool name returns empty",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{"non_existent_tool"},

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "defense-in-depth: normal AI explicitly requesting an insight-only tool by name is denied",

			cachedTools: cachedTools,
			aiType:      amai.TypeNormal,
			names:       []aitool.ToolName{aitool.ToolNameGetContactInteractions},

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "defense-in-depth: insight AI explicitly requesting a normal-only tool by name is denied",

			cachedTools: cachedTools,
			aiType:      amai.TypeInsight,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall},

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "unknown AI type denies everything, even with explicit names",

			cachedTools: cachedTools,
			aiType:      amai.Type("some_future_type"),
			names:       []aitool.ToolName{aitool.ToolNameAll},

			wantCount: 0,
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRequest := requesthandler.NewMockRequestHandler(mc)
			h := &toolHandler{
				requestHandler: mockRequest,
				tools:          tt.cachedTools,
			}

			got := h.GetByNames(tt.aiType, tt.names)

			if len(got) != tt.wantCount {
				t.Errorf("GetByNames() returned %d tools, want %d", len(got), tt.wantCount)
				return
			}

			if tt.wantNames != nil {
				gotNames := make([]aitool.ToolName, len(got))
				for i, tool := range got {
					gotNames[i] = tool.Name
				}

				if !reflect.DeepEqual(sortToolNames(gotNames), sortToolNames(tt.wantNames)) {
					t.Errorf("GetByNames() = %v, want %v", gotNames, tt.wantNames)
				}
			}
		})
	}
}

// sortToolNames sorts tool names for comparison
func sortToolNames(names []aitool.ToolName) []aitool.ToolName {
	result := make([]aitool.ToolName, len(names))
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
