package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/mcpschema"
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

// Test_resolveMcpOnly covers the fail-closed/best-effort properties resolveMcpOnly
// promises (design §9.1): a non-active server contributes no tools, a
// ListTools failure on one server does not fail the whole resolution or
// affect other servers, and the returned tool map is namespaced correctly.
func Test_resolveMcpOnly(t *testing.T) {
	tests := []struct {
		name string

		ai *ai.AI

		setupMock func(*mcpserverhandler.MockMcpServerHandler, *mcptoolhandler.MockMcpToolHandler)

		expectToolNames []string
		expectToolMap   map[string]aicall.McpToolRef
		// expectParams, when set, pins each named tool's advertised
		// Parameters (JSON; "null" for none) after normalization.
		expectParams map[string]string
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
			name: "advertised parameters are the normalized schema, not the raw one",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("a1a1a1a1-1111-4000-8000-000000000011")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("a1a1a1a1-1111-4000-8000-000000000011")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
					{Name: "get_issue", InputSchema: json.RawMessage(`{"$schema":"x","type":"object","additionalProperties":false,
						"properties":{"owner":{"type":"string","x-mcp-header":"owner"},"value":{"type":["string","number"]},
						"any":{"description":"any JSON value"}},"required":["owner"]}`)},
					{Name: "no_schema"},
				}, nil)
			},
			expectToolNames: []string{"mcp_a1a1a1a1_get_issue", "mcp_a1a1a1a1_no_schema"},
			expectToolMap: map[string]aicall.McpToolRef{
				"mcp_a1a1a1a1_get_issue": {ServerID: uuid.FromStringOrNil("a1a1a1a1-1111-4000-8000-000000000011"), ToolName: "get_issue"},
				"mcp_a1a1a1a1_no_schema": {ServerID: uuid.FromStringOrNil("a1a1a1a1-1111-4000-8000-000000000011"), ToolName: "no_schema"},
			},
			expectParams: map[string]string{
				"mcp_a1a1a1a1_get_issue": `{"type":"object","properties":{"owner":{"type":"string"},
					"value":{"anyOf":[{"type":"string"},{"type":"number"}]}},"required":["owner"]}`,
				"mcp_a1a1a1a1_no_schema": "null",
			},
		},
		{
			name: "a tool dropped by normalization is neither advertised nor dispatchable, the server's other tools are kept",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("b2b2b2b2-1111-4000-8000-000000000012")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("b2b2b2b2-1111-4000-8000-000000000012")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
					{Name: "bad", InputSchema: json.RawMessage(`{"type":"object","properties":{"v":{"description":"any"}},"required":["v"]}`)},
					{Name: "good", InputSchema: json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`)},
				}, nil)
			},
			expectToolNames: []string{"mcp_b2b2b2b2_good"},
			expectToolMap: map[string]aicall.McpToolRef{
				"mcp_b2b2b2b2_good": {ServerID: uuid.FromStringOrNil("b2b2b2b2-1111-4000-8000-000000000012"), ToolName: "good"},
			},
			expectParams: map[string]string{
				"mcp_b2b2b2b2_good": `{"type":"object","properties":{"q":{"type":"string"}}}`,
			},
		},
		{
			name: "a no-argument tool with a root {} schema is kept as an empty object",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("c3c3c3c3-1111-4000-8000-000000000013")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("c3c3c3c3-1111-4000-8000-000000000013")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tl.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
					{Name: "get_me", InputSchema: json.RawMessage(`{}`)},
				}, nil)
			},
			expectToolNames: []string{"mcp_c3c3c3c3_get_me"},
			expectToolMap: map[string]aicall.McpToolRef{
				"mcp_c3c3c3c3_get_me": {ServerID: uuid.FromStringOrNil("c3c3c3c3-1111-4000-8000-000000000013"), ToolName: "get_me"},
			},
			expectParams: map[string]string{
				"mcp_c3c3c3c3_get_me": `{"type":"object","properties":{}}`,
			},
		},
		{
			// Five $ref fan-out tools of about 35 KiB raw each fit the 256
			// KiB raw schemaBudget together, but each charges about 60 KiB of
			// output, so only four fit the 256 KiB outBudget. The raw and
			// charge sizes are asserted in Test_mcpRefFanOutSchema_Sizes.
			name: "the per-resolution output budget skips tools the raw budget would admit",
			ai: &ai.AI{
				Identity:     commonidentity.Identity{CustomerID: testCustomerID},
				McpServerIDs: []uuid.UUID{uuid.FromStringOrNil("d4d4d4d4-1111-4000-8000-000000000014")},
			},
			setupMock: func(srv *mcpserverhandler.MockMcpServerHandler, tl *mcptoolhandler.MockMcpToolHandler) {
				serverID := uuid.FromStringOrNil("d4d4d4d4-1111-4000-8000-000000000014")
				srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
					Identity: commonidentityFor(serverID),
					Status:   mcpserver.StatusActive,
				}, nil)
				tools := []mcptoolhandler.McpTool{}
				for i := 0; i < 5; i++ {
					tools = append(tools, mcptoolhandler.McpTool{Name: fmt.Sprintf("fan%d", i), InputSchema: mcpRefFanOutSchema()})
				}
				tl.EXPECT().ListTools(gomock.Any(), serverID).Return(tools, nil)
			},
			expectToolNames: []string{"mcp_d4d4d4d4_fan0", "mcp_d4d4d4d4_fan1", "mcp_d4d4d4d4_fan2", "mcp_d4d4d4d4_fan3"},
			expectToolMap: map[string]aicall.McpToolRef{
				"mcp_d4d4d4d4_fan0": {ServerID: uuid.FromStringOrNil("d4d4d4d4-1111-4000-8000-000000000014"), ToolName: "fan0"},
				"mcp_d4d4d4d4_fan1": {ServerID: uuid.FromStringOrNil("d4d4d4d4-1111-4000-8000-000000000014"), ToolName: "fan1"},
				"mcp_d4d4d4d4_fan2": {ServerID: uuid.FromStringOrNil("d4d4d4d4-1111-4000-8000-000000000014"), ToolName: "fan2"},
				"mcp_d4d4d4d4_fan3": {ServerID: uuid.FromStringOrNil("d4d4d4d4-1111-4000-8000-000000000014"), ToolName: "fan3"},
			},
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

			mergedTools, toolMap, err := h.resolveMcpOnly(context.Background(), tt.ai)
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

			for name, wantJSON := range tt.expectParams {
				var want map[string]any
				if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
					t.Fatalf("bad expectParams JSON for %s: %v", name, err)
				}
				found := false
				for _, mt := range mergedTools {
					if string(mt.Name) != name {
						continue
					}
					found = true
					if !reflect.DeepEqual(jsonRoundTrip(t, mt.Parameters), jsonRoundTrip(t, want)) {
						got, _ := json.Marshal(mt.Parameters)
						t.Errorf("Wrong match. %s parameters\nexpect: %s\ngot: %s", name, wantJSON, got)
					}
				}
				if !found {
					t.Errorf("Wrong match. expect tool %s in the advertised list", name)
				}
			}
		})
	}
}

// jsonRoundTrip normalizes v's number and nil types for DeepEqual.
func jsonRoundTrip(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal: %v", err)
	}
	var res any
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatalf("could not unmarshal: %v", err)
	}
	return res
}

// mcpRefFanOutSchema is a tool schema whose output charge is well above its
// raw size: 8 properties each $ref one string with a 7,400-byte description,
// padded with a non-allowlisted root key that costs raw bytes but no charge.
func mcpRefFanOutSchema() json.RawMessage {
	props := []string{}
	for i := 0; i < 8; i++ {
		props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/$defs/T"}`, i))
	}
	return json.RawMessage(`{"type":"object","x-pad":"` + strings.Repeat("p", 27600) +
		`","$defs":{"T":{"type":"string","description":"` + strings.Repeat("d", 7400) +
		`"}},"properties":{` + strings.Join(props, ",") + `}}`)
}

