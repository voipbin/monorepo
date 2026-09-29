package mcptoolhandler

import (
	"context"
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

	fake := newFakeMCPServer(t)
	fake.callResult = `{"content":[{"type":"text","text":"ok"}]}`
	srv := httptest.NewServer(fake)
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
	for _, r := range fake.recorded() {
		if r.Authorization != "Bearer gho_livetoken" {
			t.Fatalf("request %q carried auth header %q", r.Method, r.Authorization)
		}
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

// Test_ListTools_OAuth_ResolvedOncePerCall pins that a session re-opened
// after a 404 reuses the credential resolved for the call. Resolving an
// OAuth token can spend the stored refresh token, and the row held in memory
// is not updated afterwards, so resolving again would present the spent
// refresh token; a vendor that detects reuse then revokes the grant.
func Test_ListTools_OAuth_ResolvedOncePerCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	fake := newFakeMCPServer(t)
	fake.expireSessionsOnce = true
	srv := httptest.NewServer(fake)
	defer srv.Close()

	serverID := uuid.Must(uuid.NewV4())
	m := &mcpserver.McpServer{
		URL:         srv.URL,
		Status:      mcpserver.StatusActive,
		AuthType:    mcpserver.AuthTypeOAuth,
		OAuthVendor: "linear",
	}
	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(m, nil)
	mockOAuth.EXPECT().GetValidAccessToken(gomock.Any(), m).Return("lin_token", nil).Times(1)

	crypto, err := mcpserverhandler.NewSecretCrypto("")
	if err != nil {
		t.Fatalf("could not create secret crypto: %v", err)
	}
	h := &mcpToolHandler{db: mockDB, crypto: crypto, timeout: 2 * time.Second, oauthHandler: mockOAuth, newClient: testClient}

	if _, err := h.ListTools(context.Background(), serverID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fake.mu.Lock()
	sessions := fake.sessions
	fake.mu.Unlock()
	if sessions != 2 {
		t.Fatalf("expected the 404 to open a second session, got %d sessions", sessions)
	}
	for _, r := range fake.recorded() {
		if r.Authorization != "Bearer lin_token" {
			t.Fatalf("%s %q carried auth header %q", r.HTTPMethod, r.Method, r.Authorization)
		}
	}
}
