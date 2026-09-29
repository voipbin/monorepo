package mcptoolhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// testClient returns a plain (non-SSRF-guarded) client for tests, since
// httptest.Server binds to 127.0.0.1 which the production SSRF guard
// correctly rejects; SSRF guarding itself is covered by
// pkg/mcpserverhandler's own tests.
// testClient is the client unit tests reach local plain-http servers with.
// It keeps the production redirect policy, so no test can pass by following
// a redirect production would refuse; only the dial hook that rejects
// loopback and the https-only check are left out.
func testClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: mcpserverhandler.RefuseRedirects}
}

func newTestHandler(t *testing.T, mockDB dbhandler.DBHandler) *mcpToolHandler {
	t.Helper()
	crypto, err := mcpserverhandler.NewSecretCrypto("")
	if err != nil {
		t.Fatalf("could not create secret crypto: %v", err)
	}
	return &mcpToolHandler{
		db:        mockDB,
		crypto:    crypto,
		timeout:   2 * time.Second,
		newClient: testClient,
	}
}

func Test_ListTools_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	fake := newFakeMCPServer(t)
	fake.tools = `[{"name":"echo","description":"echoes input"}]`
	srv := httptest.NewServer(fake)
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{
		URL:      srv.URL,
		Status:   mcpserver.StatusActive,
		AuthType: mcpserver.AuthTypeNone,
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB)

	tools, err := h.ListTools(context.Background(), serverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func Test_ListTools_Timeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{
		URL:      srv.URL,
		Status:   mcpserver.StatusActive,
		AuthType: mcpserver.AuthTypeNone,
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	crypto, err := mcpserverhandler.NewSecretCrypto("")
	if err != nil {
		t.Fatalf("could not create secret crypto: %v", err)
	}
	h := &mcpToolHandler{db: mockDB, crypto: crypto, timeout: 50 * time.Millisecond, newClient: testClient}

	_, err = h.ListTools(context.Background(), serverID)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

// Test_ListTools_MethodStageFailures pins failures that happen AFTER a
// successful handshake. Each server completes initialize normally and only
// the tools/list answer is bad, and each case asserts the error names that
// specific failure. An earlier version of these tests used raw handlers that
// failed at initialize, so they passed on any error and pinned nothing.
func Test_ListTools_MethodStageFailures(t *testing.T) {
	tests := []struct {
		name      string
		configure func(f *fakeMCPServer)
		wantErr   string
	}{
		{
			name: "non-2xx status keeps the server's reason",
			configure: func(f *fakeMCPServer) {
				f.statusOnMethod = http.StatusInternalServerError
				f.statusBody = "internal error"
			},
			wantErr: "unexpected status code 500, body: internal error",
		},
		{
			name:      "malformed JSON body",
			configure: func(f *fakeMCPServer) { f.methodRawBody = `not json at all {{{` },
			wantErr:   "could not parse JSON-RPC response",
		},
		{
			name: "JSON-RPC error member on the method",
			configure: func(f *fakeMCPServer) {
				f.methodRawReply = `{"jsonrpc":"2.0","id":{{id}},"error":{"code":-32601,"message":"method not found"}}`
			},
			wantErr: "JSON-RPC error (code -32601): method not found",
		},
		{
			name:      "null result",
			configure: func(f *fakeMCPServer) { f.methodRawReply = `{"jsonrpc":"2.0","id":{{id}},"result":null}` },
			wantErr:   "JSON-RPC response has no result",
		},
		{
			// JSON-RPC answers with a null id when it could not read the
			// request id; the reason must still reach the caller.
			name: "error with a null id",
			configure: func(f *fakeMCPServer) {
				f.methodRawReply = `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`
			},
			wantErr: "JSON-RPC error (code -32700): parse error",
		},
		{
			// The body is read through the size cap, so an oversized answer
			// is cut and fails to parse instead of being buffered whole.
			name: "body over the size cap",
			configure: func(f *fakeMCPServer) {
				f.methodRawBody = `{"jsonrpc":"2.0","id":2,"result":{"tools":[],"pad":"` + strings.Repeat("x", mcpserverhandler.McpHTTPResponseSizeCapBytes) + `"}}`
			},
			wantErr: "could not parse JSON-RPC response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeMCPServer(t)
			fake.jsonResponse = true
			tt.configure(fake)

			_, err := listToolsAgainst(t, fake)
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error to contain %q, got: %v", tt.wantErr, err)
			}

			// The handshake completed, so the failure is the method's.
			if got := strings.Join(fake.methods(), ","); !strings.HasPrefix(got, "initialize,notifications/initialized,tools/list") {
				t.Errorf("failure did not happen at the method stage; requests: %s", got)
			}
		})
	}
}