// Test_mcpRefFanOutSchema_Sizes pins the premise of the output budget row in
// Test_resolveMcpOnly: five of these schemas fit the raw schemaBudget with
// bytes to spare, each fits the per-tool output cap, and exactly four fit
// the per-resolution output budget.
func Test_mcpRefFanOutSchema_Sizes(t *testing.T) {
	raw := mcpRefFanOutSchema()
	if 5*len(raw) >= mcpToolSchemaBudgetBytes {
		t.Errorf("Wrong match. five schemas are %d raw bytes, must stay under the %d schemaBudget", 5*len(raw), mcpToolSchemaBudgetBytes)
	}

	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("could not decode: %v", err)
	}
	_, rep := mcpschema.Normalize(params, mcpMaxToolSchemaBytes)
	if rep.ToolDropped {
		t.Fatalf("Wrong match. the schema must fit the per-tool cap, got %+v", rep)
	}
	if rep.OutBytes < 2*len(raw)*3/4 {
		t.Errorf("Wrong match. charge %d must be well above the raw size %d", rep.OutBytes, len(raw))
	}
	if 4*rep.OutBytes > mcpToolSchemaBudgetBytes || 5*rep.OutBytes <= mcpToolSchemaBudgetBytes {
		t.Errorf("Wrong match. charge %d must fit four times, not five, in %d", rep.OutBytes, mcpToolSchemaBudgetBytes)
	}
}

