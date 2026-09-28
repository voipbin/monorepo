package aicallhandler

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
	"monorepo/bin-ai-manager/pkg/teamhandler"
)

// commonidentityFor builds the minimal Identity embedded in an AI/McpServer
// test fixture; only ID matters for these tests.
// testCustomerID is the owner shared by the AI and its MCP servers in these
// tests. It is explicit rather than the zero UUID so an ownership gate cannot
// pass by two zero values happening to match.
var testCustomerID = uuid.FromStringOrNil("c0000000-1111-4000-8000-00000000000c")

func commonidentityFor(id uuid.UUID) commonidentity.Identity {
	return commonidentity.Identity{ID: id, CustomerID: testCustomerID}
}

// otherCustomerIdentityFor builds a server identity owned by a DIFFERENT
// customer, for the ownership-gate cases.
func otherCustomerIdentityFor(id uuid.UUID) commonidentity.Identity {
	return commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("d0000000-1111-4000-8000-00000000000d")}
}

// Test_resolveTools covers the fail-closed/best-effort properties resolveTools
// promises (design §9.1): a non-active server contributes no tools, a
// ListTools failure on one server does not fail the whole resolution or
// affect other servers, and the returned tool map is namespaced correctly.
func Test_resolveTools(t *testing.T) {
	tests := []struct {
		name string

		ai *ai.AI

		setupMock func(*mcpserverhandler.MockMcpServerHandler, *mcptoolhandler.MockMcpToolHandler)

		expectToolNames []string
		expectToolMap   map[string]aicall.McpToolRef
	}{
		{
			name: "active server contributes namespaced tools",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("aaaaaaaa-1111-4000-8000-000000000001")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("aaaaaaaa-1111-4000-8000-000000000001")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
					{Name: "search_tickets", Description: "search"},
				}, nil)
			},
			expectToolNames: []string{"mcp_aaaaaaaa_search_tickets"},
			expectToolMap: map[string]aicall.McpToolRef{
				"mcp_aaaaaaaa_search_tickets": {
					ServerID: uuid.FromStringOrNil("aaaaaaaa-1111-4000-8000-000000000001"),
					ToolName: "search_tickets",
				},
			},
		},
		{
			name: "disabled server contributes no tools and does not call ListTools",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("bbbbbbbb-1111-4000-8000-000000000002")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("bbbbbbbb-1111-4000-8000-000000000002")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusDisabled,
				}, nil)
				// ListTools must NOT be called for a disabled server.
			},
			expectToolNames: []string{},
			expectToolMap:   map[string]aicall.McpToolRef{},
		},
		{
			// Deleted is distinct from disabled: McpServerGet returns
			// soft-deleted rows on purpose (the REST read of a deleted server
			// answers 200), so Status alone stays Active and the row would
			// still contribute tools without an explicit tm_delete check.
			name: "soft-deleted server contributes no tools and does not call ListTools",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("eeeeeeee-1111-4000-8000-00000000000e")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("eeeeeeee-1111-4000-8000-00000000000e")
				ts := time.Now()
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
					TMDelete: &ts,
				}, nil)
				// ListTools must NOT be called for a deleted server.
			},
			expectToolNames: []string{},
			expectToolMap:   map[string]aicall.McpToolRef{},
		},
		{
			// A stored id outlives the validation that admitted it: the server
			// can be reassigned, or the whitelist can predate a tightening.
			// Resolution must not hand another customer's tools to this LLM.
			name: "server owned by another customer contributes no tools",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("ffffffff-1111-4000-8000-00000000000f")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("ffffffff-1111-4000-8000-00000000000f")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: otherCustomerIdentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				// ListTools must NOT be called for a foreign server.
			},
			expectToolNames: []string{},
			expectToolMap:   map[string]aicall.McpToolRef{},
		},
		{
			name: "one server's ListTools failure does not affect the other server's tools",
			ai: &ai.AI{
				Identity: commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{
					uuid.FromStringOrNil("cccccccc-1111-4000-8000-000000000003"),
					uuid.FromStringOrNil("dddddddd-1111-4000-8000-000000000004"),
				},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				failingID := uuid.FromStringOrNil("cccccccc-1111-4000-8000-000000000003")
				okID := uuid.FromStringOrNil("dddddddd-1111-4000-8000-000000000004")

				srv.EXPECT().Get(gomock.Any(), failingID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(failingID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().ListTools(gomock.Any(), failingID).Return(nil, context.DeadlineExceeded)

				srv.EXPECT().Get(gomock.Any(), okID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(okID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().ListTools(gomock.Any(), okID).Return([]mcptoolhandler.McpTool{
					{Name: "create_ticket"},
				}, nil)
			},
			expectToolNames: []string{"mcp_dddddddd_create_ticket"},
			expectToolMap: map[string]aicall.McpToolRef{
				"mcp_dddddddd_create_ticket": {
					ServerID: uuid.FromStringOrNil("dddddddd-1111-4000-8000-000000000004"),
					ToolName: "create_ticket",
				},
			},
		},
		{
			name: "McpServer.Get failure for one server does not affect others",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("eeeeeeee-1111-4000-8000-000000000005")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("eeeeeeee-1111-4000-8000-000000000005")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(nil, context.DeadlineExceeded)
				// ListTools must NOT be called when Get fails.
			},
			expectToolNames: []string{},
			expectToolMap:   map[string]aicall.McpToolRef{},
		},
		{
			name: "no McpServerIDs resolves to empty tool map without touching the handlers",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: nil,
			},
			setupMock:       func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {},
			expectToolNames: []string{},
			expectToolMap:   map[string]aicall.McpToolRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
			mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)
			tt.setupMock(mockSrv, mockTool)

			h := &aicallHandler{
				mcpServerHandler: mockSrv,
				mcptoolHandler:   mockTool,
			}

			mergedTools, toolMap, err := h.resolveTools(context.Background(), tt.ai)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotNames := map[string]bool{}
			for _, mt := range mergedTools {
				gotNames[string(mt.Name)] = true
			}
			for _, want := range tt.expectToolNames {
				if !gotNames[want] {
					t.Errorf("expected merged tool list to contain %q, got tools: %v", want, mergedTools)
				}
			}
			if len(mergedTools) != len(tt.expectToolNames) {
				t.Errorf("expected %d merged tools, got %d: %v", len(tt.expectToolNames), len(mergedTools), mergedTools)
			}

			if len(toolMap) != len(tt.expectToolMap) {
				t.Errorf("expected tool map %v, got %v", tt.expectToolMap, toolMap)
			}
			for k, v := range tt.expectToolMap {
				if toolMap[k] != v {
					t.Errorf("tool map key %q: expected %v, got %v", k, v, toolMap[k])
				}
			}
		})
	}
}

