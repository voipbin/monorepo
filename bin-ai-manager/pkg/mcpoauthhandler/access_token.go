package mcpoauthhandler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
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
	finishedAt  *time.Time
	err         error
}

// exchangeRetention is how long a finished exchange is kept for callers that
// read the row before its result was stored; see tokenExchange.usable.
const exchangeRetention = 10 * time.Minute

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
	ex, err := h.refreshExchange(ctx, m, vc, refreshToken, now)
	if err != nil {
		return "", err
	}

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
// starting it if none is running or usable. Exchanges are keyed by server and
// a digest of the refresh token itself, not its ciphertext: every encryption
// draws a fresh nonce, so keying on ciphertext would split callers holding the
// same token under two keys, and each would spend it. A server whose row now
// holds a different refresh token gets a different key and a new exchange.
//
// The backoff is checked only when a new exchange would start: a caller that
// can join one in flight, or be handed a usable result, costs the vendor
// nothing, and refusing it would fail a call that has a valid token to use.
func (h *mcpOAuthHandler) refreshExchange(ctx context.Context, m *mcpserver.McpServer, vc vendorConfig, refreshToken string, now *time.Time) (*tokenExchange, error) {
	digest := sha256.Sum256([]byte(refreshToken))
	key := m.ID.String() + ":" + hex.EncodeToString(digest[:])

	refreshExchangesMu.Lock()
	defer refreshExchangesMu.Unlock()

	if ex, ok := refreshExchanges[key]; ok {
		select {
		case <-ex.done:
			if ex.usable(now) {
				return ex, nil
			}
			delete(refreshExchanges, key)
		default:
			return ex, nil
		}
	}

	if h.isInRefreshBackoff(m.ID.String()) {
		return nil, errors.New("oauth token refresh recently failed for this server; skipping retry (backoff)")
	}

	// Drop finished exchanges that can no longer be handed out, so the map
	// holds only what is in flight or recently finished.
	for k, old := range refreshExchanges {
		select {
		case <-old.done:
			if !old.usable(now) {
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
		ex.finishedAt = h.utilHandler.TimeNow()
		if ex.err != nil {
			refreshExchangesMu.Lock()
			if refreshExchanges[key] == ex {
				delete(refreshExchanges, key)
			}
			refreshExchangesMu.Unlock()
		}
	}()

	return ex, nil
}

// usable reports whether a finished exchange may be handed to a caller at
// now: it succeeded, its access token is valid with the same margin used to
// decide a refresh, and it finished within exchangeRetention. Only callers
// that read the row before the refresh was stored need it, and a call holds
// its row for seconds, so the retention bound costs nothing and keeps a
// decrypted access token out of memory once no caller can want it. It must
// only be called after done is closed.
func (ex *tokenExchange) usable(now *time.Time) bool {
	if ex.err != nil || ex.finishedAt == nil {
		return false
	}
	if now.Sub(*ex.finishedAt) > exchangeRetention {
		return false
	}
	return ex.expiresAt == nil || now.Add(accessTokenExpiryMargin).Before(*ex.expiresAt)
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
	// it, or keep the existing one if the vendor didn't return a new one
	// for this call.
	//
	// Either way it is encrypted afresh, under the same key as the access
	// token. The row has one key_version for both, so writing back the old
	// ciphertext alongside an access token encrypted under a newer key
	// would leave a refresh token that no longer decrypts, and the next
	// refresh would fail until the customer reconnects.
	keptRefreshToken := tok.RefreshToken
	if keptRefreshToken == "" {
		keptRefreshToken = refreshToken
	}
	refreshCiphertext, refreshNonce, refreshKeyVersion, err := h.crypto.Encrypt(keptRefreshToken)
	if err != nil {
		return "", nil, errors.Wrap(err, "could not encrypt oauth refresh token")
	}
	if refreshKeyVersion != keyVersion {
		return "", nil, errors.Errorf("oauth tokens were encrypted under different keys (%d, %d)", keyVersion, refreshKeyVersion)
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
	err = h.db.McpServerUpdateOAuthTokensIfCurrent(persistCtx, m.ID, m.RefreshTokenCiphertext, fields)
	switch {
	case err == nil:
	case errors.Is(err, dbhandler.ErrNotFound):
		// The row moved on while the vendor was answering: it was
		// reconnected, left OAuth, was deleted, or holds a newer rotation.
		// The tokens are not stored, so they cannot overwrite that, but the
		// caller still gets the access token it was refreshed for; its row
		// predates the change.
		log.WithField("mcp_server_id", m.ID).Info("Discarded an oauth refresh whose server row changed while it ran.")
	default:
		// The vendor has rotated the refresh token, so the row now holds a
		// spent one. Backing off keeps the next callers from presenting it
		// again, which a vendor that detects reuse answers by revoking the
		// grant. The result is still returned, and kept for callers that
		// hold the same row, so this refresh is not wasted.
		h.setRefreshBackoff(m.ID.String())
		log.WithField("mcp_server_id", m.ID).Errorf("Could not persist a rotated oauth refresh token; the server may need to be reconnected. err: %v", err)
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