// Test_resolveMcpOnly_SchemaNormalizationLogs pins the design section 15.4
// log lines: one WARN per dropped tool, one WARN per kept tool that lost
// optional properties, and nothing at WARN for routine key stripping.
func Test_resolveMcpOnly_SchemaNormalizationLogs(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(prevLevel)

	mc := gomock.NewController(t)
	defer mc.Finish()

	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)
	serverID := uuid.FromStringOrNil("e5e5e5e5-1111-4000-8000-000000000015")
	srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		Identity: commonidentityFor(serverID),
		Status:   mcpserver.StatusActive,
	}, nil)
	tl.EXPECT().ListTools(gomock.Any(), serverID).Return([]mcptoolhandler.McpTool{
		{Name: "dropped", InputSchema: json.RawMessage(`{"type":"object","properties":{"v":{"type":"array"}},"required":["v"]}`)},
		{Name: "trimmed", InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{},"b":{},"c":{},"d":{},"k":{"type":"string"}}}`)},
		{Name: "stripped", InputSchema: json.RawMessage(`{"type":"object","title":"S","properties":{"o":{"type":"string","x-mcp-header":"o"}}}`)},
	}, nil)

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	if _, _, err := h.resolveMcpOnly(context.Background(), &ai.AI{
		Identity:     commonidentity.Identity{CustomerID: testCustomerID},
		McpServerIDs: []uuid.UUID{serverID},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	warns := []string{}
	debugs := []string{}
	for _, e := range hook.AllEntries() {
		if e.Data["func"] != "resolveMcpOnly" {
			continue
		}
		switch e.Level {
		case logrus.WarnLevel:
			warns = append(warns, e.Message)
			if e.Data["mcp_server_id"] != serverID || e.Data["tool_name"] == nil {
				t.Errorf("Wrong match. WARN must carry mcp_server_id and tool_name fields, got %v", e.Data)
			}
		case logrus.DebugLevel:
			debugs = append(debugs, e.Message)
		}
	}

	wantWarns := []string{
		"Dropped an mcp tool whose input schema cannot be made provider-safe. mcp_server_id: e5e5e5e5-1111-4000-8000-000000000015, tool_name: mcp_e5e5e5e5_dropped, reason: array_without_items, path: /properties/v",
		`Removed optional parameters an mcp tool's input schema cannot express provider-safely. mcp_server_id: e5e5e5e5-1111-4000-8000-000000000015, tool_name: mcp_e5e5e5e5_trimmed, dropped_properties: 4, first: "/properties/a", "/properties/b", "/properties/c"`,
	}
	if !reflect.DeepEqual(warns, wantWarns) {
		t.Errorf("Wrong match.\nexpect: %q\ngot: %q", wantWarns, warns)
	}
	if len(debugs) != 1 || !strings.Contains(debugs[0], "dropped_keys: 2") {
		t.Errorf("Wrong match. expect one DEBUG totals line with dropped_keys: 2, got %q", debugs)
	}
}