// Test_resolveTools_NilHandlers pins that resolveTools degrades gracefully
// (returns only built-ins, never panics) when mcpServerHandler/mcptoolHandler
// are nil -- the state every test-constructed aicallHandler that doesn't
// explicitly wire MCP support is in, and the state cmd/ai-control's minimal
// AIcallHandler construction is in today.
func Test_resolveTools_NilHandlers(t *testing.T) {
	h := &aicallHandler{}

	mergedTools, toolMap, err := h.resolveTools(context.Background(), &ai.AI{
		McpServerIDs: []uuid.UUID{uuid.Must(uuid.NewV4())},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mergedTools) != 0 {
		t.Errorf("expected no tools with nil handlers, got: %v", mergedTools)
	}
	if len(toolMap) != 0 {
		t.Errorf("expected empty tool map with nil handlers, got: %v", toolMap)
	}
}

// Test_toolHandleMcpCall covers every fail-closed branch design §9.2 calls
// out by name: unresolvable namespaced tool name, a server dropped from the
// AI's whitelist since the tool list was resolved (stale reference after a
// mid-session config change), a server flipped to disabled since resolution,
// and a downstream CallTool failure. Each must produce a generic failure
// message, never forward the remote server's raw error text unbounded, and
// never call CallTool once any earlier check has failed.
func Test_toolHandleMcpCall(t *testing.T) {
	aiID := uuid.Must(uuid.NewV4())
	serverID := uuid.Must(uuid.NewV4())
	otherServerID := uuid.Must(uuid.NewV4())
	namespacedName := message.FunctionCallName("mcp_" + mcpServerIDShort(serverID) + "_search_tickets")

	baseAIcall := func(toolMapValue any) *aicall.AIcall {
		return &aicall.AIcall{
			AssistanceType: aicall.AssistanceTypeAI,
			AssistanceID:   aiID,
			Metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: toolMapValue,
			},
		}
	}

	goValueToolMap := map[string]aicall.McpToolRef{
		string(namespacedName): {ServerID: serverID, ToolName: "search_tickets"},
	}
	// jsonRoundTrippedToolMap simulates aicall.Metadata after a JSON
	// marshal/unmarshal cycle: McpToolRef's json tags (server_id/tool_name)
	// surface as a map[string]any, not the Go struct value.
	jsonRoundTrippedToolMap := map[string]any{
		string(namespacedName): map[string]any{
			"server_id": serverID.String(),
			"tool_name": "search_tickets",
		},
	}

	tests := []struct {
		name string

		aicall    *aicall.AIcall
		toolName  message.FunctionCallName
		setupMock func(*aihandler.MockAIHandler, *mcpserverhandler.MockMcpServerHandler, *mcptoolhandler.MockMcpToolHandler)

		wantResult      string
		wantCallToolHit bool
		// Which refusal. Every gate fills the same "failed" result, so
		// asserting only on that lets a nil AI be replaced by an empty one and
		// still pass: the NEXT gate refuses and the outcome is identical.
		wantMessage string
	}{
		{
			name:     "success: Go-value metadata resolves and dispatches",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().CallTool(gomock.Any(), serverID, "search_tickets", gomock.Any()).Return("3 tickets found", nil)
			},
			wantResult:      "success",
			wantCallToolHit: true,
		},
		{
			name:     "success: JSON-round-tripped metadata resolves and dispatches",
			aicall:   baseAIcall(jsonRoundTrippedToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().CallTool(gomock.Any(), serverID, "search_tickets", gomock.Any()).Return("ok", nil)
			},
			wantResult:      "success",
			wantCallToolHit: true,
		},
		{
			name:     "fail closed: unresolvable namespaced name (not in the metadata map at all)",
			aicall:   baseAIcall(goValueToolMap),
			toolName: message.FunctionCallName("mcp_ffffffff_some_other_tool"),
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			wantMessage:     "unknown mcp tool call",
		},
		{
			// The dispatch gate is not redundant with the resolution gate: an
			// AIcall can be reused for hours after its tool map was built, so
			// the server may be deleted between resolution and this call.
			name:     "fail closed: server soft-deleted since resolution",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				ts := time.Now()
				// Status is still Active: only tm_delete marks it gone, and
				// McpServerGet returns deleted rows on purpose.
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
					TMDelete: &ts,
				}, nil)
				// CallTool must NOT be reached.
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			wantMessage:     "mcp tool is no longer available",
		},
		{
			// Being on the whitelist proves the AI still lists the server; it
			// does not prove the server still belongs to the AI's customer.
			name:     "fail closed: whitelisted server owned by another customer",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: otherCustomerIdentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				// CallTool must NOT be reached.
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			wantMessage:     "mcp tool is no longer available",
		},
		{
			// Defensive, not reachable through the current mcpServerHandler
			// (its Get returns a non-nil row whenever err is nil). The gate
			// keeps it because that is an implementation detail of one
			// implementation, not a guarantee of the interface, and a nil
			// dereference here would panic inside an LLM tool dispatch.
			name:     "fail closed: resolver returns a nil server without an error",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				srv.EXPECT().Get(gomock.Any(), serverID).Return(nil, nil)
				// CallTool must NOT be reached, and this must not panic.
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			wantMessage:     "mcp tool is no longer available",
		},
		{
			name:     "fail closed: server dropped from the AI's whitelist since resolution (stale reference)",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity: commonidentityFor(aiID),
					// serverID is no longer in the whitelist -- only otherServerID is.
					McpServerIDs: []uuid.UUID{otherServerID},
				}, nil)
				// McpServer.Get and CallTool must NOT be called once the whitelist check fails.
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			wantMessage:     "mcp tool is no longer available",
		},
		{
			name:     "fail closed: server flipped to disabled since resolution",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusDisabled,
				}, nil)
				// CallTool must NOT be called once the server is no longer active.
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			wantMessage:     "mcp tool is no longer available",
		},
		{
			name:     "fail closed: downstream CallTool failure never leaks the raw error to the LLM",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(&ai.AI{
					Identity:     commonidentityFor(aiID),
					McpServerIDs: []uuid.UUID{serverID},
				}, nil)
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().CallTool(gomock.Any(), serverID, "search_tickets", gomock.Any()).
					Return("", errorWithSecret())
			},
			wantResult:      "failed",
			wantCallToolHit: true,
			wantMessage:     "MCP tool call failed",
		},
		{
			name:     "fail closed: resolveAI failure produces a generic failure, not the raw AIHandler error",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				// Exactly once. resolveActiveAIForMcp subsumes resolveAI, so a nil
				// result fails closed directly instead of re-running the same fetch.
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(nil, context.DeadlineExceeded)
			},
			wantResult:      "failed",
			wantCallToolHit: false,
			// The resolver's OWN refusal, not the whitelist gate's. An empty
			// AI substituted for the nil would produce the whitelist
			// message instead, and that mutation used to survive.
			wantMessage: "could not retrieve AI configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAI := aihandler.NewMockAIHandler(mc)
			mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
			mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)
			tt.setupMock(mockAI, mockSrv, mockTool)

			h := &aicallHandler{
				aiHandler:        mockAI,
				mcpServerHandler: mockSrv,
				mcptoolHandler:   mockTool,
			}

			tc := &message.ToolCall{
				ID: "tool-1",
				Function: message.FunctionCall{
					Name:      tt.toolName,
					Arguments: `{}`,
				},
			}

			got := h.toolHandleMcpCall(context.Background(), tt.aicall, tc)
			if got.Result != tt.wantResult {
				t.Errorf("expected result %q, got %q (message: %q)", tt.wantResult, got.Result, got.Message)
			}

			if tt.wantMessage != "" && got.Message != tt.wantMessage {
				t.Errorf("wrong refusal. expect: %q, got: %q", tt.wantMessage, got.Message)
			}

			if tt.wantResult == "failed" {
				if strings.Contains(got.Message, "super-secret-upstream-value") {
					t.Errorf("the remote MCP server's raw error text must never reach the LLM-facing message, got: %q", got.Message)
				}
				// No refusal message may carry an internal identifier. A
				// UUID-shaped substring can only have come from a server/ai/aicall
				// id, and no legitimate generic refusal contains one, so this is a
				// leak regardless of which gate produced it.
				if uuidShapedRe.MatchString(got.Message) {
					t.Errorf("refusal message must not contain a UUID-shaped identifier, got: %q", got.Message)
				}
				// mcpServerIsUsable's internal `why` strings describe the
				// customer's configuration to the LLM. Only the unambiguous ones
				// are listed: "not found"/"deleted" are ordinary English a future
				// generic message could legitimately use.
				for _, internal := range []string{"owned by another customer", "not active:"} {
					if strings.Contains(got.Message, internal) {
						t.Errorf("refusal message must not contain the internal reason %q, got: %q", internal, got.Message)
					}
				}
			}
		})
	}
}

