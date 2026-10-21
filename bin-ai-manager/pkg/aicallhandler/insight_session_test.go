package aicallhandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

// Test_writeInsightSessionMetadata_McpToolMap pins that a refreshed Insight
// session stores the tool map of the AI's whitelisted servers, and that a
// metadata key written while the (network bound) discovery runs is kept: the
// AIcall is re-read after discovery, not before it.
func Test_writeInsightSessionMetadata_McpToolMap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
	mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)

	aicallID := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	serverID := uuid.FromStringOrNil("aaaaaaaa-1111-4000-8000-000000000001")

	// A session that is listening to a live call refreshes its tool map like
	// any other Insight session.
	existing := &aicall.AIcall{
		Identity:      commonidentity.Identity{ID: aicallID, CustomerID: testCustomerID},
		ReferenceType: aicall.ReferenceTypeContactCase,
		ListenCallID:  uuid.Must(uuid.NewV4()),
	}
	a := &ai.AI{
		Identity:     commonidentity.Identity{ID: aiID, CustomerID: testCustomerID},
		Type:         ai.TypeInsight,
		McpServerIDs: []uuid.UUID{serverID},
	}

	mockSrv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		Identity: commonidentityFor(serverID),
		Status:   mcpserver.StatusActive,
	}, nil)

	discovered := false
	mockTool.EXPECT().ListTools(gomock.Any(), serverID).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) ([]mcptoolhandler.McpTool, error) {
			discovered = true
			return []mcptoolhandler.McpTool{{Name: "search_tickets", Description: "search"}}, nil
		},
	)

	// The row as it is stored when the re-read happens: a key written by
	// someone else during discovery is there only if the re-read came after.
	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*aicall.AIcall, error) {
			md := map[string]any{}
			if discovered {
				md["written_during_discovery"] = "kept"
			}
			return &aicall.AIcall{
				Identity: commonidentity.Identity{ID: aicallID, CustomerID: testCustomerID},
				Metadata: md,
			}, nil
		},
	)

	var written map[string]any
	mockDB.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), aicallID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
			written, _ = fields[aicall.FieldMetadata].(map[string]any)
			return nil
		},
	)

	h := &aicallHandler{db: mockDB, mcpServerHandler: mockSrv, mcptoolHandler: mockTool}

	if err := h.writeInsightSessionMetadata(context.Background(), existing, a, time.Now(), "prompt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	toolMap, _ := written[aicall.MetaKeyMcpToolMap].(map[string]aicall.McpToolRef)
	if _, ok := toolMap["mcp_aaaaaaaa_search_tickets"]; !ok || len(toolMap) != 1 {
		t.Errorf("expected the tool map to hold the whitelisted server's tool, got: %v", toolMap)
	}
	if written["written_during_discovery"] != "kept" {
		t.Errorf("expected the key written during discovery to be kept, got: %v", written)
	}
}