func Test_CallTool_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	fake := newFakeMCPServer(t)
	srv := httptest.NewServer(fake)
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	crypto, err := mcpserverhandler.NewSecretCrypto("1:MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=")
	if err != nil {
		t.Fatalf("could not build crypto: %v", err)
	}
	ct, nonce, ver, err := crypto.Encrypt("my-secret-token")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}

	m := &mcpserver.McpServer{
		URL:              srv.URL,
		Status:           mcpserver.StatusActive,
		AuthType:         mcpserver.AuthTypeBearer,
		SecretCiphertext: ct,
		SecretNonce:      nonce,
		KeyVersion:       ver,
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := &mcpToolHandler{db: mockDB, crypto: crypto, timeout: 2 * time.Second, newClient: testClient}

	result, _, err := h.CallTool(context.Background(), serverID, "echo", `{"input":"x"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello\nworld" {
		t.Fatalf("unexpected result: %q", result)
	}
	// Every request on the session carries the credential, not only the
	// method call: the reference server authenticates initialize too.
	for _, r := range fake.recorded() {
		if r.Authorization != "Bearer my-secret-token" {
			t.Fatalf("request %q carried auth header %q", r.Method, r.Authorization)
		}
	}
	if got := fake.methods(); strings.Join(got, ",") != "initialize,notifications/initialized,tools/call,DELETE" {
		t.Fatalf("unexpected request sequence: %v", got)
	}
}

func Test_CallTool_MalformedJSONRPC(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	fake := newFakeMCPServer(t)
	fake.methodRawBody = `{{{not-json`
	srv := httptest.NewServer(fake)
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeNone}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB)

	_, _, err := h.CallTool(context.Background(), serverID, "echo", `{}`)
	if err == nil || !strings.Contains(err.Error(), "could not parse JSON-RPC response") {
		t.Fatalf("expected a method-stage parse error, got: %v", err)
	}
	if got := strings.Join(fake.methods(), ","); !strings.HasPrefix(got, "initialize,notifications/initialized,tools/call") {
		t.Errorf("failure did not happen at the method stage; requests: %s", got)
	}
}

// Test_TransportRefusesDeletedServer pins the transport-layer backstop.
// McpServerGet returns soft-deleted rows on purpose, so without this check a
// caller reaching the transport directly would open a connection to a server
// the customer has revoked -- using credentials the delete path already zeroed.
//
// The server fixture points at a LIVE httptest server that answers both RPCs
// successfully. That matters: an earlier version of this test pointed at a
// closed port, so it passed on a connection-refused error and kept passing
// with the gate deleted. Here the only reason a call can fail is the gate, and
// the live-row subtests prove the same URL succeeds without it.
//
// reqCount asserts the refusal happens before anything is sent, which a plain
// error check cannot distinguish from a refusal made after the round trip.
func Test_TransportRefusesDeletedServer(t *testing.T) {
	serverID := uuid.Must(uuid.NewV4())
	ts := time.Now()

	// A conformant fake: a live row must complete the whole handshake, so the
	// only reason a deleted row can fail is the gate. The count is of HTTP
	// requests the server actually received: initialize, the notification,
	// the method and the closing DELETE.
	var fake *fakeMCPServer
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		fake.ServeHTTP(w, r)
	}))
	defer srv.Close()

	liveRow := func() *mcpserver.McpServer {
		return &mcpserver.McpServer{
			Identity: commonidentity.Identity{ID: serverID},
			Status:   mcpserver.StatusActive,
			URL:      srv.URL,
		}
	}
	deletedRow := func() *mcpserver.McpServer {
		m := liveRow()
		// Status stays Active: only tm_delete marks the row gone, which is
		// why Status alone is not a sufficient gate here.
		m.TMDelete = &ts
		return m
	}

	tests := []struct {
		name string

		row *mcpserver.McpServer

		wantErr      bool
		wantErrText  string
		wantRequests int
	}{
		{
			// Negative control: without this a gate that refused every row
			// would pass the deleted cases below.
			name:         "live row: ListTools reaches the server",
			row:          liveRow(),
			wantRequests: 4,
		},
		{
			name:         "deleted row: ListTools refuses without sending anything",
			row:          deletedRow(),
			wantErr:      true,
			wantErrText:  "mcp server is deleted",
			wantRequests: 0,
		},
	}

	for _, tt := range tests {
		t.Run("ListTools/"+tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandler(t, mockDB)
			mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(tt.row, nil)

			fake = newFakeMCPServer(t)
			reqCount = 0
			_, err := h.ListTools(context.Background(), serverID)
			assertGate(t, err, tt.wantErr, tt.wantErrText, reqCount, tt.wantRequests)
		})
	}

	callToolTests := []struct {
		name string

		row *mcpserver.McpServer

		wantErr      bool
		wantErrText  string
		wantRequests int
	}{
		{
			name:         "live row: CallTool reaches the server",
			row:          liveRow(),
			wantRequests: 4,
		},
		{
			name:         "deleted row: CallTool refuses without sending anything",
			row:          deletedRow(),
			wantErr:      true,
			wantErrText:  "mcp server is deleted",
			wantRequests: 0,
		},
	}

	for _, tt := range callToolTests {
		t.Run("CallTool/"+tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandler(t, mockDB)
			mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(tt.row, nil)

			fake = newFakeMCPServer(t)
			reqCount = 0
			_, _, err := h.CallTool(context.Background(), serverID, "probe_tool", "{}")
			assertGate(t, err, tt.wantErr, tt.wantErrText, reqCount, tt.wantRequests)
		})
	}
}

// Test_TransportRefusesNilServer covers the nil arm of the same gate. It is
// defensive against the interface rather than the current dbhandler (whose Get
// returns a non-nil row whenever err is nil), but a nil dereference inside the
// transport would panic mid tool call.
func Test_TransportRefusesNilServer(t *testing.T) {
	serverID := uuid.Must(uuid.NewV4())

	t.Run("ListTools", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := newTestHandler(t, mockDB)
		mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(nil, nil)

		_, err := h.ListTools(context.Background(), serverID)
		if err == nil {
			t.Fatal("expected a nil server to be refused")
		}
		if !strings.Contains(err.Error(), "mcp server not found") {
			t.Errorf("expected the not-found refusal, got: %v", err)
		}
	})

	t.Run("CallTool", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := newTestHandler(t, mockDB)
		mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(nil, nil)

		_, _, err := h.CallTool(context.Background(), serverID, "probe_tool", "{}")
		if err == nil {
			t.Fatal("expected a nil server to be refused")
		}
		if !strings.Contains(err.Error(), "mcp server not found") {
			t.Errorf("expected the not-found refusal, got: %v", err)
		}
	})
}

// assertGate checks the outcome AND the reason: asserting only "an error
// happened" is what let the earlier version of this test pass on a
// connection-refused error with the gate deleted.
func assertGate(t *testing.T, err error, wantErr bool, wantErrText string, gotRequests, wantRequests int) {
	t.Helper()

	if wantErr {
		if err == nil {
			t.Fatalf("expected the call to be refused, got nil")
		}
		if !strings.Contains(err.Error(), wantErrText) {
			t.Errorf("expected the refusal to name %q, got: %v", wantErrText, err)
		}
	} else if err != nil {
		t.Fatalf("expected the call to succeed, got: %v", err)
	}

	if gotRequests != wantRequests {
		t.Errorf("expected %d outbound request(s), got %d", wantRequests, gotRequests)
	}
}