// Test_lookupMcpToolRef_decodeMcpToolRef pins the two Metadata shapes
// lookupMcpToolRef/decodeMcpToolRef must handle: the in-process Go value
// written by resolveTools' callers, and the map[string]any shape Metadata
// arrives in after a JSON round trip (e.g. read back from the DB). Malformed
// entries of the JSON shape must miss (fail closed), never panic.
func Test_lookupMcpToolRef_decodeMcpToolRef(t *testing.T) {
	serverID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name     string
		metadata map[string]any
		lookup   string

		wantOK  bool
		wantRef aicall.McpToolRef
	}{
		{
			name: "Go value map hit",
			metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: map[string]aicall.McpToolRef{
					"mcp_aaaaaaaa_search": {ServerID: serverID, ToolName: "search"},
				},
			},
			lookup:  "mcp_aaaaaaaa_search",
			wantOK:  true,
			wantRef: aicall.McpToolRef{ServerID: serverID, ToolName: "search"},
		},
		{
			name: "JSON round-tripped map hit",
			metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: map[string]any{
					"mcp_aaaaaaaa_search": map[string]any{
						"server_id": serverID.String(),
						"tool_name": "search",
					},
				},
			},
			lookup:  "mcp_aaaaaaaa_search",
			wantOK:  true,
			wantRef: aicall.McpToolRef{ServerID: serverID, ToolName: "search"},
		},
		{
			name: "key not present in the map",
			metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: map[string]any{},
			},
			lookup: "mcp_aaaaaaaa_search",
			wantOK: false,
		},
		{
			name:     "MetaKeyMcpToolMap entirely absent from Metadata",
			metadata: map[string]any{},
			lookup:   "mcp_aaaaaaaa_search",
			wantOK:   false,
		},
		{
			name: "malformed JSON entry: server_id not a valid UUID",
			metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: map[string]any{
					"mcp_aaaaaaaa_search": map[string]any{
						"server_id": "not-a-uuid",
						"tool_name": "search",
					},
				},
			},
			lookup: "mcp_aaaaaaaa_search",
			wantOK: false,
		},
		{
			name: "malformed JSON entry: tool_name missing",
			metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: map[string]any{
					"mcp_aaaaaaaa_search": map[string]any{
						"server_id": serverID.String(),
					},
				},
			},
			lookup: "mcp_aaaaaaaa_search",
			wantOK: false,
		},
		{
			name: "MetaKeyMcpToolMap has an unexpected top-level type",
			metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: "not a map at all",
			},
			lookup: "mcp_aaaaaaaa_search",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &aicall.AIcall{Metadata: tt.metadata}
			ref, ok := lookupMcpToolRef(c, tt.lookup)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got ok=%v (ref=%v)", tt.wantOK, ok, ref)
			}
			if tt.wantOK && ref != tt.wantRef {
				t.Errorf("expected ref %v, got %v", tt.wantRef, ref)
			}
		})
	}
}

