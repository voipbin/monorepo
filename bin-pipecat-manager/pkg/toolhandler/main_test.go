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
			name: "insight all returns only insight tools and the SMS and email tools, never other normal tools (leak regression guard)",

			cachedTools: cachedTools,
			aiType:      amai.TypeInsight,
			names:       []aitool.ToolName{aitool.ToolNameAll},

			// connect_call and set_variables are in the cache and stay refused: they are the leak sentinels.
			wantCount: 4,
			wantNames: []aitool.ToolName{
				aitool.ToolNameGetContactInteractions, aitool.ToolNameGetConversationContent,
				aitool.ToolNameSendMessage, aitool.ToolNameSendEmail,
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

// Test_ToolHandler_GetByNames_InsightMessageTools: the "all" selector returns
// the Insight SMS and email tools together with the other Insight tools
// (VOIP-1708, VOIP-1710, exactly 12 for a full cache); an explicit name still
// returns just that tool; a Normal AI never gets draft_message or draft_email.
func Test_ToolHandler_GetByNames_InsightMessageTools(t *testing.T) {
	cached := []aitool.Tool{
		{Name: aitool.ToolNameSendMessage},
		{Name: aitool.ToolNameDraftMessage},
		{Name: aitool.ToolNameSendEmail},
		{Name: aitool.ToolNameDraftEmail},
		{Name: aitool.ToolNameDraftCall},
		{Name: aitool.ToolNameCreateCall},
		{Name: aitool.ToolNameGetContactInteractions},
		{Name: aitool.ToolNameEmitInfoCard},
		{Name: aitool.ToolNameConnectCall},
	}

	tests := []struct {
		name    string
		aiType  amai.Type
		names   []aitool.ToolName
		want    []aitool.ToolName
		wantNot []aitool.ToolName
	}{
		{
			name:    "insight all includes the message and email tools",
			aiType:  amai.TypeInsight,
			names:   []aitool.ToolName{aitool.ToolNameAll},
			want:    []aitool.ToolName{aitool.ToolNameGetContactInteractions, aitool.ToolNameEmitInfoCard, aitool.ToolNameSendMessage, aitool.ToolNameDraftMessage, aitool.ToolNameSendEmail, aitool.ToolNameDraftEmail, aitool.ToolNameDraftCall, aitool.ToolNameCreateCall},
			wantNot: []aitool.ToolName{aitool.ToolNameConnectCall},
		},
		{
			name:    "insight explicit send_email",
			aiType:  amai.TypeInsight,
			names:   []aitool.ToolName{aitool.ToolNameSendEmail},
			want:    []aitool.ToolName{aitool.ToolNameSendEmail},
			wantNot: []aitool.ToolName{aitool.ToolNameDraftEmail, aitool.ToolNameEmitInfoCard},
		},
		{
			name:    "insight explicit draft_email",
			aiType:  amai.TypeInsight,
			names:   []aitool.ToolName{aitool.ToolNameDraftEmail},
			want:    []aitool.ToolName{aitool.ToolNameDraftEmail},
			wantNot: []aitool.ToolName{aitool.ToolNameSendEmail, aitool.ToolNameEmitInfoCard},
		},
		{
			name:   "insight all plus send_email",
			aiType: amai.TypeInsight,
			names:  []aitool.ToolName{aitool.ToolNameAll, aitool.ToolNameSendEmail},
			want:   []aitool.ToolName{aitool.ToolNameGetContactInteractions, aitool.ToolNameEmitInfoCard, aitool.ToolNameSendMessage, aitool.ToolNameDraftMessage, aitool.ToolNameSendEmail, aitool.ToolNameDraftEmail, aitool.ToolNameDraftCall, aitool.ToolNameCreateCall},
		},
		{
			name:    "normal explicit draft_email is dropped",
			aiType:  amai.TypeNormal,
			names:   []aitool.ToolName{aitool.ToolNameDraftEmail},
			wantNot: []aitool.ToolName{aitool.ToolNameDraftEmail},
		},
		{
			name:    "insight explicit send_message",
			aiType:  amai.TypeInsight,
			names:   []aitool.ToolName{aitool.ToolNameSendMessage},
			want:    []aitool.ToolName{aitool.ToolNameSendMessage},
			wantNot: []aitool.ToolName{aitool.ToolNameDraftMessage, aitool.ToolNameEmitInfoCard},
		},
		{
			name:    "insight explicit draft_message",
			aiType:  amai.TypeInsight,
			names:   []aitool.ToolName{aitool.ToolNameDraftMessage},
			want:    []aitool.ToolName{aitool.ToolNameDraftMessage},
			wantNot: []aitool.ToolName{aitool.ToolNameSendMessage, aitool.ToolNameEmitInfoCard},
		},
		{
			name:   "insight all plus send_message",
			aiType: amai.TypeInsight,
			names:  []aitool.ToolName{aitool.ToolNameAll, aitool.ToolNameSendMessage},
			want:   []aitool.ToolName{aitool.ToolNameGetContactInteractions, aitool.ToolNameEmitInfoCard, aitool.ToolNameSendMessage, aitool.ToolNameDraftMessage, aitool.ToolNameSendEmail, aitool.ToolNameDraftEmail, aitool.ToolNameDraftCall, aitool.ToolNameCreateCall},
		},
		{
			name:    "normal all has send_message, send_email and create_call but never a draft tool",
			aiType:  amai.TypeNormal,
			names:   []aitool.ToolName{aitool.ToolNameAll},
			want:    []aitool.ToolName{aitool.ToolNameSendMessage, aitool.ToolNameSendEmail, aitool.ToolNameConnectCall, aitool.ToolNameCreateCall},
			wantNot: []aitool.ToolName{aitool.ToolNameDraftMessage, aitool.ToolNameDraftEmail, aitool.ToolNameDraftCall, aitool.ToolNameEmitInfoCard},
		},
		{
			name:    "normal explicit draft_message is dropped",
			aiType:  amai.TypeNormal,
			names:   []aitool.ToolName{aitool.ToolNameDraftMessage},
			wantNot: []aitool.ToolName{aitool.ToolNameDraftMessage},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &toolHandler{tools: cached}
			got := h.GetByNames(tt.aiType, tt.names)

			set := map[aitool.ToolName]bool{}
			for _, g := range got {
				set[g.Name] = true
			}
			if len(got) != len(tt.want) {
				t.Errorf("GetByNames() returned %d tools, want %d", len(got), len(tt.want))
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

// Test_ToolHandler_GetByNames_InsightAllReturnsSixteenTools: with every tool in
// the cache, the "all" selector of an Insight AI returns exactly the 9
// AllInsightToolNames plus the 2 InsightMessageToolNames (VOIP-1708), the 2
// InsightEmailToolNames (VOIP-1710) and the 3 InsightCallToolNames (VOIP-1717),
// and no Normal-only tool. The 9 include search_knowledge (VOIP-1720), which
// AllToolNames also holds, so the cache is seeded with the de-duplicated union.
func Test_ToolHandler_GetByNames_InsightAllReturnsSixteenTools(t *testing.T) {
	cached := []aitool.Tool{}
	seeded := map[aitool.ToolName]bool{}
	for _, n := range append(append([]aitool.ToolName{}, aitool.AllToolNames...), aitool.AllInsightToolNames...) {
		if seeded[n] {
			continue
		}
		seeded[n] = true
		cached = append(cached, aitool.Tool{Name: n})
	}
	cached = append(cached, aitool.Tool{Name: aitool.ToolNameDraftMessage}, aitool.Tool{Name: aitool.ToolNameDraftEmail}, aitool.Tool{Name: aitool.ToolNameDraftCall})

	h := &toolHandler{tools: cached}
	got := h.GetByNames(amai.TypeInsight, []aitool.ToolName{aitool.ToolNameAll})

	want := map[aitool.ToolName]bool{}
	for _, n := range aitool.AllInsightToolNames {
		want[n] = true
	}
	for _, n := range aitool.InsightMessageToolNames {
		want[n] = true
	}
	for _, n := range aitool.InsightEmailToolNames {
		want[n] = true
	}
	for _, n := range aitool.InsightCallToolNames {
		want[n] = true
	}
	if len(want) != 16 {
		t.Fatalf("expected set has %d names, want 16", len(want))
	}

	seen := map[aitool.ToolName]bool{}
	for _, g := range got {
		if !want[g.Name] {
			t.Errorf("GetByNames() returned unexpected %s", g.Name)
		}
		if seen[g.Name] {
			t.Errorf("GetByNames() returned %s twice", g.Name)
		}
		seen[g.Name] = true
	}
	if len(seen) != 16 {
		t.Errorf("GetByNames() returned %d distinct tools, want 16", len(seen))
	}
}