// Test_resolveMcpOnly_NilHandlers pins that resolveMcpOnly degrades gracefully
// (returns only built-ins, never panics) when mcpServerHandler/mcptoolHandler
// are nil -- the state every test-constructed aicallHandler that doesn't
// explicitly wire MCP support is in, and the state cmd/ai-control's minimal
// AIcallHandler construction is in today.
func Test_resolveMcpOnly_NilHandlers(t *testing.T) {
	h := &aicallHandler{}

	mergedTools, toolMap, err := h.resolveMcpOnly(context.Background(), &ai.AI{
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
				tl.EXPECT().CallTool(gomock.Any(), serverID, "search_tickets", gomock.Any()).Return("3 tickets found", false, nil)
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
				tl.EXPECT().CallTool(gomock.Any(), serverID, "search_tickets", gomock.Any()).Return("ok", false, nil)
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
					Return("", false, errorWithSecret())
			},
			wantResult:      "failed",
			wantCallToolHit: true,
			wantMessage:     "MCP tool call failed",
		},
		{
			// B24: a remote tool that returns isError:true must never be
			// labeled a success -- CallTool's second return, IsError, was
			// never read before this. The 200-char cap already used for
			// transport errors applies here too.
			name:     "fail closed: isError:true result is never labeled a success (B24)",
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
					Return("remote tool reported: order not found", true, nil)
			},
			wantResult:      "failed",
			wantCallToolHit: true,
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
// written by resolveMcpOnly's callers, and the map[string]any shape Metadata
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
				tl.EXPECT().CallTool(gomock.Any(), curOnlyServerID, "search_tickets", gomock.Any()).Return("ok", false, nil)
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
				tl.EXPECT().CallTool(gomock.Any(), startOnlyServerID, "search_tickets", gomock.Any()).Return("ok", false, nil)
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

// Test_capErrText_RuneSafe pins that the log-line cap never splits a
// multi-byte character. The previous byte slice turned a remote server's
// Korean or emoji error text into invalid UTF-8 at the cut point.
func Test_capErrText_RuneSafe(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{name: "shorter than the cap is untouched", in: "short", max: 200, want: "short"},
		{name: "ascii is cut at the cap", in: "abcdef", max: 3, want: "abc"},
		{name: "cut inside a three-byte rune backs off to its start", in: "ab가나", max: 4, want: "ab"},
		{name: "cut exactly on a rune boundary keeps the rune", in: "ab가나", max: 5, want: "ab가"},
		{name: "cut inside a four-byte rune", in: "x\U0001F600y", max: 3, want: "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capErrText(tt.in, tt.max)
			if got != tt.want {
				t.Errorf("capErrText(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result is not valid UTF-8: %q", got)
			}
			if len(got) > tt.max {
				t.Errorf("result is %d bytes, over the %d cap", len(got), tt.max)
			}
		})
	}
}

// Test_decodeToolSchema pins the per-tool and per-resolution limits on
// decoding remote input schemas, that a rejected schema costs nothing, and
// that running out of the shared budget is told apart from a bad schema.
func Test_decodeToolSchema(t *testing.T) {
	big := json.RawMessage(`{"type":"object","description":"` + strings.Repeat("x", mcpMaxToolSchemaBytes) + `"}`)

	tests := []struct {
		name       string
		raw        json.RawMessage
		budget     int
		want       schemaVerdict
		wantParams bool
		wantBudget int
	}{
		{name: "absent schema", raw: nil, budget: 100, want: schemaOK, wantBudget: 100},
		{name: "null schema", raw: json.RawMessage(` null `), budget: 100, want: schemaOK, wantBudget: 100},
		{name: "small object", raw: json.RawMessage(`{"type":"object"}`), budget: 100, want: schemaOK, wantParams: true, wantBudget: 100 - len(`{"type":"object"}`)},
		{name: "over the per-tool limit", raw: big, budget: mcpToolSchemaBudgetBytes, want: schemaInvalid, wantBudget: mcpToolSchemaBudgetBytes},
		{name: "over what is left of the budget", raw: json.RawMessage(`{"type":"object"}`), budget: 5, want: schemaOverBudget, wantBudget: 5},
		{name: "not an object", raw: json.RawMessage(`[1,2]`), budget: 100, want: schemaInvalid, wantBudget: 100},
		{name: "malformed", raw: json.RawMessage(`{"type":`), budget: 100, want: schemaInvalid, wantBudget: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := tt.budget
			params, got := decodeToolSchema(tt.raw, &budget)
			if got != tt.want || (params != nil) != tt.wantParams || budget != tt.wantBudget {
				t.Fatalf("got verdict=%v params=%v budget=%d; want verdict=%v params=%v budget=%d", got, params != nil, budget, tt.want, tt.wantParams, tt.wantBudget)
			}
		})
	}
}