// Test_mcpServerIDIsWhitelisted covers the whitelist membership check used
// by toolHandleMcpCall's stale-reference fail-closed guard.
func Test_mcpServerIDIsWhitelisted(t *testing.T) {
	a := uuid.Must(uuid.NewV4())
	b := uuid.Must(uuid.NewV4())
	c := uuid.Must(uuid.NewV4())

	if !mcpServerIDIsWhitelisted([]uuid.UUID{a, b}, a) {
		t.Errorf("expected a to be whitelisted")
	}
	if mcpServerIDIsWhitelisted([]uuid.UUID{a, b}, c) {
		t.Errorf("expected c to NOT be whitelisted")
	}
	if mcpServerIDIsWhitelisted(nil, a) {
		t.Errorf("expected a nil whitelist to reject everything")
	}
}

// uuidShapedRe matches any RFC-4122-shaped identifier. An LLM-facing refusal
// message has no legitimate reason to contain one: every internal id (MCP
// server, AI, aicall) is UUID-shaped, so a match is a leak. It cannot produce
// a false positive on the generic refusals this package emits, none of which
// embed any identifier.
var uuidShapedRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// errorWithSecret simulates a misbehaving remote MCP server whose error
// carries a value that must never reach the LLM-facing message verbatim.
func errorWithSecret() error {
	return mcpToolCallError("upstream said: super-secret-upstream-value")
}

