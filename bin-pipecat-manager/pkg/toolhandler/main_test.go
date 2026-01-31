package toolhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

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

			// Verify tools were cached
			got := h.GetAll()
			if !reflect.DeepEqual(got, tt.responseTools) {
				t.Errorf("GetAll() = %v, want %v", got, tt.responseTools)
			}
		})
	}
}

func TestToolHandler_GetAll(t *testing.T) {
	tests := []struct {
		name string

		cachedTools []aitool.Tool
		want        []aitool.Tool
	}{
		{
			name: "returns all cached tools",

			cachedTools: []aitool.Tool{
				{
					Name:        aitool.ToolNameConnectCall,
					Description: "Connects to another endpoint",
				},
				{
					Name:        aitool.ToolNameSendEmail,
					Description: "Sends an email",
				},
			},
			want: []aitool.Tool{
				{
					Name:        aitool.ToolNameConnectCall,
					Description: "Connects to another endpoint",
				},
				{
					Name:        aitool.ToolNameSendEmail,
					Description: "Sends an email",
				},
			},
		},
		{
			name: "empty cache returns empty slice",

			cachedTools: []aitool.Tool{},
			want:        []aitool.Tool{},
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

			got := h.GetAll()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAll() = %v, want %v", got, tt.want)
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
	}

	tests := []struct {
		name string

		cachedTools []aitool.Tool
		names       []aitool.ToolName

		wantCount int
		wantNames []aitool.ToolName
	}{
		{
			name: "empty names returns empty slice",

			cachedTools: cachedTools,
			names:       []aitool.ToolName{},

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "nil names returns empty slice",

			cachedTools: cachedTools,
			names:       nil,

			wantCount: 0,
			wantNames: nil,
		},
		{
			name: "all returns all tools",

			cachedTools: cachedTools,
			names:       []aitool.ToolName{aitool.ToolNameAll},

			wantCount: 4,
			wantNames: nil, // Don't check specific names, just count
		},
		{
			name: "single tool name",

			cachedTools: cachedTools,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall},

			wantCount: 1,
			wantNames: []aitool.ToolName{aitool.ToolNameConnectCall},
		},
		{
			name: "multiple tool names",

			cachedTools: cachedTools,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameSendEmail},

			wantCount: 2,
			wantNames: []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameSendEmail},
		},
		{
			name: "all with other names returns all",

			cachedTools: cachedTools,
			names:       []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameAll},

			wantCount: 4,
			wantNames: nil, // Don't check specific names, just count
		},
		{
			name: "non-existent tool name returns empty",

			cachedTools: cachedTools,
			names:       []aitool.ToolName{"non_existent_tool"},

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

			got := h.GetByNames(tt.names)

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
