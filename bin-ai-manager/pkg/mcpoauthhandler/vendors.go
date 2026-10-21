package mcpoauthhandler

// vendorConfig is one entry of the hand-coded, static vendor catalog
// (design §6) -- deliberately not DB-backed.
type vendorConfig struct {
	MCPServerURL string
	AuthorizeURL string
	TokenURL     string
	Scopes       []string
	ClientID     string
	ClientSecret string
}

// Vendor identifiers -- the exact strings stored in
// McpServer.OAuthVendor / ai_mcp_oauth_states.vendor.
const (
	VendorGitHub = "github"
	VendorLinear = "linear"
	VendorAsana  = "asana"
)

// newVendorCatalog builds the vendor map with client credentials injected
// from config (design §6's MCP_OAUTH_<VENDOR>_CLIENT_ID/SECRET env vars).
func newVendorCatalog(githubClientID, githubClientSecret, linearClientID, linearClientSecret, asanaClientID, asanaClientSecret string) map[string]vendorConfig {
	return map[string]vendorConfig{
		VendorGitHub: {
			MCPServerURL: "https://api.githubcopilot.com/mcp/",
			AuthorizeURL: "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			// Narrowest scope covering the GitHub MCP server's default
			// toolset; revisit if customers hit permission errors (design §6).
			Scopes:       []string{"repo", "read:org"},
			ClientID:     githubClientID,
			ClientSecret: githubClientSecret,
		},
		VendorLinear: {
			MCPServerURL: "https://mcp.linear.app/mcp",
			AuthorizeURL: "https://linear.app/oauth/authorize",
			TokenURL:     "https://api.linear.app/oauth/token",
			Scopes:       []string{"read", "write"},
			ClientID:     linearClientID,
			ClientSecret: linearClientSecret,
		},
		VendorAsana: {
			MCPServerURL: "https://mcp.asana.com/v2/mcp",
			AuthorizeURL: "https://app.asana.com/-/oauth_authorize",
			TokenURL:     "https://app.asana.com/-/oauth_token",
			// No scopes: Asana rejects an explicit scope for MCP apps
			// ("Invalid scope(s) requested"), so the scope parameter is omitted.
			ClientID:     asanaClientID,
			ClientSecret: asanaClientSecret,
		},
	}
}

// isValidVendor reports whether id names a vendor in the catalog.
func (h *mcpOAuthHandler) isValidVendor(id string) bool {
	_, ok := h.vendors[id]
	return ok
}
