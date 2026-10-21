package aicallhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

// enableMcpToolExposureForTest sets the package-global config singleton's
// McpToolExposureEnabled directly (there is no LoadGlobalConfig call in unit
// tests, so the flag defaults to the Go zero value, false, unless a test sets
// it). It restores the previous value on cleanup so tests do not leak state
// into each other.
func enableMcpToolExposureForTest(t *testing.T, enabled bool) {
	t.Helper()
	prev := config.Get().McpToolExposureEnabled
	config.Get().McpToolExposureEnabled = enabled
	t.Cleanup(func() {
		config.Get().McpToolExposureEnabled = prev
	})
}

// Test_ResolveMcpTools_ConfigDisabled pins B27: the flag is the FIRST thing
// checked, before any DB read or RPC -- disabled must return (nil, nil)
// without ever touching the db or aiHandler mocks (zero EXPECT() calls set;
// gomock fails the test on any unexpected call).
func Test_ResolveMcpTools_ConfigDisabled(t *testing.T) {
	enableMcpToolExposureForTest(t, false)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	h := &aicallHandler{db: mockDB, aiHandler: mockAI}

	got, err := h.ResolveMcpTools(context.Background(), uuid.Must(uuid.NewV4()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil tools when disabled, got: %v", got)
	}
}

// Test_ResolveMcpTools_Team pins B11: a team AIcall returns (nil, nil)
// without ever resolving an AI (aiHandler.Get is never called; the mock has
// no EXPECT() for it).
func Test_ResolveMcpTools_Team(t *testing.T) {
	enableMcpToolExposureForTest(t, true)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	aicallID := uuid.Must(uuid.NewV4())
	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(&aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID},
		AssistanceType: aicall.AssistanceTypeTeam,
	}, nil)

	h := &aicallHandler{db: mockDB, aiHandler: mockAI}

	got, err := h.ResolveMcpTools(context.Background(), aicallID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil tools for a team aicall, got: %v", got)
	}
}

// Test_ResolveMcpTools_Insight pins that an Insight AI is handled like a
// Normal AI: the tools of the servers in its own mcp_server_ids are
// advertised and persisted, an empty list adds nothing, and a server of
// another customer is looked up but never listed. The aicall references a
// contact case with listen metadata, i.e. the shape a listen turn runs on.
func Test_ResolveMcpTools_Insight(t *testing.T) {
	serverID := uuid.FromStringOrNil("aaaaaaaa-1111-4000-8000-000000000001")

	tests := []struct {
		name string

		mcpServerIDs []uuid.UUID
		serverOwner  commonidentity.Identity

		expectServerGet bool
		expectListTools bool
		expectTools     []string
	}{
		{
			name:            "whitelisted server: its tools are advertised",
			mcpServerIDs:    []uuid.UUID{serverID},
			serverOwner:     commonidentityFor(serverID),
			expectServerGet: true,
			expectListTools: true,
			expectTools:     []string{"mcp_aaaaaaaa_search_tickets"},
		},
		{
			name:         "empty whitelist: nothing is advertised or looked up",
			mcpServerIDs: []uuid.UUID{},
			expectTools:  []string{},
		},
		{
			name:            "server of another customer: skipped without listing",
			mcpServerIDs:    []uuid.UUID{serverID},
			serverOwner:     otherCustomerIdentityFor(serverID),
			expectServerGet: true,
			expectTools:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enableMcpToolExposureForTest(t, true)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
			mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)

			aicallID := uuid.Must(uuid.NewV4())
			assistanceID := uuid.Must(uuid.NewV4())

			c := &aicall.AIcall{
				Identity:       commonidentity.Identity{ID: aicallID, CustomerID: testCustomerID},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   assistanceID,
				ReferenceType:  aicall.ReferenceTypeContactCase,
				ListenCallID:   uuid.Must(uuid.NewV4()),
				Metadata:       map[string]any{},
			}
			// h.Get, then persistToolMap's own re-read.
			mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(c, nil).Times(2)
			mockAI.EXPECT().Get(gomock.Any(), assistanceID).Return(&ai.AI{
				Identity:     commonidentity.Identity{ID: assistanceID, CustomerID: testCustomerID},
				Type:         ai.TypeInsight,
				McpServerIDs: tt.mcpServerIDs,
			}, nil)
			if tt.expectServerGet {
				mockSrv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: tt.serverOwner,
					Status:   mcpserver.StatusActive,
				}, nil)
			}
			if tt.expectListTools {
				mockTool.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
					{Name: "search_tickets", Description: "search"},
				}, nil)
			}

			var persistedMetadata map[string]any
			mockDB.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), aicallID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
					persistedMetadata, _ = fields[aicall.FieldMetadata].(map[string]any)
					return nil
				},
			)

			h := &aicallHandler{db: mockDB, aiHandler: mockAI, mcpServerHandler: mockSrv, mcptoolHandler: mockTool}

			got, err := h.ResolveMcpTools(context.Background(), aicallID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotNames := []string{}
			for _, gt := range got {
				gotNames = append(gotNames, string(gt.Name))
			}
			if !reflect.DeepEqual(gotNames, tt.expectTools) {
				t.Errorf("tool names mismatch. expected: %v, got: %v", tt.expectTools, gotNames)
			}

			toolMap, _ := persistedMetadata[aicall.MetaKeyMcpToolMap].(map[string]aicall.McpToolRef)
			if len(toolMap) != len(tt.expectTools) {
				t.Errorf("persisted tool map size mismatch. expected: %d, got: %d", len(tt.expectTools), len(toolMap))
			}
		})
	}
}

