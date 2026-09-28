package mcptoolhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// These tests pin the MCP Streamable HTTP behaviour listed in section 4b of
// docs/plans/2026-09-27-mcp-phase2-tool-exposure-analysis-v2.md. Every case
// runs against fakeMCPServer, which rejects the way the reference server
// does; the previous client (one bare POST, Accept: application/json only,
// no initialize, JSON-only parsing, hardcoded id 1) fails all of them.

func listToolsAgainst(t *testing.T, fake *fakeMCPServer) ([]McpTool, error) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	serverID := uuid.Must(uuid.NewV4())
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		URL:      srv.URL,
		Status:   mcpserver.StatusActive,
		AuthType: mcpserver.AuthTypeNone,
	}, nil)

	return newTestHandler(t, mockDB).ListTools(context.Background(), serverID)
}

func Test_Protocol_ListTools(t *testing.T) {
	tests := []struct {
		name string

		configure func(f *fakeMCPServer)

		wantErr      string
		wantMethods  string
		wantSessions int
		wantTool     string
	}{
		{
			// Requirements 1, 2, 4, 5, 6, 8: the default stateful server
			// answering SSE-framed. The old client got 406 here.
			name:         "stateful server with SSE framing",
			configure:    func(f *fakeMCPServer) {},
			wantMethods:  "initialize,notifications/initialized,tools/list",
			wantSessions: 1,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 6: no session header means stateless, with no
			// separate branch in the client.
			name: "stateless server with JSON bodies",
			configure: func(f *fakeMCPServer) {
				f.stateless = true
				f.jsonResponse = true
			},
			wantMethods:  "initialize,notifications/initialized,tools/list",
			wantSessions: 1,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 8: the matching event is found among a
			// notification and a response to a different id.
			name:         "SSE stream with unrelated events before the answer",
			configure:    func(f *fakeMCPServer) { f.sseNoise = true },
			wantMethods:  "initialize,notifications/initialized,tools/list",
			wantSessions: 1,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 5: an older handshake-era version is echoed as
			// negotiated, not replaced with the one we asked for.
			name:         "server negotiates an older handshake version",
			configure:    func(f *fakeMCPServer) { f.negotiate = "2025-03-26" },
			wantMethods:  "initialize,notifications/initialized,tools/list",
			wantSessions: 1,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 5: a version outside the handshake set is refused
			// before any tools/list is sent.
			name:         "server negotiates a version outside the handshake era",
			configure:    func(f *fakeMCPServer) { f.negotiate = "2026-07-28" },
			wantErr:      `negotiated protocol version "2026-07-28"`,
			wantMethods:  "initialize",
			wantSessions: 1,
		},
		{
			// Requirement 3: HTTP 200 plus a usable session id must not be
			// mistaken for a successful initialize.
			name:         "initialize answered 200 with a JSON-RPC error and a session id",
			configure:    func(f *fakeMCPServer) { f.initializeErrorWithSession = true },
			wantErr:      "Invalid request parameters",
			wantMethods:  "initialize",
			wantSessions: 1,
		},
		{
			// Requirement 11: a dropped session is re-opened exactly once.
			name:         "session expires once and is re-initialized",
			configure:    func(f *fakeMCPServer) { f.expireSessionsOnce = true },
			wantMethods:  "initialize,notifications/initialized,tools/list,initialize,notifications/initialized,tools/list",
			wantSessions: 2,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 11: the re-initialization does not loop.
			name:         "session keeps expiring and the client stops after one retry",
			configure:    func(f *fakeMCPServer) { f.expireAlways = true },
			wantErr:      "404",
			wantMethods:  "initialize,notifications/initialized,tools/list,initialize,notifications/initialized,tools/list",
			wantSessions: 2,
		},
		{
			// Requirement 11: a stateless server has no session to expire, so
			// its 404 is final.
			name: "stateless 404 is not retried",
			configure: func(f *fakeMCPServer) {
				f.stateless = true
				f.statusOnMethod = http.StatusNotFound
				f.statusBody = "no such endpoint"
			},
			wantErr:      "404",
			wantMethods:  "initialize,notifications/initialized,tools/list",
			wantSessions: 1,
		},
		{
			// Requirement 10: the reason the server gave is kept.
			name: "non-2xx body survives into the error",
			configure: func(f *fakeMCPServer) {
				f.statusOnMethod = http.StatusBadRequest
				f.statusBody = "Bad Request: something specific"
			},
			wantErr:      "Bad Request: something specific",
			wantMethods:  "initialize,notifications/initialized,tools/list",
			wantSessions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeMCPServer(t)
			tt.configure(fake)

			tools, err := listToolsAgainst(t, fake)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got tools %+v", tt.wantErr, tools)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error to contain %q, got: %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := strings.Join(fake.methods(), ","); got != tt.wantMethods {
				t.Errorf("request sequence\n got: %s\nwant: %s", got, tt.wantMethods)
			}
			if fake.sessions != tt.wantSessions {
				t.Errorf("expected %d initialize call(s), got %d", tt.wantSessions, fake.sessions)
			}

			if tt.wantTool != "" {
				if len(tools) != 1 || tools[0].Name != tt.wantTool {
					t.Fatalf("unexpected tools: %+v", tools)
				}
			}
		})
	}
}