// Test_validMcpToolName pins which remote tool names are kept.
func Test_validMcpToolName(t *testing.T) {
	tests := map[string]bool{
		"search_tickets":                         true,
		"a-b_C9":                                 true,
		strings.Repeat("a", mcpMaxToolNameLen):   true,
		strings.Repeat("a", mcpMaxToolNameLen+1): false,
		"":                                       false,
		"has space":                              false,
		"dot.name":                               false,
		"<script>":                               false,
		"caf\u00e9":                              false,
	}
	for name, want := range tests {
		if got := validMcpToolName(name); got != want {
			t.Errorf("validMcpToolName(%q) = %v, want %v", name, got, want)
		}
	}
	if mcpMaxToolNameLen+len("mcp_12345678_") != 64 {
		t.Fatalf("a namespaced name at the limit must be exactly 64 bytes, got %d", mcpMaxToolNameLen+len("mcp_12345678_"))
	}
}

// Test_resolveMcpToolMap_Bounds pins that the stored tool map is bounded in
// entries and bytes however many servers and tools are listed, drops invalid
// names while keeping the server's other tools, and never decodes schemas.
func Test_resolveMcpToolMap_Bounds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	// A schema that is not JSON at all: decoding it would fail, so if the
	// map path ever decoded schemas, these tools would go missing.
	undecodable := json.RawMessage(`{`)

	ids := []uuid.UUID{}
	for i := 0; i < 5; i++ {
		id := uuid.Must(uuid.NewV4())
		ids = append(ids, id)
		srv.EXPECT().Get(gomock.Any(), id).Return(&mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive}, nil).AnyTimes()

		tools := []mcptoolhandler.McpTool{{Name: "bad name"}, {Name: "<x>"}, {Name: strings.Repeat("n", mcpMaxToolNameLen+1)}}
		// 100 per server does not divide the cap, so the cap must cut a
		// server's list partway, not only stop at a server boundary.
		for j := 0; j < 100; j++ {
			tools = append(tools, mcptoolhandler.McpTool{Name: fmt.Sprintf("%s%03d", strings.Repeat("n", mcpMaxToolNameLen-3), j), InputSchema: undecodable})
		}
		tl.EXPECT().ListTools(gomock.Any(), id).Return(tools, nil).AnyTimes()
	}

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	toolMap := h.resolveMcpToolMap(context.Background(), &ai.AI{
		Identity:     commonidentity.Identity{CustomerID: testCustomerID},
		McpServerIDs: ids,
	})

	if len(toolMap) != mcpMaxToolsPerResolution {
		t.Fatalf("tool map has %d entries, want the per-resolution cap %d", len(toolMap), mcpMaxToolsPerResolution)
	}
	for name := range toolMap {
		if len(name) > 64 || strings.ContainsAny(name, " <>") {
			t.Fatalf("invalid name stored: %q", name)
		}
	}
	encoded, err := json.Marshal(toolMap)
	if err != nil {
		t.Fatalf("could not encode: %v", err)
	}
	if len(encoded) > 64<<10 {
		t.Fatalf("stored tool map is %d bytes; it must stay a few tens of KiB", len(encoded))
	}
}