// Test_ResolveMcpTools_Normal_MCPOnly pins the core B12 contract (design
// §2.3's round-1 correction, explicitly called out as the test that would
// have caught the duplicate-built-ins bug): a Normal AI's ResolveMcpTools
// result contains ONLY MCP-derived tools, never a built-in tool name, even
// though a toolNameResolver is wired on the handler. It also asserts the
// resolved tool map is persisted (D15) via AIcallUpdateNoTouchTMUpdate.
func Test_ResolveMcpTools_Normal_MCPOnly(t *testing.T) {
	enableMcpToolExposureForTest(t, true)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)
	mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
	mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)

	aicallID := uuid.Must(uuid.NewV4())
	assistanceID := uuid.Must(uuid.NewV4())
	serverID := uuid.FromStringOrNil("aaaaaaaa-1111-4000-8000-000000000001")

	c := &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID, CustomerID: testCustomerID},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   assistanceID,
		Metadata:       map[string]any{},
	}

	// h.Get (AIcallHandler.Get) reads via h.db.AIcallGet; persistToolMap then
	// re-reads the SAME row again before merging (its own
	// read-modify-write-by-key discipline) -- both calls hit AIcallGet.
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
		{Name: "search_tickets", Description: "search"},
	}, nil)

	var persistedMetadata map[string]any
	mockDB.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), aicallID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
			persistedMetadata, _ = fields[aicall.FieldMetadata].(map[string]any)
			return nil
		},
	)

	h := &aicallHandler{db: mockDB, aiHandler: mockAI, mcpServerHandler: mockSrv, mcptoolHandler: mockTool}

	got, err := h.ResolveMcpTools(context.Background(), aicallID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || string(got[0].Name) != "mcp_aaaaaaaa_search_tickets" {
		t.Fatalf("expected exactly one mcp-namespaced tool, got: %v", got)
	}
	for _, gt := range got {
		if len(string(gt.Name)) < len(mcpToolNamePrefix) || string(gt.Name)[:len(mcpToolNamePrefix)] != mcpToolNamePrefix {
			t.Errorf("ResolveMcpTools returned a non-mcp-namespaced (built-in) tool: %s", gt.Name)
		}
	}

	if persistedMetadata == nil {
		t.Fatalf("expected the tool map to be persisted via AIcallUpdateNoTouchTMUpdate")
	}
	if _, ok := persistedMetadata[aicall.MetaKeyMcpToolMap]; !ok {
		t.Errorf("expected persisted metadata to contain MetaKeyMcpToolMap, got: %v", persistedMetadata)
	}
}

// Test_ResolveMcpTools_AIResolutionFailure pins that an AI-resolution
// failure propagates as an error (the caller's fail-open-to-built-ins policy
// is exercised at the runner.go level, not here -- design §2.1/§11).
func Test_ResolveMcpTools_AIResolutionFailure(t *testing.T) {
	enableMcpToolExposureForTest(t, true)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	aicallID := uuid.Must(uuid.NewV4())
	assistanceID := uuid.Must(uuid.NewV4())
	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(&aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   assistanceID,
	}, nil)
	mockAI.EXPECT().Get(gomock.Any(), assistanceID).Return(nil, context.DeadlineExceeded)

	h := &aicallHandler{db: mockDB, aiHandler: mockAI}

	_, err := h.ResolveMcpTools(context.Background(), aicallID)
	if err == nil {
		t.Fatalf("expected an error when AI resolution fails")
	}
}
