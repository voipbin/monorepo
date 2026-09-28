package mcptoolhandler

import (
	"context"
	"encoding/json"
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
func testClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"echo","description":"echoes input"}]}}`))
	}))
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

func Test_ListTools_NonSuccessStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal error`))
	}))
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeNone}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB)

	_, err := h.ListTools(context.Background(), serverID)
	if err == nil {
		t.Fatalf("expected error for non-2xx status")
	}
}

func Test_ListTools_MalformedJSONRPC(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json at all {{{`))
	}))
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeNone}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB)

	_, err := h.ListTools(context.Background(), serverID)
	if err == nil {
		t.Fatalf("expected error for malformed JSON-RPC body")
	}
}

func Test_ListTools_JSONRPCErrorObject(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`))
	}))
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeNone}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB)

	_, err := h.ListTools(context.Background(), serverID)
	if err == nil {
		t.Fatalf("expected error for JSON-RPC error object")
	}
}

func Test_CallTool_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "tools/call" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"hello"},{"type":"text","text":"world"}]}}`))
	}))
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

	result, err := h.CallTool(context.Background(), serverID, "echo", `{"input":"x"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello\nworld" {
		t.Fatalf("unexpected result: %q", result)
	}
	if gotAuth != "Bearer my-secret-token" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
}

func Test_CallTool_MalformedJSONRPC(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{{{not-json`))
	}))
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeNone}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB)

	_, err := h.CallTool(context.Background(), serverID, "echo", `{}`)
	if err == nil {
		t.Fatalf("expected error for malformed JSON-RPC body")
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

	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		result := `{"tools":[{"name":"probe_tool"}]}`
		if req.Method == "tools/call" {
			result = `{"content":[{"type":"text","text":"ok"}]}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":` + result + `}`))
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
			wantRequests: 1,
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
			wantRequests: 1,
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

			reqCount = 0
			_, err := h.CallTool(context.Background(), serverID, "probe_tool", "{}")
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

		_, err := h.CallTool(context.Background(), serverID, "probe_tool", "{}")
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
