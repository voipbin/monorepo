package mcptoolhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

const coverageTestKeys = "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func newCoverageCrypto(t *testing.T) *mcpserverhandler.SecretCrypto {
	t.Helper()
	c, err := mcpserverhandler.NewSecretCrypto(coverageTestKeys)
	if err != nil {
		t.Fatalf("could not build crypto: %v", err)
	}
	return c
}

func Test_buildAuthHeader_branches(t *testing.T) {
	crypto := newCoverageCrypto(t)
	ct, nonce, version, err := crypto.Encrypt("fake-secret-value")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		server *mcpserver.McpServer

		expectName  string
		expectValue string
		expectMsg   string
	}{
		{
			name:   "no auth sends no header",
			server: &mcpserver.McpServer{AuthType: mcpserver.AuthTypeNone},
		},
		{
			name:        "bearer",
			server:      &mcpserver.McpServer{AuthType: mcpserver.AuthTypeBearer, SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: version},
			expectName:  "Authorization",
			expectValue: "Bearer fake-secret-value",
		},
		{
			name:      "bearer with unknown key version",
			server:    &mcpserver.McpServer{AuthType: mcpserver.AuthTypeBearer, SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: 7},
			expectMsg: "could not decrypt secret",
		},
		{
			name:        "api key with custom header",
			server:      &mcpserver.McpServer{AuthType: mcpserver.AuthTypeAPIKey, APIKeyHeader: "X-Custom-Key", SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: version},
			expectName:  "X-Custom-Key",
			expectValue: "fake-secret-value",
		},
		{
			name:        "api key falls back to the default header name",
			server:      &mcpserver.McpServer{AuthType: mcpserver.AuthTypeAPIKey, SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: version},
			expectName:  "X-API-Key",
			expectValue: "fake-secret-value",
		},
		{
			name:      "api key with unknown key version",
			server:    &mcpserver.McpServer{AuthType: mcpserver.AuthTypeAPIKey, SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: 7},
			expectMsg: "could not decrypt secret",
		},
		{
			name:      "unsupported auth type",
			server:    &mcpserver.McpServer{AuthType: mcpserver.AuthType("basic")},
			expectMsg: `unsupported auth type: "basic"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &mcpToolHandler{crypto: crypto}

			name, value, err := h.buildAuthHeader(context.Background(), tt.server)
			if tt.expectMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
					t.Fatalf("error = %v, expect to contain %q", err, tt.expectMsg)
				}
				if name != "" || value != "" {
					t.Errorf("expected empty header on error, got %q=%q", name, value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.expectName || value != tt.expectValue {
				t.Errorf("header = %q=%q, expect: %q=%q", name, value, tt.expectName, tt.expectValue)
			}
		})
	}
}

func Test_ListTools_apiKeyHeaderIsSent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	crypto := newCoverageCrypto(t)
	ct, nonce, version, err := crypto.Encrypt("fake-secret-value")
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	seen := []string{}
	fake := newFakeMCPServer(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("X-Custom-Key")+"|"+r.Header.Get("Authorization"))
		mu.Unlock()
		fake.ServeHTTP(w, r)
	}))
	defer srv.Close()

	serverID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		URL: srv.URL, Status: mcpserver.StatusActive, AuthType: mcpserver.AuthTypeAPIKey, APIKeyHeader: "X-Custom-Key",
		SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: version,
	}, nil)

	h := newTestHandler(t, mockDB)
	h.crypto = crypto

	tools, err := h.ListTools(context.Background(), serverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "lookup_order" {
		t.Errorf("unexpected tools: %+v", tools)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) == 0 {
		t.Fatal("no requests reached the server")
	}
	for _, s := range seen {
		if s != "fake-secret-value|" {
			t.Errorf("request carried %q, expect the api key header only", s)
		}
	}
}

func Test_ListTools_errors(t *testing.T) {
	errBoom := errors.New("boom")
	serverID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		dbErr     error
		configure func(f *fakeMCPServer)

		expectMsg string
	}{
		{name: "db lookup fails", dbErr: errBoom, expectMsg: "mcptoolhandler.ListTools: could not get mcp server"},
		{
			name:      "result is not an object",
			configure: func(f *fakeMCPServer) { f.methodRawReply = `{"jsonrpc":"2.0","id":{{id}},"result":[1,2]}` },
			expectMsg: "could not parse tools/list result",
		},
		{
			name:      "tools member is not an array",
			configure: func(f *fakeMCPServer) { f.methodRawReply = `{"jsonrpc":"2.0","id":{{id}},"result":{"tools":"x"}}` },
			expectMsg: "could not parse tools/list result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			fake := newFakeMCPServer(t)
			if tt.configure != nil {
				tt.configure(fake)
			}
			srv := httptest.NewServer(fake)
			defer srv.Close()

			if tt.dbErr != nil {
				mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(nil, tt.dbErr)
			} else {
				mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive}, nil)
			}

			h := newTestHandler(t, mockDB)
			tools, err := h.ListTools(context.Background(), serverID)
			if tools != nil {
				t.Errorf("expected nil tools, got %v", tools)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
			if tt.dbErr != nil && !errors.Is(err, tt.dbErr) {
				t.Errorf("db error is not wrapped: %v", err)
			}
		})
	}
}

func Test_CallTool_errorsAndResults(t *testing.T) {
	errBoom := errors.New("boom")
	serverID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name string
		args string

		dbErr     error
		configure func(f *fakeMCPServer)
		noServer  bool // the server must not be contacted at all

		expectMsg       string
		expectText      string
		expectIsError   bool
		expectParamsHas string
	}{
		{name: "db lookup fails", args: `{}`, dbErr: errBoom, noServer: true, expectMsg: "mcptoolhandler.CallTool: could not get mcp server"},
		{name: "arguments are not json", args: `{not json`, noServer: true, expectMsg: "could not parse arguments JSON"},
		{
			name:      "server answers with an http error",
			args:      `{}`,
			configure: func(f *fakeMCPServer) { f.statusOnMethod = http.StatusBadGateway; f.statusBody = "upstream down" },
			expectMsg: "unexpected status code 502",
		},
		{
			name:      "result does not match the call result shape",
			args:      `{}`,
			configure: func(f *fakeMCPServer) { f.callResult = `"just a string"` },
			expectMsg: "could not parse tools/call result",
		},
		{
			name: "blank arguments are sent as an empty object and empty texts are dropped",
			args: "  ",
			configure: func(f *fakeMCPServer) {
				f.callResult = `{"content":[{"type":"text","text":"a"},{"type":"image"},{"type":"text","text":""},{"type":"text","text":"b"}],"isError":true}`
			},
			expectText:      "a\nb",
			expectIsError:   true,
			expectParamsHas: `"arguments":{}`,
		},
		{
			name:            "arguments are passed through",
			args:            `{"order_id":"o-1","n":2}`,
			configure:       func(f *fakeMCPServer) { f.callResult = `{"content":[]}` },
			expectText:      "",
			expectParamsHas: `"order_id":"o-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			fake := newFakeMCPServer(t)
			if tt.configure != nil {
				tt.configure(fake)
			}
			srv := httptest.NewServer(fake)
			defer srv.Close()

			if tt.dbErr != nil {
				mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(nil, tt.dbErr)
			} else {
				mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{URL: srv.URL, Status: mcpserver.StatusActive}, nil)
			}

			h := newTestHandler(t, mockDB)
			text, isError, err := h.CallTool(context.Background(), serverID, "lookup_order", tt.args)

			if tt.noServer && len(fake.recorded()) != 0 {
				t.Errorf("server must not be contacted, got %v", fake.methods())
			}

			if tt.expectMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
					t.Fatalf("error = %v, expect to contain %q", err, tt.expectMsg)
				}
				if text != "" || isError {
					t.Errorf("expected zero results on error, got %q %v", text, isError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if text != tt.expectText || isError != tt.expectIsError {
				t.Errorf("result = %q/%v, expect: %q/%v", text, isError, tt.expectText, tt.expectIsError)
			}
			for _, r := range fake.recorded() {
				if r.Method == "tools/call" && !strings.Contains(string(r.Params), tt.expectParamsHas) {
					t.Errorf("params %s do not contain %s", r.Params, tt.expectParamsHas)
				}
			}
		})
	}
}

