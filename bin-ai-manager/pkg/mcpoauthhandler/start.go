package mcpoauthhandler

import (
	"context"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// oauthRedirectURI is the single fixed redirect_uri registered with both
// vendor apps (design §3, §5): "https://api.voipbin.net/mcpservers/oauth/callback".
// Not configurable per-request -- both GitHub and Linear OAuth apps are
// registered with exactly this URI, and using anything else would be
// rejected by the vendor's own redirect_uri validation.
const oauthRedirectURI = "https://api.voipbin.net/mcpservers/oauth/callback"

// Start implements McpOAuthHandler.Start (design §7 step 2, §9).
func (h *mcpOAuthHandler) Start(ctx context.Context, customerID uuid.UUID, vendor string, mcpServerID *uuid.UUID) (string, string, error) {
	if !h.isValidVendor(vendor) {
		return "", "", cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_OAUTH_VENDOR", "unknown oauth vendor: "+vendor)
	}
	vc := h.vendors[vendor]

	// Reconnect ownership check (design §9, Round 1 IDOR fix): a bare
	// mcp_server_id must be verified to belong to the requesting
	// customer BEFORE any state row is created, otherwise any customer
	// could hijack another customer's MCP server row's OAuth
	// connection on callback.
	if mcpServerID != nil {
		existing, err := h.db.McpServerGet(ctx, *mcpServerID)
		if err != nil {
			if errors.Is(err, dbhandler.ErrNotFound) {
				return "", "", cerrors.NotFound(
					commonoutline.ServiceNameAIManager,
					"MCP_SERVER_NOT_FOUND",
					"The MCP server was not found.",
				).Wrap(err)
			}
			return "", "", errors.Wrapf(err, "could not get mcp server for reconnect")
		}
		if existing.CustomerID != customerID {
			// Do not leak "found but not yours" -- same NotFound the
			// customer would see for a nonexistent id (design §9).
			return "", "", cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"MCP_SERVER_NOT_FOUND",
				"The MCP server was not found.",
			)
		}
	}

	state, err := generateRandomToken(32)
	if err != nil {
		return "", "", errors.Wrap(err, "could not generate oauth state")
	}

	verifier, challenge, err := generatePKCE()
	if err != nil {
		return "", "", errors.Wrap(err, "could not generate pkce pair")
	}

	now := h.utilHandler.TimeNow()
	expire := h.utilHandler.TimeNowAdd(stateTTL)

	row := &mcpoauthstate.McpOAuthState{
		State:        state,
		CustomerID:   customerID,
		McpServerID:  mcpServerID,
		Vendor:       vendor,
		PKCEVerifier: verifier,
		TMCreate:     now,
		TMExpire:     expire,
	}
	if err := h.db.McpOAuthStateCreate(ctx, row); err != nil {
		return "", "", errors.Wrap(err, "could not create oauth state")
	}

	authorizeURL := buildAuthorizeURL(vc, state, challenge)

	return authorizeURL, state, nil
}

// buildAuthorizeURL constructs the vendor authorization endpoint URL with
// every parameter every flow needs (design §7b: code_challenge is sent
// unconditionally, vendor-uniform, harmless for vendors that don't require it).
func buildAuthorizeURL(vc vendorConfig, state string, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", vc.ClientID)
	q.Set("redirect_uri", oauthRedirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	if len(vc.Scopes) > 0 {
		scope := vc.Scopes[0]
		for _, s := range vc.Scopes[1:] {
			scope += " " + s
		}
		q.Set("scope", scope)
	}

	return vc.AuthorizeURL + "?" + q.Encode()
}
