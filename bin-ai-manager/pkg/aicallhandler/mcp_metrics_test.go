package aicallhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

// Test_labelForToolExecuteMetric pins B17/B18's cardinality fix: an
// mcp_-prefixed tool name is labeled by the bounded constant "mcp", never by
// the raw (customer-controlled) name; a built-in tool name passes through
// unchanged.
func Test_labelForToolExecuteMetric(t *testing.T) {
	tests := []struct {
		name string
		in   message.FunctionCallName
		want string
	}{
		{
			name: "built-in tool name passes through",
			in:   message.FunctionCallNameConnectCall,
			want: string(message.FunctionCallNameConnectCall),
		},
		{
			name: "mcp tool name collapses to the bounded constant",
			in:   message.FunctionCallName("mcp_aaaaaaaa_whatever_the_customer_named_it"),
			want: "mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := labelForToolExecuteMetric(tt.in); got != tt.want {
				t.Errorf("labelForToolExecuteMetric(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Test_ResolveMcpTools_AdvertisedMetric pins that a non-empty ResolveMcpTools
// resolution increments promMcpToolAdvertisedTotal exactly once, and an
// empty resolution (Insight AI, gated at B10) does not increment it at all
// (B17/B18).
func Test_ResolveMcpTools_AdvertisedMetric(t *testing.T) {
	enableMcpToolExposureForTest(t, true)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)
	mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
	mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)

	aicallID := uuid.Must(uuid.NewV4())
	assistanceID := uuid.Must(uuid.NewV4())
	serverID := uuid.FromStringOrNil("bbbbbbbb-1111-4000-8000-000000000002")

	c := &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID, CustomerID: testCustomerID},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   assistanceID,
		Metadata:       map[string]any{},
	}
	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(c, nil).Times(2)
	mockAI.EXPECT().Get(gomock.Any(), assistanceID).Return(&ai.AI{
		Identity:     commonidentity.Identity{ID: assistanceID, CustomerID: testCustomerID},
		Type:         ai.TypeNormal,
		McpServerIDs: []uuid.UUID{serverID},
	}, nil)
	mockSrv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		Identity: commonidentityFor(serverID),
		Status:   mcpserver.StatusActive,
	}, nil)
	mockTool.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
		{Name: "do_the_thing"},
	}, nil)
	mockDB.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), aicallID, gomock.Any()).Return(nil)

	h := &aicallHandler{db: mockDB, aiHandler: mockAI, mcpServerHandler: mockSrv, mcptoolHandler: mockTool}

	before := testutil.ToFloat64(promMcpToolAdvertisedTotal)
	got, err := h.ResolveMcpTools(context.Background(), aicallID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly one resolved tool, got: %v", got)
	}
	after := testutil.ToFloat64(promMcpToolAdvertisedTotal)
	if after != before+1 {
		t.Errorf("expected promMcpToolAdvertisedTotal to increment by 1, before=%v after=%v", before, after)
	}
}