// Test_resolveTools_SchemaLimits pins that a tool whose schema is too large
// is dropped while the server's other tools are kept, and that the total
// decoded across a resolution is bounded.
func Test_resolveMcpOnly_SchemaLimits(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	serverID := uuid.FromStringOrNil("eeeeeeee-1111-4000-8000-000000000005")
	srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		Identity: commonidentityFor(serverID),
		Status:   mcpserver.StatusActive,
	}, nil)

	// Each schema is just under the per-tool limit, so the budget admits
	// only a few of them; one more is far over the per-tool limit.
	nearLimit := json.RawMessage(`{"type":"object","description":"` + strings.Repeat("x", mcpMaxToolSchemaBytes-64) + `"}`)
	tools := []mcptoolhandler.McpTool{
		{Name: "huge", InputSchema: json.RawMessage(`{"d":"` + strings.Repeat("x", 2*mcpMaxToolSchemaBytes) + `"}`)},
		{Name: "plain"},
	}
	for i := 0; i < 8; i++ {
		tools = append(tools, mcptoolhandler.McpTool{Name: fmt.Sprintf("big%d", i), InputSchema: nearLimit})
	}
	tl.EXPECT().ListTools(gomock.Any(), serverID).Return(tools, nil)

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	merged, toolMap, err := h.resolveMcpOnly(context.Background(), &ai.AI{
		Identity:     commonidentity.Identity{CustomerID: testCustomerID},
		McpServerIDs: []uuid.UUID{serverID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := toolMap["mcp_eeeeeeee_huge"]; ok {
		t.Error("a schema over the per-tool limit must be dropped")
	}
	if _, ok := toolMap["mcp_eeeeeeee_plain"]; !ok {
		t.Error("a tool with no schema must be kept")
	}

	decoded := 0
	for _, tl := range merged {
		if strings.HasPrefix(string(tl.Name), "mcp_eeeeeeee_big") {
			decoded++
		}
	}
	wantDecoded := mcpToolSchemaBudgetBytes / len(nearLimit)
	if decoded != wantDecoded {
		t.Fatalf("decoded %d near-limit schemas, want %d within the %d byte budget", decoded, wantDecoded, mcpToolSchemaBudgetBytes)
	}
	if len(toolMap) != len(merged) {
		t.Fatalf("tool map (%d) and merged list (%d) must describe the same tools", len(toolMap), len(merged))
	}
}

// Test_discoverMcpTools_SlotsBoundConcurrency pins that at most
// cap(mcpDiscoverySlots) tools/list requests run at once in the process,
// however many session starts resolve at the same time.
// Test_discoverMcpTools_InsightGate pins B10's residual gate: discovery must
// never even attempt to list a whitelisted server's tools for an Insight AI,
// closing the last hole after PR B1's write-gate (a pre-B1 or exempted
// Insight AI could otherwise still trigger live discovery, though not
// advertisement, via writeInsightSessionMetadata's unconditional
// resolveMcpToolMap call).
func Test_discoverMcpTools_InsightGate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// No EXPECT() calls set on either mock: if discoverMcpTools reaches
	// mcpServerHandler.Get or mcptoolHandler.ListTools for an Insight AI,
	// gomock fails this test on the unexpected call.
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	a := &ai.AI{
		Identity:     commonidentity.Identity{CustomerID: testCustomerID},
		Type:         ai.TypeInsight,
		McpServerIDs: []uuid.UUID{uuid.Must(uuid.NewV4())},
	}

	got := h.discoverMcpTools(context.Background(), a, true)
	if len(got) != 0 {
		t.Errorf("expected no discovered tools for an insight AI, got: %v", got)
	}
}

// Test_resolveMcpOnly_SessionStartBudget is the regression test for review
// round 1's §2.3a Critical finding.
//
// resolveMcpOnly's discoverMcpTools loop had no aggregate timeout of its own:
// mcpDiscoverySlotWait (2s) only bounds time spent waiting for a free
// discovery slot, and mcp_tool_call_timeout_seconds bounds one server's
// transport round-trip in isolation, not the whole loop across up to
// ai.MaxMcpServerIDs (8) servers. A slow/hanging server that never returns
// (and is not itself killed by a per-call timeout, simulating a
// misconfigured or malicious remote that ignores the request but never
// closes the connection) must not be allowed to block resolveMcpOnly beyond
// mcpSessionStartDiscoveryBudgetSeconds.
func Test_resolveMcpOnly_SessionStartBudget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	oldBudget := mcpSessionStartDiscoveryBudget
	mcpSessionStartDiscoveryBudget = 50 * time.Millisecond
	defer func() { mcpSessionStartDiscoveryBudget = oldBudget }()

	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	serverID := uuid.FromStringOrNil("aaaaaaaa-2222-4000-8000-000000000001")
	srv.EXPECT().Get(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		Identity: commonidentityFor(serverID),
		Status:   mcpserver.StatusActive,
	}, nil).AnyTimes()

	// ListTools never returns on its own; it only unblocks when the ctx
	// passed to it is cancelled. Without a session-start budget wrapping
	// the loop, this would hang resolveMcpOnly indefinitely (or until
	// whatever mcp_tool_call_timeout_seconds is, which is not this
	// function's context to set up).
	tl.EXPECT().ListTools(gomock.Any(), serverID).DoAndReturn(
		func(ctx context.Context, _ uuid.UUID) ([]mcptoolhandler.McpTool, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	).AnyTimes()

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	a := &ai.AI{
		Identity:     commonidentity.Identity{CustomerID: testCustomerID},
		McpServerIDs: []uuid.UUID{serverID},
	}

	done := make(chan struct{})
	start := time.Now()
	go func() {
		_, _, _ = h.resolveMcpOnly(context.Background(), a)
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		if elapsed > 2*time.Second {
			t.Errorf("resolveMcpOnly took %v, expected it to return within the session-start discovery budget", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resolveMcpOnly did not return within 2s; the session-start discovery budget is not being enforced")
	}
}

func Test_discoverMcpTools_SlotsBoundConcurrency(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	var mu sync.Mutex
	inFlight, maxInFlight := 0, 0
	ids := []uuid.UUID{}
	for i := 0; i < 3; i++ {
		id := uuid.Must(uuid.NewV4())
		ids = append(ids, id)
		srv.EXPECT().Get(gomock.Any(), id).Return(&mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive}, nil).AnyTimes()
		tl.EXPECT().ListTools(gomock.Any(), id).DoAndReturn(func(context.Context, uuid.UUID) ([]mcptoolhandler.McpTool, error) {
			mu.Lock()
			inFlight++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			inFlight--
			mu.Unlock()
			return []mcptoolhandler.McpTool{{Name: "t"}}, nil
		}).AnyTimes()
	}

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	a := &ai.AI{Identity: commonidentity.Identity{CustomerID: testCustomerID}, McpServerIDs: ids}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := len(h.resolveMcpToolMap(context.Background(), a)); got != 3 {
				t.Errorf("resolution got %d tools, want 3", got)
			}
		}()
	}
	wg.Wait()

	// The bound is a memory decision measured against the 40M container
	// limit, so it is pinned by value, not only against the channel.
	const wantSlots = 2
	if cap(mcpDiscoverySlots) != wantSlots {
		t.Fatalf("mcpDiscoverySlots has %d slots, want %d; re-measure peak memory before changing it", cap(mcpDiscoverySlots), wantSlots)
	}
	if maxInFlight > wantSlots {
		t.Fatalf("%d tools/list requests ran at once; the process-wide bound is %d", maxInFlight, wantSlots)
	}
}