// Test_toolHandleMcpCall_team covers dispatch for a TEAM aicall, which the
// AI-typed table above cannot reach. This is defect D28: before the fix the
// dispatch gate resolved the START member, so after a mid-conversation member
// switch a tool call was authorised against the wrong member's whitelist.
//
// It also pins the fail-closed property for an assistance type that is neither
// AI nor team: such an aicall must be refused, never resolved by treating
// AssistanceID as an AI id.
func Test_toolHandleMcpCall_team(t *testing.T) {
	teamID := uuid.Must(uuid.NewV4())
	curMemberID := uuid.Must(uuid.NewV4())
	curAIID := uuid.Must(uuid.NewV4())
	startMemberID := uuid.Must(uuid.NewV4())
	startAIID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	// the server is whitelisted by the CURRENT member only
	curOnlyServerID := uuid.Must(uuid.NewV4())
	// and this one by the START member only
	startOnlyServerID := uuid.Must(uuid.NewV4())

	teamFixture := &team.Team{
		Identity:      commonidentityFor(teamID),
		StartMemberID: startMemberID,
		Members: []team.Member{
			{ID: curMemberID, AIID: curAIID},
			{ID: startMemberID, AIID: startAIID},
		},
	}
	curAI := &ai.AI{
		Identity:     commonidentity.Identity{ID: curAIID, CustomerID: customerID},
		McpServerIDs: []uuid.UUID{curOnlyServerID},
	}
	startAI := &ai.AI{
		Identity:     commonidentity.Identity{ID: startAIID, CustomerID: customerID},
		McpServerIDs: []uuid.UUID{startOnlyServerID},
	}

	aicallFor := func(assistanceType aicall.AssistanceType, assistanceID uuid.UUID, serverID uuid.UUID, toolName string) *aicall.AIcall {
		return &aicall.AIcall{
			AssistanceType:  assistanceType,
			AssistanceID:    assistanceID,
			CurrentMemberID: curMemberID,
			Metadata: map[string]any{
				aicall.MetaKeyMcpToolMap: map[string]aicall.McpToolRef{
					"mcp_" + mcpServerIDShort(serverID) + "_" + toolName: {ServerID: serverID, ToolName: toolName},
				},
			},
		}
	}

	tests := []struct {
		name       string
		aicall     *aicall.AIcall
		toolName   message.FunctionCallName
		setupMock  func(th *teamhandler.MockTeamHandler, aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler)
		wantResult string
	}{
		{
			name:     "team: a server the CURRENT member whitelists is dispatched",
			aicall:   aicallFor(aicall.AssistanceTypeTeam, teamID, curOnlyServerID, "search_tickets"),
			toolName: message.FunctionCallName("mcp_" + mcpServerIDShort(curOnlyServerID) + "_search_tickets"),
			setupMock: func(th *teamhandler.MockTeamHandler, aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(teamFixture, nil)
				aiH.EXPECT().Get(gomock.Any(), curAIID).Return(curAI, nil)
				srv.EXPECT().Get(gomock.Any(), curOnlyServerID).Return(&mcpserver.McpServer{
					Identity: commonidentity.Identity{ID: curOnlyServerID, CustomerID: customerID},
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().CallTool(gomock.Any(), curOnlyServerID, "search_tickets", gomock.Any()).Return("ok", nil)
			},
			wantResult: "success",
		},
		{
			name:     "team: a server only the START member whitelists is refused after a member switch",
			aicall:   aicallFor(aicall.AssistanceTypeTeam, teamID, startOnlyServerID, "search_tickets"),
			toolName: message.FunctionCallName("mcp_" + mcpServerIDShort(startOnlyServerID) + "_search_tickets"),
			setupMock: func(th *teamhandler.MockTeamHandler, aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(teamFixture, nil)
				aiH.EXPECT().Get(gomock.Any(), curAIID).Return(curAI, nil)
				// no CallTool: the whitelist gate must refuse before dispatch. Before
				// the D28 fix the start member resolved here and this call succeeded.
			},
			wantResult: "failed",
		},
		{
			name:     "team, degraded: current member AI unfetchable falls back to the START member and still dispatches",
			aicall:   aicallFor(aicall.AssistanceTypeTeam, teamID, startOnlyServerID, "search_tickets"),
			toolName: message.FunctionCallName("mcp_" + mcpServerIDShort(startOnlyServerID) + "_search_tickets"),
			setupMock: func(th *teamhandler.MockTeamHandler, aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(teamFixture, nil)
				aiH.EXPECT().Get(gomock.Any(), curAIID).Return(nil, context.DeadlineExceeded)
				// the fallback must keep dispatch working, not fail the call
				aiH.EXPECT().Get(gomock.Any(), startAIID).Return(startAI, nil)
				srv.EXPECT().Get(gomock.Any(), startOnlyServerID).Return(&mcpserver.McpServer{
					Identity: commonidentity.Identity{ID: startOnlyServerID, CustomerID: customerID},
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().CallTool(gomock.Any(), startOnlyServerID, "search_tickets", gomock.Any()).Return("ok", nil)
			},
			wantResult: "success",
		},
		{
			name:     "team, degraded: team unfetchable refuses without dispatching",
			aicall:   aicallFor(aicall.AssistanceTypeTeam, teamID, curOnlyServerID, "search_tickets"),
			toolName: message.FunctionCallName("mcp_" + mcpServerIDShort(curOnlyServerID) + "_search_tickets"),
			setupMock: func(th *teamhandler.MockTeamHandler, aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				// Exactly once: there is no second resolution to fall back to.
				th.EXPECT().Get(gomock.Any(), teamID).Return(nil, context.DeadlineExceeded)
			},
			wantResult: "failed",
		},
		{
			name:     "unsupported assistance type is refused, not resolved as an AI",
			aicall:   aicallFor("unknown", curAIID, curOnlyServerID, "search_tickets"),
			toolName: message.FunctionCallName("mcp_" + mcpServerIDShort(curOnlyServerID) + "_search_tickets"),
			setupMock: func(th *teamhandler.MockTeamHandler, aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				// no aiHandler.Get at all: treating AssistanceID as an AI id would
				// authorise the call against whatever row it happens to hit.
			},
			wantResult: "failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockTeam := teamhandler.NewMockTeamHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockSrv := mcpserverhandler.NewMockMcpServerHandler(mc)
			mockTool := mcptoolhandler.NewMockMcpToolHandler(mc)
			tt.setupMock(mockTeam, mockAI, mockSrv, mockTool)

			h := &aicallHandler{
				teamHandler:      mockTeam,
				aiHandler:        mockAI,
				mcpServerHandler: mockSrv,
				mcptoolHandler:   mockTool,
			}

			tc := &message.ToolCall{
				ID:       "tool-1",
				Function: message.FunctionCall{Name: tt.toolName, Arguments: `{}`},
			}

			got := h.toolHandleMcpCall(context.Background(), tt.aicall, tc)
			if got.Result != tt.wantResult {
				t.Errorf("Wrong match. expect: %s, got: %s (message: %s)", tt.wantResult, got.Result, got.Message)
			}
		})
	}
}
