package mcpoauthhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// tokenResponse is the JSON shape returned by both vendors' token
// endpoints (subset of fields this handler needs). GitHub's classic OAuth
// Apps token endpoint returns form-encoded by default; requesting it with
// Accept: application/json (below) makes it return this same JSON shape.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // seconds; 0/absent = vendor did not report an expiry (design §4, e.g. GitHub)
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// errStateNotFoundOrUsed is the single generic error surfaced for every
// state-validation failure in Complete (not found, expired, or owned by a
// different customer) -- design §7a's anti-enumeration posture: never a
// more specific error that would let a caller distinguish these cases.
var errStateNotFoundOrUsed = cerrors.NotFound(
	commonoutline.ServiceNameAIManager,
	"MCP_OAUTH_STATE_NOT_FOUND",
	"The OAuth authorization state was not found or has already been used.",
)

// Complete implements McpOAuthHandler.Complete (design §7 steps 7-12,
// §7a's real security boundary).
func (h *mcpOAuthHandler) Complete(ctx context.Context, customerID uuid.UUID, state string, code string) (*mcpserver.McpServer, error) {
	row, err := h.db.McpOAuthStateGet(ctx, state)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, errStateNotFoundOrUsed
		}
		return nil, errors.Wrap(err, "could not get oauth state")
	}

	now := h.utilHandler.TimeNow()
	if row.IsExpired(*now) {
		return nil, errStateNotFoundOrUsed
	}

	// THE real security boundary (design §7a): the state row's
	// CustomerID must match the JWT-authenticated caller's customer_id.
	// Reject with the SAME generic error as "not found" -- never a more
	// specific mismatch error (anti-enumeration, same posture as Layer 1).
	if row.CustomerID != customerID {
		return nil, errStateNotFoundOrUsed
	}

	vc, ok := h.vendors[row.Vendor]
	if !ok {
		// Should be unreachable (Start already validated the vendor at
		// state-creation time), but fail closed rather than panic on a
		// map lookup for an unknown key.
		return nil, cerrors.Internal(commonoutline.ServiceNameAIManager, "INVALID_MCP_OAUTH_VENDOR", "the oauth state names an unknown vendor")
	}

	// Single-use: delete the state row BEFORE the token exchange (design
	// §7a Layer 1) -- a replayed /oauth/complete call for the same state
	// (retry or replay attempt) then fails at McpOAuthStateGet above with
	// the same generic not-found error, because this delete already ran.
	if err := h.db.McpOAuthStateDelete(ctx, state); err != nil {
		return nil, errors.Wrap(err, "could not delete oauth state")
	}

	tok, err := h.exchangeCode(ctx, vc, code, row.PKCEVerifier)
	if err != nil {
		return nil, errors.Wrap(err, "could not exchange oauth code for tokens")
	}

	accessCiphertext, accessNonce, keyVersion, err := h.crypto.Encrypt(tok.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt access token")
	}

	var refreshCiphertext, refreshNonce []byte
	if tok.RefreshToken != "" {
		refreshCiphertext, refreshNonce, _, err = h.crypto.Encrypt(tok.RefreshToken)
		if err != nil {
			return nil, errors.Wrap(err, "could not encrypt refresh token")
		}
	}

	var expiresAt *time.Time
	if tok.ExpiresIn > 0 {
		t := now.Add(time.Duration(tok.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	if row.McpServerID != nil {
		// Reconnect (design §9): ownership was already verified in
		// Start before the state row was created, so this is a plain
		// update, not a second ownership check.
		fields := map[mcpserver.Field]any{
			mcpserver.FieldAuthType:               mcpserver.AuthTypeOAuth,
			mcpserver.FieldOAuthVendor:            row.Vendor,
			mcpserver.FieldAccessTokenCiphertext:  accessCiphertext,
			mcpserver.FieldAccessTokenNonce:       accessNonce,
			mcpserver.FieldAccessTokenExpiresAt:   expiresAt,
			mcpserver.FieldRefreshTokenCiphertext: refreshCiphertext,
			mcpserver.FieldRefreshTokenNonce:      refreshNonce,
			mcpserver.FieldKeyVersion:             keyVersion,
		}
		if err := h.db.McpServerUpdate(ctx, *row.McpServerID, fields); err != nil {
			return nil, errors.Wrap(err, "could not update mcp server with oauth tokens")
		}
		res, err := h.db.McpServerGet(ctx, *row.McpServerID)
		if err != nil {
			return nil, errors.Wrap(err, "could not get updated mcp server")
		}
		return res, nil
	}

	m := &mcpserver.McpServer{
		Identity: identity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},

		Name:   vendorDisplayName(row.Vendor),
		Status: mcpserver.StatusActive,
		URL:    vc.MCPServerURL,

		AuthType:    mcpserver.AuthTypeOAuth,
		OAuthVendor: row.Vendor,

		AccessTokenCiphertext:  accessCiphertext,
		AccessTokenNonce:       accessNonce,
		AccessTokenExpiresAt:   expiresAt,
		RefreshTokenCiphertext: refreshCiphertext,
		RefreshTokenNonce:      refreshNonce,
		KeyVersion:             keyVersion,
	}

	if err := h.db.McpServerCreate(ctx, m); err != nil {
		return nil, errors.Wrap(err, "could not create mcp server for oauth connection")
	}

	res, err := h.db.McpServerGet(ctx, m.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get created mcp server")
	}

	return res, nil
}

// vendorDisplayName returns a human-readable name for a newly-created
// OAuth-connected McpServer row's Name field.
func vendorDisplayName(vendor string) string {
	switch vendor {
	case VendorGitHub:
		return "GitHub"
	case VendorLinear:
		return "Linear"
	default:
		return vendor
	}
}

// exchangeCode POSTs the authorization code (+ PKCE verifier) to the
// vendor's token endpoint and returns the parsed token response (design
// §7 step 10-11).
func (h *mcpOAuthHandler) exchangeCode(ctx context.Context, vc vendorConfig, code string, pkceVerifier string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", vc.ClientID)
	form.Set("client_secret", vc.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", oauthRedirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", pkceVerifier)

	return h.postTokenRequest(ctx, vc, form)
}

// refreshToken POSTs a grant_type=refresh_token request to the vendor's
// token endpoint (design §8 step 3). Shares postTokenRequest with
// exchangeCode -- same envelope, different form fields.
func (h *mcpOAuthHandler) refreshToken(ctx context.Context, vc vendorConfig, refreshToken string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", vc.ClientID)
	form.Set("client_secret", vc.ClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	return h.postTokenRequest(ctx, vc, form)
}

// postTokenRequest POSTs form to vc.TokenURL and parses the common
// token-response envelope both the authorization-code exchange
// (exchangeCode) and the refresh-token exchange (refreshToken) use.
func (h *mcpOAuthHandler) postTokenRequest(ctx context.Context, vc vendorConfig, form url.Values) (*tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vc.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "could not build token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// GitHub's classic OAuth Apps token endpoint returns form-encoded by
	// default; Accept: application/json makes both vendors return JSON.
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not call vendor token endpoint")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.Wrap(err, "could not read vendor token response")
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("could not parse vendor token response (status %d)", resp.StatusCode)
	}

	if tok.Error != "" {
		return nil, fmt.Errorf("vendor token endpoint returned error: %s: %s", tok.Error, tok.ErrorDesc)
	}
	if resp.StatusCode != http.StatusOK || tok.AccessToken == "" {
		return nil, fmt.Errorf("vendor token endpoint returned no access_token (status %d)", resp.StatusCode)
	}

	return &tok, nil
}
