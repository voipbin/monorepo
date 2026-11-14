package mcpoauthhandler

import (
	"context"
	"time"

	"github.com/pkg/errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// vendorMeta is one entry of the hand-coded, static vendor catalog
// (design §6) -- deliberately not DB-backed. It never resolves anything.
type vendorMeta struct {
	MCPServerURL string
	AuthorizeURL string
	TokenURL     string
	Scopes       []string
}

// Credentials is the OAuth app of a vendor: one (client_id, client_secret)
// pair. It is the result of one resolution and is fixed for the call that
// resolved it.
type Credentials struct {
	ClientID     string
	ClientSecret string
}

// CredentialsFunc resolves the Credentials of a vendor at the time of use.
// An empty pair with no error means "not configured": Start and Complete
// fail closed on it (MCP_OAUTH_VENDOR_NOT_CONFIGURED), the refresh exchange
// does not. An error is a failed read or a pair that is stored incompletely;
// it must never be answered with a value from another source.
type CredentialsFunc func(ctx context.Context, vendor string) (Credentials, error)

// credentialResolveTimeout bounds the resolution made by the detached refresh
// exchange, which does not run inside the deadline of its caller.
const credentialResolveTimeout = 2 * time.Second

// vendorConfig is the resolved view one request works with: the static
// metadata plus the credentials resolved for that request.
type vendorConfig struct {
	vendorMeta
	Credentials
}

// configured reports whether the resolved app can be used: both values are present.
func (vc vendorConfig) configured() bool {
	return vc.ClientID != "" && vc.ClientSecret != ""
}

// notConfiguredError is the error of a vendor whose app is not stored. It is a platform side
// condition (the customer can not fix it) and carries no value.
func notConfiguredError() error {
	return cerrors.Internal(commonoutline.ServiceNameAIManager, "MCP_OAUTH_VENDOR_NOT_CONFIGURED", "the oauth app of the vendor is not configured")
}

// errUnknownVendor is returned for a vendor that is not in the catalog.
var errUnknownVendor = errors.New("unknown oauth vendor")

// Vendor identifiers -- the exact strings stored in
// McpServer.OAuthVendor / ai_mcp_oauth_states.vendor.
const (
	VendorGitHub  = "github"
	VendorLinear  = "linear"
	VendorAsana   = "asana"
	VendorHubSpot = "hubspot"
)

// newVendorCatalog builds the static vendor map. The credentials of a vendor
// are not part of it: they are resolved per request (see CredentialsFunc).
func newVendorCatalog() map[string]vendorMeta {
	return map[string]vendorMeta{
		VendorGitHub: {
			MCPServerURL: "https://api.githubcopilot.com/mcp/",
			AuthorizeURL: "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			// Narrowest scope covering the GitHub MCP server's default
			// toolset; revisit if customers hit permission errors (design §6).
			Scopes: []string{"repo", "read:org"},
		},
		VendorLinear: {
			MCPServerURL: "https://mcp.linear.app/mcp",
			AuthorizeURL: "https://linear.app/oauth/authorize",
			TokenURL:     "https://api.linear.app/oauth/token",
			Scopes:       []string{"read", "write"},
		},
		VendorAsana: {
			MCPServerURL: "https://mcp.asana.com/v2/mcp",
			AuthorizeURL: "https://app.asana.com/-/oauth_authorize",
			TokenURL:     "https://app.asana.com/-/oauth_token",
			// No scopes: Asana rejects an explicit scope for MCP apps
			// ("Invalid scope(s) requested"), so the scope parameter is omitted.
		},
		// HubSpot's connector scope is fixed when the connector is installed
		// and its metadata lists no scopes, so the authorize request carries
		// none. The token endpoint accepts client_secret_post only.
		VendorHubSpot: {
			MCPServerURL: "https://mcp.hubspot.com",
			AuthorizeURL: "https://mcp.hubspot.com/oauth/authorize/user",
			TokenURL:     "https://mcp.hubspot.com/oauth/v3/token",
		},
	}
}

// isValidVendor reports whether id names a vendor in the catalog.
func (h *mcpOAuthHandler) isValidVendor(id string) bool {
	_, ok := h.vendors[id]
	return ok
}

// resolveVendor returns the resolved view of a vendor: its static metadata
// and the credentials resolved now. The caller keeps the result in a local
// variable for the whole request, so one request resolves once.
func (h *mcpOAuthHandler) resolveVendor(ctx context.Context, id string) (vendorConfig, error) {
	meta, ok := h.vendors[id]
	if !ok {
		return vendorConfig{}, errUnknownVendor
	}

	creds, err := h.credentials(ctx, id)
	if err != nil {
		return vendorConfig{}, errors.Wrap(err, "could not resolve the oauth app of the vendor")
	}

	return vendorConfig{vendorMeta: meta, Credentials: creds}, nil
}
