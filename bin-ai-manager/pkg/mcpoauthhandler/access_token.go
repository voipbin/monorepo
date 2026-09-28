package mcpoauthhandler

import (
	"context"
	"encoding/hex"
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

// tokenExchange is one refresh of one server's refresh token, shared by every
// caller holding that refresh token. done is closed when it finishes; the
// other fields are written before that and only read after.
type tokenExchange struct {
	done        chan struct{}
	accessToken string
	expiresAt   *time.Time
	err         error
}

// refreshExchanges holds the exchange in flight, or the last successful one,
// per server and refresh token. Process-local, like refreshBackoff: callers in
// different pods can still race, which this narrows but does not close.
var (
	refreshExchangesMu sync.Mutex
	refreshExchanges   = map[string]*tokenExchange{}
)

// persistRotatedTokenTimeout bounds the write that stores a vendor-rotated
// refresh token. Like the vendor call before it, it is detached from the
// caller's deadline on purpose; see GetValidAccessToken.
const persistRotatedTokenTimeout = 5 * time.Second

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

	// The refresh and the write that stores its result are one exchange that
	// runs to completion on its own, detached from the caller. The vendor
	// rotates the refresh token when it answers, invalidating the one we
	// hold, so an answer dropped mid-flight or a write abandoned after it
	// would strand the server with a dead refresh token and force the
	// customer to reconnect. The caller waits for the exchange only as long
	// as its own context allows.
	//
	// Concurrent callers holding the same refresh token share one exchange,
	// and a caller arriving after it finished, still holding the refresh
	// token it spent, is given its result. Presenting a spent refresh token
	// again fails at every vendor, and a vendor that detects reuse revokes
	// the whole grant.
	ex := h.refreshExchange(ctx, m, vc, refreshToken, now)

	select {
	case <-ex.done:
		if ex.err != nil {
			return "", ex.err
		}
		return ex.accessToken, nil
	case <-ctx.Done():
		return "", errors.Wrap(ctx.Err(), "gave up waiting for the oauth token refresh; it continues and will be stored")
	}
}

// refreshExchange returns the exchange for m's current refresh token,
// starting it if none is running or recorded. Exchanges are keyed by server
// and refresh token ciphertext, so a server whose row now holds a newer
// refresh token starts a new exchange rather than reusing an old answer.
func (h *mcpOAuthHandler) refreshExchange(ctx context.Context, m *mcpserver.McpServer, vc vendorConfig, refreshToken string, now *time.Time) *tokenExchange {
	key := m.ID.String() + ":" + hex.EncodeToString(m.RefreshTokenCiphertext)

	refreshExchangesMu.Lock()
	defer refreshExchangesMu.Unlock()

	if ex, ok := refreshExchanges[key]; ok {
		select {
		case <-ex.done:
			// Finished: reuse only a success whose access token is still
			// valid. A failure is not reused here; the backoff already
			// throttles retries against the vendor.
			if ex.err == nil && (ex.expiresAt == nil || now.Add(accessTokenExpiryMargin).Before(*ex.expiresAt)) {
				return ex
			}
			delete(refreshExchanges, key)
		default:
			return ex
		}
	}

	// Drop finished exchanges whose access token has expired, so the map
	// holds at most one live entry per server plus those still in flight.
	for k, old := range refreshExchanges {
		select {
		case <-old.done:
			if old.err != nil || (old.expiresAt != nil && !now.Before(*old.expiresAt)) {
				delete(refreshExchanges, k)
			}
		default:
		}
	}

	ex := &tokenExchange{done: make(chan struct{})}
	refreshExchanges[key] = ex

	exchangeCtx := context.WithoutCancel(ctx)
	go func() {
		defer close(ex.done)
		ex.accessToken, ex.expiresAt, ex.err = h.exchangeAndStore(exchangeCtx, m, vc, refreshToken, now)
		if ex.err != nil {
			refreshExchangesMu.Lock()
			if refreshExchanges[key] == ex {
				delete(refreshExchanges, key)
			}
			refreshExchangesMu.Unlock()
		}
	}()

	return ex
}

// exchangeAndStore performs one refresh against the vendor and stores the
// rotated tokens. The vendor call is bounded by h.httpClient's timeout and
// the write by persistRotatedTokenTimeout.
func (h *mcpOAuthHandler) exchangeAndStore(ctx context.Context, m *mcpserver.McpServer, vc vendorConfig, refreshToken string, now *time.Time) (string, *time.Time, error) {
	tok, err := h.refreshToken(ctx, vc, refreshToken)
	if err != nil {
		h.setRefreshBackoff(m.ID.String())
		return "", nil, errors.Wrap(err, "could not refresh oauth access token")
	}

	accessCiphertext, accessNonce, keyVersion, err := h.crypto.Encrypt(tok.AccessToken)
	if err != nil {
		return "", nil, errors.Wrap(err, "could not encrypt refreshed access token")
	}

	// OAuth 2.1 refresh token rotation: both GitHub and Linear issue a
	// new refresh token on every refresh (design §8 step 3) -- persist
	// it, or fall back to keeping the existing one if the vendor didn't
	// return a new one for this call.
	refreshCiphertext, refreshNonce := m.RefreshTokenCiphertext, m.RefreshTokenNonce
	if tok.RefreshToken != "" {
		refreshCiphertext, refreshNonce, _, err = h.crypto.Encrypt(tok.RefreshToken)
		if err != nil {
			return "", nil, errors.Wrap(err, "could not encrypt rotated oauth refresh token")
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
	persistCtx, cancel := context.WithTimeout(ctx, persistRotatedTokenTimeout)
	defer cancel()
	if err := h.db.McpServerUpdate(persistCtx, m.ID, fields); err != nil {
		return "", nil, errors.Wrap(err, "could not persist refreshed oauth tokens")
	}

	return tok.AccessToken, expiresAt, nil
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
