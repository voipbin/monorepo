package mcptoolhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
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
