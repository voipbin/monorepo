package mcptoolhandler

import (
	"context"
	"os"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// Test_Conformance_ReferenceServer runs the client against a real MCP
// reference server (the python `mcp` SDK) instead of the hermetic fake. The
// fake encodes what the reference server does; this test is what catches the
// fake drifting from it.
//
// It is skipped unless the server URLs are provided, and deliberately gated at
// runtime rather than by a build tag: CI runs `go test $(go list ./...)` with
// no -tags, so a tagged file would compile out and silently never run.
//
//	MCP_CONFORMANCE_STATEFUL_URL   reference server in its default mode
//	MCP_CONFORMANCE_STATELESS_URL  reference server with stateless_http and
//	                               json_response enabled
//
// Both servers must expose a tool named lookup_order with a string order_id
// parameter.
func Test_Conformance_ReferenceServer(t *testing.T) {
	targets := []struct {
		name string
		env  string
	}{
		{name: "stateful default", env: "MCP_CONFORMANCE_STATEFUL_URL"},
		{name: "stateless json", env: "MCP_CONFORMANCE_STATELESS_URL"},
	}

	ran := 0
	for _, target := range targets {
		url := os.Getenv(target.env)
		if url == "" {
			continue
		}
		ran++

		t.Run(target.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			serverID := uuid.Must(uuid.NewV4())
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{
				URL:      url,
				Status:   mcpserver.StatusActive,
				AuthType: mcpserver.AuthTypeNone,
			}, nil).Times(2)

			h := newTestHandler(t, mockDB)

			tools, err := h.ListTools(context.Background(), serverID)
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			var found *McpTool
			for i := range tools {
				if tools[i].Name == "lookup_order" {
					found = &tools[i]
				}
			}
			if found == nil {
				t.Fatalf("lookup_order not advertised: %+v", tools)
			}
			if found.InputSchema == nil || found.InputSchema["type"] != "object" {
				t.Fatalf("inputSchema missing or malformed: %+v", found.InputSchema)
			}

			text, err := h.CallTool(context.Background(), serverID, "lookup_order", `{"order_id":"A-1"}`)
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if text == "" {
				t.Fatal("CallTool returned no text")
			}
		})
	}

	if ran == 0 {
		t.Skip("set MCP_CONFORMANCE_STATEFUL_URL and/or MCP_CONFORMANCE_STATELESS_URL to run against a reference MCP server")
	}
}
