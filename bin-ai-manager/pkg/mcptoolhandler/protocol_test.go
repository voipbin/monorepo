package mcptoolhandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
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
			wantMethods:  "initialize,notifications/initialized,tools/list,DELETE",
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
			wantMethods:  "initialize,notifications/initialized,tools/list,DELETE",
			wantSessions: 1,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 5: an older handshake-era version is echoed as
			// negotiated, not replaced with the one we asked for.
			name:         "server negotiates an older handshake version",
			configure:    func(f *fakeMCPServer) { f.negotiate = "2025-03-26" },
			wantMethods:  "initialize,notifications/initialized,tools/list,DELETE",
			wantSessions: 1,
			wantTool:     "lookup_order",
		},
		{
			// Requirement 5: a version outside the handshake set is refused
			// before any tools/list is sent.
			name:         "server negotiates a version outside the handshake era",
			configure:    func(f *fakeMCPServer) { f.negotiate = "2026-07-28" },
			wantErr:      `negotiated protocol version "2026-07-28"`,
			wantMethods:  "initialize,DELETE",
			wantSessions: 1,
		},
		{
			// Requirement 3: HTTP 200 plus a usable session id must not be
			// mistaken for a successful initialize.
			name:         "initialize answered 200 with a JSON-RPC error and a session id",
			configure:    func(f *fakeMCPServer) { f.initializeErrorWithSession = true },
			wantErr:      "Invalid request parameters",
			wantMethods:  "initialize,DELETE",
			wantSessions: 1,
		},
		{
			// Requirement 11: a dropped session is re-opened exactly once.
			name:         "session expires once and is re-initialized",
			configure:    func(f *fakeMCPServer) { f.expireSessionsOnce = true },
			wantMethods:  "initialize,notifications/initialized,tools/list,initialize,notifications/initialized,tools/list,DELETE",
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
			wantMethods:  "initialize,notifications/initialized,tools/list,DELETE",
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
	if len(reqs) != 4 || reqs[3].HTTPMethod != http.MethodDelete {
		t.Fatalf("expected initialize, notification, method and a closing DELETE, got %d requests: %v", len(reqs), fake.methods())
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
			name:        "SSE data field with no space after the colon",
			contentType: "text/event-stream",
			body:        "data:{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ok\":true}}\n\n",
			wantID:      2,
			wantResult:  `{"ok":true}`,
		},
		{
			name:        "SSE null-id error answers the request",
			contentType: "text/event-stream",
			body:        "data: {\"jsonrpc\":\"2.0\",\"id\":null,\"error\":{\"code\":-32700,\"message\":\"bad\"}}\n\n",
			wantID:      2,
			wantResult:  ``,
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

// Test_Protocol_SessionClose pins that a stateful session is ended with one
// DELETE carrying the session id, the negotiated version and the credential,
// and that a stateless session is not. Without it every call left a session
// open on the customer's server until the server's idle timeout.
func Test_Protocol_SessionClose(t *testing.T) {
	t.Run("stateful: one DELETE for the session, with auth", func(t *testing.T) {
		fake := newFakeMCPServer(t)
		_, err := listToolsBearer(t, fake, "tok-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := fake.closedSessions(); strings.Join(got, ",") != "session-1" {
			t.Fatalf("closed sessions = %v, want [session-1]", got)
		}
		for _, r := range fake.recorded() {
			if r.Authorization != "Bearer tok-1" {
				t.Errorf("%s %q carried auth %q", r.HTTPMethod, r.Method, r.Authorization)
			}
		}
		last := fake.recorded()[len(fake.recorded())-1]
		if last.ProtocolVersion != "2025-11-25" {
			t.Errorf("DELETE protocol version = %q", last.ProtocolVersion)
		}
	})

	t.Run("stateless: no DELETE", func(t *testing.T) {
		fake := newFakeMCPServer(t)
		fake.stateless = true
		fake.jsonResponse = true
		if _, err := listToolsAgainst(t, fake); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range fake.recorded() {
			if r.HTTPMethod == http.MethodDelete {
				t.Fatal("a stateless session must not be closed")
			}
		}
	})

	t.Run("re-initialized: the replacement session carries auth and is the one closed", func(t *testing.T) {
		fake := newFakeMCPServer(t)
		fake.expireSessionsOnce = true
		if _, err := listToolsBearer(t, fake, "tok-2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range fake.recorded() {
			if r.Authorization != "Bearer tok-2" {
				t.Errorf("%s %q carried auth %q", r.HTTPMethod, r.Method, r.Authorization)
			}
		}
		if got := fake.closedSessions(); strings.Join(got, ",") != "session-2" {
			t.Fatalf("closed sessions = %v, want only the live replacement [session-2]", got)
		}
	})

	t.Run("a failing DELETE does not fail the call", func(t *testing.T) {
		fake := newFakeMCPServer(t)
		fake.deleteStatus = http.StatusMethodNotAllowed
		if _, err := listToolsAgainst(t, fake); err != nil {
			t.Fatalf("a refused close must not fail the call, got: %v", err)
		}
	})
}

// Test_Protocol_WholeCallDeadline pins that the configured timeout bounds the
// whole call. Each request here takes most of the timeout on its own, so a
// per-request deadline would let the call succeed after several multiples of
// it; the call must instead fail within about one.
func Test_Protocol_WholeCallDeadline(t *testing.T) {
	const timeout = 300 * time.Millisecond

	fake := newFakeMCPServer(t)
	fake.jsonResponse = true
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(timeout * 2 / 3)
		fake.ServeHTTP(w, r)
	})

	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := httptest.NewServer(slow)
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeNone,
	}, nil)

	h := newTestHandler(t, mockDB)
	h.timeout = timeout

	start := time.Now()
	_, err := h.ListTools(context.Background(), serverID)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the call to exceed its deadline")
	}
	if elapsed > 2*timeout {
		t.Fatalf("call took %v; the %v timeout must bound the whole call, not each request", elapsed, timeout)
	}
}

