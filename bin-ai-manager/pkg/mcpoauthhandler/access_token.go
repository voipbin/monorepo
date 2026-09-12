package mcpoauthhandler

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/mcpserver"
)

// accessTokenExpiryMargin is the safety margin subtracted from
// AccessTokenExpiresAt when deciding whether a refresh is needed (design
// §8 step 2) -- avoids a race where the token expires between this check
// and the outbound MCP call actually using it.
const accessTokenExpiryMargin = 60 * time.Second

// refreshBackoff tracks the last failed-refresh time per McpServer ID, so
// a burst of tool calls against a server whose refresh token has been
// revoked vendor-side doesn't hammer the vendor's token endpoint on every
// single call (design §8 step 5, Round 1 review). Process-local and
// best-effort: a pod restart or multi-pod deployment simply loses/splits
// this state, which only costs a few extra doomed refresh attempts, not a
// correctness issue.
var (
	refreshBackoffMu sync.Mutex
	refreshBackoff   = map[string]time.Time{}
)

const refreshBackoffDuration = 60 * time.Second

// GetValidAccessToken implements McpOAuthHandler.GetValidAccessToken
// (design §8). Returns a decrypted, currently-valid OAuth access token
// for m, refreshing it first if it has expired (or is about to) and a
// refresh token is available.
func (h *mcpOAuthHandler) GetValidAccessToken(ctx context.Context, m *mcpserver.McpServer) (string, error) {
	accessToken, err := h.crypto.Decrypt(m.AccessTokenCiphertext, m.AccessTokenNonce, m.KeyVersion)
	if err != nil {
		return "", errors.Wrap(err, "could not decrypt oauth access token")
	}

	// nil expiry (e.g. GitHub classic OAuth Apps tokens) or still valid
	// with margin -- no refresh needed (design §8 step 2).
	now := h.utilHandler.TimeNow()
	if m.AccessTokenExpiresAt == nil || now.Add(accessTokenExpiryMargin).Before(*m.AccessTokenExpiresAt) {
		return accessToken, nil
	}

	if len(m.RefreshTokenCiphertext) == 0 {
		// Expired, no refresh token available (design §8 step 4) --
		// surfaced as a normal tool-call failure by the caller, same
		// as any other MCP call error.
		return "", errors.New("oauth access token expired and no refresh token is available; reconnect required")
	}

	if h.isInRefreshBackoff(m.ID.String()) {
		return "", errors.New("oauth token refresh recently failed for this server; skipping retry (backoff)")
	}

	refreshToken, err := h.crypto.Decrypt(m.RefreshTokenCiphertext, m.RefreshTokenNonce, m.KeyVersion)
	if err != nil {
		return "", errors.Wrap(err, "could not decrypt oauth refresh token")
	}

	vc, ok := h.vendors[m.OAuthVendor]
	if !ok {
		return "", errors.Errorf("mcp server names an unknown oauth vendor: %q", m.OAuthVendor)
	}

	tok, err := h.refreshToken(ctx, vc, refreshToken)
	if err != nil {
		h.setRefreshBackoff(m.ID.String())
		return "", errors.Wrap(err, "could not refresh oauth access token")
	}

	accessCiphertext, accessNonce, keyVersion, err := h.crypto.Encrypt(tok.AccessToken)
	if err != nil {
		return "", errors.Wrap(err, "could not encrypt refreshed access token")
	}

	// OAuth 2.1 refresh token rotation: both GitHub and Linear issue a
	// new refresh token on every refresh (design §8 step 3) -- persist
	// it, or fall back to keeping the existing one if the vendor didn't
	// return a new one for this call.
	refreshCiphertext, refreshNonce := m.RefreshTokenCiphertext, m.RefreshTokenNonce
	if tok.RefreshToken != "" {
		refreshCiphertext, refreshNonce, _, err = h.crypto.Encrypt(tok.RefreshToken)
		if err != nil {
			return "", errors.Wrap(err, "could not encrypt rotated oauth refresh token")
		}
	}

	var expiresAt *time.Time
	if tok.ExpiresIn > 0 {
		t := now.Add(time.Duration(tok.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	fields := map[mcpserver.Field]any{
		mcpserver.FieldAccessTokenCiphertext:  accessCiphertext,
		mcpserver.FieldAccessTokenNonce:       accessNonce,
		mcpserver.FieldAccessTokenExpiresAt:   expiresAt,
		mcpserver.FieldRefreshTokenCiphertext: refreshCiphertext,
		mcpserver.FieldRefreshTokenNonce:      refreshNonce,
		mcpserver.FieldKeyVersion:             keyVersion,
	}
	if err := h.db.McpServerUpdate(ctx, m.ID, fields); err != nil {
		return "", errors.Wrap(err, "could not persist refreshed oauth tokens")
	}

	return tok.AccessToken, nil
}

func (h *mcpOAuthHandler) isInRefreshBackoff(serverID string) bool {
	refreshBackoffMu.Lock()
	defer refreshBackoffMu.Unlock()

	until, ok := refreshBackoff[serverID]
	if !ok {
		return false
	}
	if h.utilHandler.TimeNow().After(until) {
		delete(refreshBackoff, serverID)
		return false
	}
	return true
}

func (h *mcpOAuthHandler) setRefreshBackoff(serverID string) {
	refreshBackoffMu.Lock()
	defer refreshBackoffMu.Unlock()

	refreshBackoff[serverID] = h.utilHandler.TimeNow().Add(refreshBackoffDuration)
}
