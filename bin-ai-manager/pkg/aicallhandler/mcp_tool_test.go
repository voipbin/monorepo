package aicallhandler

import (
	"context"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

// commonidentityFor builds the minimal Identity embedded in an AI/McpServer
// test fixture; only ID matters for these tests.
func commonidentityFor(id uuid.UUID) commonidentity.Identity {
	return commonidentity.Identity{ID: id}
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
			name: "one server's ListTools failure does not affect the other server's tools",
			ai: &ai.AI{
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
			name:            "fail closed: unresolvable namespaced name (not in the metadata map at all)",
			aicall:          baseAIcall(goValueToolMap),
			toolName:        message.FunctionCallName("mcp_ffffffff_some_other_tool"),
			setupMock:       func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {},
			wantResult:      "failed",
			wantCallToolHit: false,
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
		},
		{
			name:     "fail closed: resolveAI failure produces a generic failure, not the raw AIHandler error",
			aicall:   baseAIcall(goValueToolMap),
			toolName: namespacedName,
			setupMock: func(aiH *aihandler.MockAIHandler, srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				aiH.EXPECT().Get(gomock.Any(), aiID).Return(nil, context.DeadlineExceeded)
			},
			wantResult:      "failed",
			wantCallToolHit: false,
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

			if tt.wantResult == "failed" {
				if strings.Contains(got.Message, "super-secret-upstream-value") {
					t.Errorf("the remote MCP server's raw error text must never reach the LLM-facing message, got: %q", got.Message)
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

// errorWithSecret simulates a misbehaving remote MCP server whose error
// carries a value that must never reach the LLM-facing message verbatim.
func errorWithSecret() error {
	return mcpToolCallError("upstream said: super-secret-upstream-value")
}
