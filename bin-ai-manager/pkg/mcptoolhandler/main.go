// Package mcptoolhandler implements the MCP (Model Context Protocol)
// JSON-RPC client used to discover and call tools on a customer-registered
// McpServer: tools/list and tools/call over Streamable-HTTP, per
// docs/plans/2026-09-11-mcp-tool-integration-design.md §9.1/§9.2.
package mcptoolhandler

//go:generate mockgen -package mcptoolhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

// McpTool is one tool definition returned by a remote MCP server's
// tools/list call.
type McpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

// McpToolHandler discovers and calls tools exposed by a customer's remote
// MCP server. Every outbound call goes through the SSRF-guarded HTTP client
// from pkg/mcpserverhandler (design §7); the server's secret is decrypted
// only at the point of building the outbound request (design §6).
type McpToolHandler interface {
	// ListTools sends an MCP tools/list request to the server identified by
	// serverID and returns its advertised tools.
	ListTools(ctx context.Context, serverID uuid.UUID) ([]McpTool, error)

	// CallTool sends an MCP tools/call request for toolName on the server
	// identified by serverID, with argumentsJSON passed through unchanged as
	// the JSON-RPC arguments object, and returns the joined text content of
	// the result.
	CallTool(ctx context.Context, serverID uuid.UUID, toolName string, argumentsJSON string) (string, error)
}

type mcpToolHandler struct {
	db      dbhandler.DBHandler
	crypto  *mcpserverhandler.SecretCrypto
	timeout time.Duration

	// oauthHandler resolves a valid (transparently refreshed if needed)
	// OAuth access token for AuthTypeOAuth servers (design
	// docs/plans/2026-09-12-mcp-server-oauth-support-design.md §8). Nil
	// is tolerated (buildAuthHeader fails closed with an error for
	// AuthTypeOAuth) so existing call sites that don't yet pass one
	// don't break -- production wiring always sets it.
	oauthHandler mcpoauthhandler.McpOAuthHandler

	// newClient builds the http.Client used for every outbound tools/list
	// and tools/call request. Defaults to
	// mcpserverhandler.NewSSRFGuardedClient in production
	// (NewMcpToolHandler); overridden in unit tests only, so tests can
	// point at an httptest.Server (which binds to 127.0.0.1, a loopback
	// address the SSRF guard correctly rejects in real traffic) without
	// weakening the guard used at runtime.
	newClient func(timeout time.Duration) *http.Client
}

// NewMcpToolHandler creates a new McpToolHandler. cryptoKeys is the raw
// MCP_SECRET_ENCRYPTION_KEYS config value; timeoutSeconds is
// mcp_tool_call_timeout_seconds (design §13), applied to every outbound
// tools/list and tools/call request. oauthHandler resolves valid access
// tokens for AuthTypeOAuth servers (design §8); pass nil only in tests
// that don't exercise OAuth servers.
func NewMcpToolHandler(db dbhandler.DBHandler, cryptoKeys string, timeoutSeconds int, oauthHandler mcpoauthhandler.McpOAuthHandler) (McpToolHandler, error) {
	crypto, err := mcpserverhandler.NewSecretCrypto(cryptoKeys)
	if err != nil {
		return nil, err
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}

	return &mcpToolHandler{
		db:           db,
		crypto:       crypto,
		timeout:      time.Duration(timeoutSeconds) * time.Second,
		oauthHandler: oauthHandler,
		newClient:    mcpserverhandler.NewSSRFGuardedClient,
	}, nil
}
