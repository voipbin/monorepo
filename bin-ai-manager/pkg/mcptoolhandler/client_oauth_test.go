package mcptoolhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

// Test_CallTool_OAuth_Success verifies buildAuthHeader's AuthTypeOAuth
// case (design docs/plans/2026-09-12-mcp-server-oauth-support-design.md
// §8): it must delegate to McpOAuthHandler.GetValidAccessToken and send
// the returned token as a Bearer Authorization header, same as
// AuthTypeBearer -- proving the OAuth "connect" flow's stored token is
// actually usable on a real outbound MCP call, not just persisted.
func Test_CallTool_OAuth_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]}}`))
	}))
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{
		URL:         srv.URL,
		Status:      mcpserver.StatusActive,
		AuthType:    mcpserver.AuthTypeOAuth,
		OAuthVendor: "github",
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)
	mockOAuth.EXPECT().GetValidAccessToken(gomock.Any(), m).Return("gho_livetoken", nil)

	crypto, err := mcpserverhandler.NewSecretCrypto("")
	if err != nil {
		t.Fatalf("could not create secret crypto: %v", err)
	}
	h := &mcpToolHandler{db: mockDB, crypto: crypto, timeout: 2 * time.Second, oauthHandler: mockOAuth, newClient: testClient}

	result, err := h.CallTool(context.Background(), serverID, "echo", `{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("unexpected result: %q", result)
	}
	if gotAuth != "Bearer gho_livetoken" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
}

// Test_CallTool_OAuth_RefreshFailurePropagates verifies that a
// GetValidAccessToken failure (e.g. expired token, no refresh token
// available, or a failed refresh call) surfaces as a normal CallTool
// error -- design §8 step 4's "no new error class" requirement.
func Test_CallTool_OAuth_RefreshFailurePropagates(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{
		URL:         "https://example.invalid/mcp",
		Status:      mcpserver.StatusActive,
		AuthType:    mcpserver.AuthTypeOAuth,
		OAuthVendor: "github",
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)
	mockOAuth.EXPECT().GetValidAccessToken(gomock.Any(), m).Return("", assertErr("oauth access token expired and no refresh token is available; reconnect required"))

	crypto, err := mcpserverhandler.NewSecretCrypto("")
	if err != nil {
		t.Fatalf("could not create secret crypto: %v", err)
	}
	h := &mcpToolHandler{db: mockDB, crypto: crypto, timeout: 2 * time.Second, oauthHandler: mockOAuth, newClient: testClient}

	_, err = h.CallTool(context.Background(), serverID, "echo", `{}`)
	if err == nil {
		t.Fatalf("expected error when oauth token cannot be resolved")
	}
}

// Test_CallTool_OAuth_NoHandlerConfigured verifies buildAuthHeader fails
// closed (rather than panicking on a nil oauthHandler) if an
// AuthTypeOAuth server is somehow reached without one configured.
func Test_CallTool_OAuth_NoHandlerConfigured(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{
		URL:         "https://example.invalid/mcp",
		Status:      mcpserver.StatusActive,
		AuthType:    mcpserver.AuthTypeOAuth,
		OAuthVendor: "github",
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)

	h := newTestHandler(t, mockDB) // oauthHandler left nil

	_, err := h.CallTool(context.Background(), serverID, "echo", `{}`)
	if err == nil {
		t.Fatalf("expected error when oauthHandler is nil")
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