// Test_discoverMcpTools_SlotWaitIsBounded pins that a resolution does not
// wait indefinitely for a slot held by someone else's slow servers: it skips
// the server once its total wait is spent.
func Test_discoverMcpTools_SlotWaitIsBounded(t *testing.T) {
	defer shortenSlotWait(t, 300*time.Millisecond)()

	// Occupy every slot, as another customer's slow listings would.
	for i := 0; i < cap(mcpDiscoverySlots); i++ {
		mcpDiscoverySlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(mcpDiscoverySlots); i++ {
			<-mcpDiscoverySlots
		}
	}()

	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	ids := []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())}
	for _, id := range ids {
		srv.EXPECT().Get(gomock.Any(), id).Return(&mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive}, nil)
	}
	// No ListTools expectation: no server may be listed without a slot.

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	start := time.Now()
	// Run aside: an unbounded wait would never return, and must fail here
	// rather than hang the test binary.
	resolved := make(chan map[string]aicall.McpToolRef, 1)
	go func() {
		resolved <- h.resolveMcpToolMap(context.Background(), &ai.AI{Identity: commonidentity.Identity{CustomerID: testCustomerID}, McpServerIDs: ids})
	}()
	var toolMap map[string]aicall.McpToolRef
	select {
	case toolMap = <-resolved:
	case <-time.After(mcpDiscoverySlotWait + time.Second):
		t.Fatal("the resolution waited for a slot past its bound")
	}
	elapsed := time.Since(start)

	if len(toolMap) != 0 {
		t.Fatalf("got %d tools with no slot free", len(toolMap))
	}
	// Two servers share one total wait, not one wait each.
	if elapsed < mcpDiscoverySlotWait || elapsed > mcpDiscoverySlotWait+500*time.Millisecond {
		t.Fatalf("resolution took %v; its total slot wait is %v", elapsed, mcpDiscoverySlotWait)
	}
}