func Test_doJSONRPCRequest_defaultClientRefusesPlainHTTP(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var hits int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
	}))
	defer srv.Close()

	h := &mcpToolHandler{db: dbhandler.NewMockDBHandler(mc), crypto: newCoverageCrypto(t), timeout: 2 * time.Second}

	res, err := h.doJSONRPCRequest(context.Background(), &mcpserver.McpServer{URL: srv.URL}, "tools/list", map[string]any{})
	if res != nil {
		t.Errorf("expected nil result, got %s", res)
	}
	if err == nil || !strings.Contains(err.Error(), "https only") {
		t.Errorf("error = %v, expect the https-only refusal", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if hits != 0 {
		t.Errorf("a plain http request reached the server %d times", hits)
	}
}

func Test_doJSONRPCRequest_authHeaderFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &mcpToolHandler{db: dbhandler.NewMockDBHandler(mc), crypto: newCoverageCrypto(t), timeout: time.Second, newClient: testClient}

	res, err := h.doJSONRPCRequest(context.Background(), &mcpserver.McpServer{URL: "https://mcp.example.com/", AuthType: mcpserver.AuthType("basic")}, "tools/list", nil)
	if res != nil {
		t.Errorf("expected nil result, got %s", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not build auth header") {
		t.Errorf("error = %v", err)
	}
}

func Test_doJSONRPCRequest_unusableServerURL(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &mcpToolHandler{db: dbhandler.NewMockDBHandler(mc), crypto: newCoverageCrypto(t), timeout: time.Second, newClient: testClient}

	// a host that cannot be parsed fails when the request is built, before any network use
	res, err := h.doJSONRPCRequest(context.Background(), &mcpserver.McpServer{URL: "http://[::1"}, "tools/list", nil)
	if res != nil {
		t.Errorf("expected nil result, got %s", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not build request") {
		t.Errorf("error = %v", err)
	}
}

// scriptedMCP answers by JSON-RPC method with hand-written replies, for
// handshake-stage failures the shared fake cannot produce.
type scriptedMCP struct {
	mu       sync.Mutex
	inits    int
	onInit   func(n int, w http.ResponseWriter, id json.RawMessage)
	onNotify func(w http.ResponseWriter)
	calls    []string
}

func (s *scriptedMCP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		s.mu.Lock()
		s.calls = append(s.calls, "DELETE")
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}

	var msg struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	_ = json.NewDecoder(r.Body).Decode(&msg)

	s.mu.Lock()
	s.calls = append(s.calls, msg.Method)
	n := 0
	if msg.Method == "initialize" {
		s.inits++
		n = s.inits
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch msg.Method {
	case "initialize":
		w.Header().Set("Mcp-Session-Id", fmt.Sprintf("session-%d", n))
		s.onInit(n, w, msg.ID)
	case "notifications/initialized":
		if s.onNotify != nil {
			s.onNotify(w)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	default:
		// every method call finds the session gone
		http.Error(w, "Not Found: Session not found", http.StatusNotFound)
	}
}

func (s *scriptedMCP) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

func goodInit(w http.ResponseWriter, id json.RawMessage) {
	_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-11-25","capabilities":{},"serverInfo":{"name":"x","version":"1"}}}`, id)
}

func runScripted(t *testing.T, s *scriptedMCP) (json.RawMessage, error) {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)

	h := &mcpToolHandler{db: dbhandler.NewMockDBHandler(mc), crypto: newCoverageCrypto(t), timeout: 3 * time.Second, newClient: testClient}
	return h.doJSONRPCRequest(context.Background(), &mcpserver.McpServer{URL: srv.URL}, "tools/list", map[string]any{})
}

func Test_doJSONRPCRequest_reinitializeFails(t *testing.T) {
	s := &scriptedMCP{onInit: func(n int, w http.ResponseWriter, id json.RawMessage) {
		if n == 1 {
			goodInit(w, id)
			return
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}}

	res, err := runScripted(t, s)
	if res != nil {
		t.Errorf("expected nil result, got %s", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not re-initialize after the session expired") {
		t.Errorf("error = %v", err)
	}
}

func Test_openSession_initializeResultNotAnObject(t *testing.T) {
	s := &scriptedMCP{onInit: func(n int, w http.ResponseWriter, id json.RawMessage) {
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"nope"}`, id)
	}}

	_, err := runScripted(t, s)
	if err == nil || !strings.Contains(err.Error(), "could not parse initialize result") {
		t.Fatalf("error = %v", err)
	}
	// the session the server opened is closed again
	if calls := s.snapshot(); len(calls) != 2 || calls[0] != "initialize" || calls[1] != "DELETE" {
		t.Errorf("calls = %v, expect [initialize DELETE]", calls)
	}
}

func Test_openSession_initializedNotificationRejected(t *testing.T) {
	s := &scriptedMCP{
		onInit:   func(n int, w http.ResponseWriter, id json.RawMessage) { goodInit(w, id) },
		onNotify: func(w http.ResponseWriter) { http.Error(w, "nope", http.StatusInternalServerError) },
	}

	_, err := runScripted(t, s)
	if err == nil || !strings.Contains(err.Error(), "initialized notification failed") {
		t.Fatalf("error = %v", err)
	}
	if calls := s.snapshot(); len(calls) != 3 || calls[2] != "DELETE" {
		t.Errorf("calls = %v, expect the session to be closed after the failed notification", calls)
	}
}

func Test_decodeToolsList_malformedInputs(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty input", in: ``},
		{name: "invalid token where a key is expected", in: `{x`},
		{name: "input ends while skipping a member", in: `{"a":`},
		{name: "input ends inside a skipped array", in: `{"a":[1`},
		{name: "input ends right after the tools array", in: `{"tools":[{"name":"a"}`},
		{name: "input ends right after tools key", in: `{"tools"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools, truncated, err := decodeToolsList(json.RawMessage(tt.in), 5)
			if err == nil {
				t.Fatalf("expected an error, got %v", tools)
			}
			if tools != nil || truncated {
				t.Errorf("expected zero results with the error, got %v %v", tools, truncated)
			}
		})
	}
}

func Test_expectDelim(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want json.Delim

		expectMsg string
	}{
		{name: "matching object start", in: `{`, want: '{'},
		{name: "array where object expected", in: `[`, want: '{', expectMsg: `expected "{", got [`},
		{name: "scalar where delimiter expected", in: `5`, want: '[', expectMsg: `expected "[", got 5`},
		{name: "empty input", in: ``, want: '{', expectMsg: "EOF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := json.NewDecoder(strings.NewReader(tt.in))
			err := expectDelim(dec, tt.want)
			if tt.expectMsg == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
		})
	}
}

func Test_NewMcpToolHandler(t *testing.T) {
	tests := []struct {
		name string

		keys           string
		timeoutSeconds int

		expectErr     bool
		expectTimeout time.Duration
	}{
		{name: "explicit timeout", keys: coverageTestKeys, timeoutSeconds: 7, expectTimeout: 7 * time.Second},
		{name: "zero timeout falls back to ten seconds", keys: "", timeoutSeconds: 0, expectTimeout: 10 * time.Second},
		{name: "negative timeout falls back to ten seconds", keys: "", timeoutSeconds: -3, expectTimeout: 10 * time.Second},
		{name: "malformed keys are refused", keys: "bad", timeoutSeconds: 5, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res, err := NewMcpToolHandler(dbhandler.NewMockDBHandler(mc), tt.keys, tt.timeoutSeconds, nil)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("expected error and nil handler, got %v / %v", res, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			impl, ok := res.(*mcpToolHandler)
			if !ok {
				t.Fatalf("unexpected type %T", res)
			}
			if impl.timeout != tt.expectTimeout || impl.crypto == nil || impl.newClient == nil || impl.oauthHandler != nil {
				t.Errorf("handler wiring wrong: timeout=%v crypto=%v newClient=%v oauth=%v", impl.timeout, impl.crypto, impl.newClient != nil, impl.oauthHandler)
			}
		})
	}
}