// Test_Protocol_Headers asserts header VALUES, not presence: an echoed
// constant would pass a presence check while breaking any server that
// negotiated something else.
func Test_Protocol_Headers(t *testing.T) {
	fake := newFakeMCPServer(t)
	fake.negotiate = "2025-06-18"

	if _, err := listToolsAgainst(t, fake); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := fake.recorded()
	if len(reqs) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(reqs))
	}

	for _, r := range reqs {
		if r.Accept != "application/json, text/event-stream" {
			t.Errorf("request %q: Accept = %q", r.Method, r.Accept)
		}
	}

	init := reqs[0]
	if init.SessionID != "" || init.ProtocolVersion != "" {
		t.Errorf("initialize must carry neither a session id nor a protocol version, got session=%q version=%q", init.SessionID, init.ProtocolVersion)
	}
	for _, key := range []string{`"protocolVersion"`, `"capabilities"`, `"clientInfo"`} {
		if !strings.Contains(string(init.Params), key) {
			t.Errorf("initialize params missing %s: %s", key, init.Params)
		}
	}

	for _, r := range reqs[1:] {
		if r.SessionID != "session-1" {
			t.Errorf("request %q: session id = %q, want session-1", r.Method, r.SessionID)
		}
		if r.ProtocolVersion != "2025-06-18" {
			t.Errorf("request %q: protocol version = %q, want the negotiated 2025-06-18", r.Method, r.ProtocolVersion)
		}
	}
}

// Test_Protocol_RequestIDs pins requirement 9: ids increase within a session
// and a notification carries none.
func Test_Protocol_RequestIDs(t *testing.T) {
	fake := newFakeMCPServer(t)

	if _, err := listToolsAgainst(t, fake); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := fake.recorded()
	if got := string(reqs[0].ID); got != "1" {
		t.Errorf("initialize id = %s, want 1", got)
	}
	if got := string(reqs[1].ID); got != "" {
		t.Errorf("notifications/initialized must carry no id, got %s", got)
	}
	if got := string(reqs[2].ID); got != "2" {
		t.Errorf("tools/list id = %s, want 2", got)
	}
}

// Test_Protocol_InputSchemaParsed pins the camelCase wire field. Under the
// old snake_case tag this came back nil for every tool.
func Test_Protocol_InputSchemaParsed(t *testing.T) {
	fake := newFakeMCPServer(t)

	tools, err := listToolsAgainst(t, fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("unexpected tools: %+v", tools)
	}
	if tools[0].InputSchema == nil {
		t.Fatal("inputSchema was dropped")
	}
	if tools[0].InputSchema["type"] != "object" {
		t.Errorf("unexpected schema: %+v", tools[0].InputSchema)
	}
}

func Test_readJSONRPCResponse(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantID      int
		wantResult  string
		wantErr     string
	}{
		{
			name:        "bare JSON with matching id",
			contentType: "application/json",
			body:        `{"jsonrpc":"2.0","id":3,"result":{"ok":true}}`,
			wantID:      3,
			wantResult:  `{"ok":true}`,
		},
		{
			name:        "bare JSON with mismatched id is rejected",
			contentType: "application/json",
			body:        `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`,
			wantID:      3,
			wantErr:     "does not match request id 3",
		},
		{
			name:        "SSE with LF line endings and a charset parameter",
			contentType: "text/event-stream; charset=utf-8",
			body:        "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ok\":true}}\n\n",
			wantID:      2,
			wantResult:  `{"ok":true}`,
		},
		{
			name:        "SSE event split across two data lines",
			contentType: "text/event-stream",
			body:        "data: {\"jsonrpc\":\"2.0\",\ndata: \"id\":2,\"result\":{\"ok\":true}}\n\n",
			wantID:      2,
			wantResult:  `{"ok":true}`,
		},
		{
			name:        "SSE skips a server-to-client request that reuses the id",
			contentType: "text/event-stream",
			body:        "data: {\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"sampling/createMessage\"}\n\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ok\":true}}\n\n",
			wantID:      2,
			wantResult:  `{"ok":true}`,
		},
		{
			name:        "SSE final event without a trailing blank line",
			contentType: "text/event-stream",
			body:        "data: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ok\":true}}",
			wantID:      2,
			wantResult:  `{"ok":true}`,
		},
		{
			name:        "SSE stream that never answers",
			contentType: "text/event-stream",
			body:        ": keep-alive\n\ndata: {\"jsonrpc\":\"2.0\",\"id\":9,\"result\":{}}\n\n",
			wantID:      2,
			wantErr:     "ended without a response to request id 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := readJSONRPCResponse(tt.contentType, strings.NewReader(tt.body), tt.wantID)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(resp.Result) != tt.wantResult {
				t.Errorf("result = %s, want %s", resp.Result, tt.wantResult)
			}
		})
	}
}

// Test_truncation_RuneSafe pins requirement 12 for both truncation helpers
// in this package. The old byte slice cut the three-byte character in half.
func Test_truncation_RuneSafe(t *testing.T) {
	long := strings.Repeat("a", 511) + strings.Repeat("가", 10)

	got := truncateForError([]byte(long))
	if !utf8.ValidString(got) {
		t.Fatalf("truncateForError produced invalid UTF-8: %q", got)
	}
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Errorf("expected the truncation marker, got %q", got)
	}

	for n := 0; n <= 9; n++ {
		if s := truncateUTF8("가나다", n); !utf8.ValidString(s) || len(s) > n {
			t.Errorf("truncateUTF8(_, %d) = %q", n, s)
		}
	}
}