// shortenSlotWait sets mcpDiscoverySlotWait for one test and returns the
// function that restores it.
func shortenSlotWait(t *testing.T, d time.Duration) func() {
	t.Helper()
	prev := mcpDiscoverySlotWait
	mcpDiscoverySlotWait = d
	return func() { mcpDiscoverySlotWait = prev }
}

// Test_discoverMcpTools_ListingTimeIsNotWaitTime pins that time spent
// listing a customer's own servers is not charged to the slot wait. With no
// other load, a first server slower than the whole wait must not cause the
// servers after it to be skipped.
func Test_discoverMcpTools_ListingTimeIsNotWaitTime(t *testing.T) {
	defer shortenSlotWait(t, 100*time.Millisecond)()

	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	ids := []uuid.UUID{}
	for i := 0; i < 4; i++ {
		id := uuid.Must(uuid.NewV4())
		ids = append(ids, id)
		srv.EXPECT().Get(gomock.Any(), id).Return(&mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive}, nil)
		delay := time.Duration(0)
		if i == 0 {
			delay = 250 * time.Millisecond
		}
		tl.EXPECT().ListTools(gomock.Any(), id).DoAndReturn(func(context.Context, uuid.UUID) ([]mcptoolhandler.McpTool, error) {
			time.Sleep(delay)
			return []mcptoolhandler.McpTool{{Name: "t"}}, nil
		})
	}

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	// Several runs: the defect this pins dropped servers at random.
	for run := 0; run < 5; run++ {
		if run > 0 {
			for _, id := range ids {
				srv.EXPECT().Get(gomock.Any(), id).Return(&mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive}, nil)
			}
			for i, id := range ids {
				delay := time.Duration(0)
				if i == 0 {
					delay = 250 * time.Millisecond
				}
				tl.EXPECT().ListTools(gomock.Any(), id).DoAndReturn(func(context.Context, uuid.UUID) ([]mcptoolhandler.McpTool, error) {
					time.Sleep(delay)
					return []mcptoolhandler.McpTool{{Name: "t"}}, nil
				})
			}
		}
		toolMap := h.resolveMcpToolMap(context.Background(), &ai.AI{Identity: commonidentity.Identity{CustomerID: testCustomerID}, McpServerIDs: ids})
		if len(toolMap) != len(ids) {
			t.Fatalf("run %d: got %d servers' tools, want all %d; listing time must not count as waiting", run, len(toolMap), len(ids))
		}
	}
}

// Test_listToolsWithSlot_ReleasesOnPanic pins that a panic out of ListTools
// does not keep a slot, so one failure cannot starve every later discovery
// in the process.
func Test_listToolsWithSlot_ReleasesOnPanic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)
	id := uuid.Must(uuid.NewV4())
	tl.EXPECT().ListTools(gomock.Any(), id).DoAndReturn(func(context.Context, uuid.UUID) ([]mcptoolhandler.McpTool, error) {
		panic("boom")
	})

	h := &aicallHandler{mcptoolHandler: tl}
	func() {
		defer func() { _ = recover() }()
		wait := time.Second
		_, _, _ = h.listToolsWithSlot(context.Background(), id, &wait)
	}()

	if held := len(mcpDiscoverySlots); held != 0 {
		t.Fatalf("%d slot(s) still held after a panic", held)
	}
}

// Test_sampleToolNames pins that dropped names are logged boundedly.
func Test_sampleToolNames(t *testing.T) {
	got := sampleToolNames([]string{"a.b", "c/d", strings.Repeat("x", 500), "fourth"})
	if strings.Contains(got, "fourth") {
		t.Errorf("more than three names logged: %s", got)
	}
	if !strings.Contains(got, `"a.b"`) || !strings.Contains(got, `"c/d"`) {
		t.Errorf("names missing: %s", got)
	}
	if len(got) > 3*(80+8) {
		t.Errorf("log text not bounded: %d bytes", len(got))
	}
}