// Test_Protocol_SessionIDValidated pins that a session id is checked before
// it is echoed back in a request header.
func Test_Protocol_SessionIDValidated(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr string
	}{
		{name: "visible ASCII is accepted", id: "abc-123_XYZ.~"},
		{name: "over the length limit", id: strings.Repeat("a", maxSessionIDBytes+1), wantErr: "over the 1024 byte limit"},
		{name: "a space", id: "a b", wantErr: "outside visible ASCII"},
		{name: "a non-ASCII byte", id: "a\xc3\xa9", wantErr: "outside visible ASCII"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionID(tt.id)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// Test_Protocol_ServerControlledErrorTextIsBounded pins that text a server
// controls cannot inflate an error string: a huge JSON-RPC error message and
// a huge negotiated version are both cut.
func Test_Protocol_ServerControlledErrorTextIsBounded(t *testing.T) {
	huge := strings.Repeat("x", 200_000)

	t.Run("JSON-RPC error message", func(t *testing.T) {
		fake := newFakeMCPServer(t)
		fake.jsonResponse = true
		fake.methodRawReply = `{"jsonrpc":"2.0","id":{{id}},"error":{"code":1,"message":"` + huge + `"}}`
		_, err := listToolsAgainst(t, fake)
		if err == nil || len(err.Error()) > 2048 {
			t.Fatalf("error text not bounded: %d bytes", len(fmt.Sprint(err)))
		}
	})

	t.Run("negotiated protocol version", func(t *testing.T) {
		fake := newFakeMCPServer(t)
		fake.negotiate = huge
		_, err := listToolsAgainst(t, fake)
		if err == nil || len(err.Error()) > 2048 {
			t.Fatalf("error text not bounded: %d bytes", len(fmt.Sprint(err)))
		}
	})
}

func listToolsBearer(t *testing.T, fake *fakeMCPServer, token string) ([]McpTool, error) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	crypto, err := mcpserverhandler.NewSecretCrypto("1:MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=")
	if err != nil {
		t.Fatalf("could not build crypto: %v", err)
	}
	ct, nonce, ver, err := crypto.Encrypt(token)
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}

	serverID := uuid.Must(uuid.NewV4())
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeBearer,
		SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: ver,
	}, nil)

	h := newTestHandler(t, mockDB)
	h.crypto = crypto
	return h.ListTools(context.Background(), serverID)
}
