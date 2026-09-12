// Package mcpoauthhandler implements the vendor-fixed OAuth 2.1 flow used
// to connect a customer-registered McpServer to GitHub or Linear without
// the customer ever handling a client_id/client_secret. See
// docs/plans/2026-09-12-mcp-server-oauth-support-design.md.
package mcpoauthhandler

//go:generate mockgen -package mcpoauthhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// McpOAuthHandler drives the vendor-fixed OAuth 2.1 authorization-code +
// PKCE flow backing AIV1McpOAuthStart/Callback/Complete (design §7).
type McpOAuthHandler interface {
	// Start generates a state+PKCE pair, persists it, and returns the
	// vendor's authorize_url plus the link_token (== state, design §7a
	// Layer 2). When mcpServerID is non-nil, verifies customerID owns
	// that row (design §9's IDOR fix) before persisting the state.
	Start(ctx context.Context, customerID uuid.UUID, vendor string, mcpServerID *uuid.UUID) (authorizeURL string, linkToken string, err error)

	// CallbackExists reports whether a still-unexpired state row exists
	// for the given state token. This backs the PUBLIC callback's
	// thin, no-mutation existence check (design §7a Layer 1, §10) --
	// it never deletes the row or exchanges the code.
	CallbackExists(ctx context.Context, state string) (bool, error)

	// Complete verifies the state row's CustomerID matches customerID
	// (design §7a's real security boundary), deletes the state row,
	// exchanges code for tokens, encrypts them, and creates/updates the
	// McpServer row (design §7 steps 9-12).
	Complete(ctx context.Context, customerID uuid.UUID, state string, code string) (*mcpserver.McpServer, error)

	// GetValidAccessToken returns a decrypted, currently-valid OAuth
	// access token for m, transparently refreshing it first via the
	// vendor's token endpoint if it has expired (or is about to) and a
	// refresh token is available (design §8). Used by
	// mcptoolhandler.buildAuthHeader's AuthTypeOAuth case on every
	// outbound MCP tools/list and tools/call request.
	GetValidAccessToken(ctx context.Context, m *mcpserver.McpServer) (string, error)
}

// stateTTL is the lifetime of an ai_mcp_oauth_states row (design §5, §7a
// Layer 1) -- matches typical OAuth authorization-code lifetimes.
const stateTTL = 10 * time.Minute

type mcpOAuthHandler struct {
	utilHandler utilhandler.UtilHandler
	db          dbhandler.DBHandler
	crypto      *mcpserverhandler.SecretCrypto
	vendors     map[string]vendorConfig

	// httpClient performs the outbound vendor token-endpoint POST in
	// Complete. Overridable in unit tests only.
	httpClient *http.Client
}

// NewMcpOAuthHandler creates a new McpOAuthHandler. cryptoKeys is the raw
// MCP_SECRET_ENCRYPTION_KEYS config value (reused byte-for-byte, design
// §4/§12 -- no new key set). Vendor client_id/client_secret are read from
// the corresponding env vars (design §6).
func NewMcpOAuthHandler(
	db dbhandler.DBHandler,
	cryptoKeys string,
	githubClientID string,
	githubClientSecret string,
	linearClientID string,
	linearClientSecret string,
) (McpOAuthHandler, error) {
	crypto, err := mcpserverhandler.NewSecretCrypto(cryptoKeys)
	if err != nil {
		return nil, err
	}

	return &mcpOAuthHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		crypto:      crypto,
		vendors:     newVendorCatalog(githubClientID, githubClientSecret, linearClientID, linearClientSecret),
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}
