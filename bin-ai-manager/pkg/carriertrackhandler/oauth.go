package carriertrackhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/mcpserver"
)

const (
	// tokenResponseMaxBytes bounds a token response; a token response is a
	// small json document, so the general response limit is not used.
	tokenResponseMaxBytes = 64 << 10

	// tokenMaxBytes bounds one access token.
	tokenMaxBytes = 4 << 10

	// tokenExpiryMargin is taken off expires_in so that a token is not used
	// when it is about to expire.
	tokenExpiryMargin = time.Minute

	// tokenMaxTTL is the longest a token is cached.
	tokenMaxTTL = time.Hour
)

// TokenAuthStyle is how the client credentials are sent to a token endpoint.
type TokenAuthStyle int

const (
	// TokenAuthBasic sends them as HTTP Basic authentication.
	TokenAuthBasic TokenAuthStyle = iota
	// TokenAuthFormBody sends client_id and client_secret in the form body.
	TokenAuthFormBody
)

// TokenSpec describes a carrier's OAuth client credentials token endpoint.
// The authentication style, the extra headers and form fields and the status
// mapping are the parts a carrier's documentation decides, so each carrier
// supplies its own.
type TokenSpec struct {
	// URL is a constant of the adapter, never a value of a customer.
	URL string

	Style        TokenAuthStyle
	ExtraHeaders map[string]string
	ExtraForm    map[string]string

	// Map overrides the default mapping of a non-200 status. A nil result
	// falls back to the default.
	Map func(status int) *CarrierError
}

// AccessToken returns the access token of the call's key from the cache or
// from a new exchange. The new token is cached when it carries a usable
// lifetime.
func (d Deps) AccessToken(ctx context.Context, spec TokenSpec, cred mcpserver.Credential) (string, *CarrierError) {
	if d.Tokens != nil && d.TokenKey != "" {
		if token, ok := d.Tokens.Get(d.TokenKey); ok {
			return token, nil
		}
	}

	token, ttl, cerr := d.ExchangeClientCredentials(ctx, spec, cred)
	if cerr != nil {
		promTokenExchangeTotal.WithLabelValues(d.Carrier, string(cerr.Code)).Inc()
		return "", cerr
	}
	promTokenExchangeTotal.WithLabelValues(d.Carrier, outcomeSuccess).Inc()

	// A result that came after the call's deadline is not stored.
	if d.Tokens != nil && d.TokenKey != "" && ttl > 0 && ctx.Err() == nil {
		d.Tokens.Put(d.TokenKey, token, ttl)
	}
	return token, nil
}

// ExchangeClientCredentials runs the OAuth client credentials grant once and
// returns the token with the time to cache it for (zero when it must not be
// cached). Every outcome that is not listed in the status mapping, a broken
// answer included, is carrier_unavailable. No text of the answer, no error of
// the transport and no part of the credential is returned or logged.
func (d Deps) ExchangeClientCredentials(ctx context.Context, spec TokenSpec, cred mcpserver.Credential) (string, time.Duration, *CarrierError) {
	unavailable := &CarrierError{Code: CodeCarrierUnavailable}

	form := url.Values{"grant_type": {"client_credentials"}}
	for k, v := range spec.ExtraForm {
		form.Set(k, v)
	}
	if spec.Style == TokenAuthFormBody {
		form.Set("client_id", cred.ID)
		form.Set("client_secret", cred.Secret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, spec.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, unavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	for k, v := range spec.ExtraHeaders {
		req.Header.Set(k, v)
	}
	if spec.Style == TokenAuthBasic {
		req.SetBasicAuth(cred.ID, cred.Secret)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return "", 0, unavailable
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if spec.Map != nil {
			if cerr := spec.Map(resp.StatusCode); cerr != nil {
				return "", 0, cerr
			}
		}
		return "", 0, defaultTokenStatusError(resp.StatusCode)
	}

	body, err := ReadBody(resp.Body, tokenResponseMaxBytes)
	if err != nil {
		return "", 0, unavailable
	}

	var parsed struct {
		AccessToken string          `json:"access_token"`
		TokenType   string          `json:"token_type"`
		ExpiresIn   json.RawMessage `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", 0, unavailable
	}
	if parsed.AccessToken == "" || len(parsed.AccessToken) > tokenMaxBytes || !printableASCII(parsed.AccessToken) {
		return "", 0, unavailable
	}
	if parsed.TokenType != "" && !strings.EqualFold(parsed.TokenType, "bearer") {
		return "", 0, unavailable
	}

	return parsed.AccessToken, tokenTTL(parsed.ExpiresIn), nil
}

// defaultTokenStatusError maps a non-200 status of a token endpoint. The
// mapping is provisional until a carrier's documentation confirms it: a 401 is
// taken as the carrier rejecting the key, a 400 is not blamed on the customer's
// key, a 403 is a short block and a 429 makes only this call busy.
func defaultTokenStatusError(status int) *CarrierError {
	switch status {
	case http.StatusUnauthorized:
		return &CarrierError{Code: CodeCredentialInvalid}
	case http.StatusForbidden:
		return &CarrierError{Code: CodeCarrierUnavailable, Cooldown: time.Minute}
	case http.StatusTooManyRequests:
		return &CarrierError{Code: CodeServiceBusy}
	}
	return &CarrierError{Code: CodeCarrierUnavailable}
}

// tokenTTL returns how long a token with the given expires_in is cached for.
// An absent, non-numeric or too short expires_in gives zero: the token is
// used for the current call only.
func tokenTTL(raw json.RawMessage) time.Duration {
	var seconds float64
	if err := json.Unmarshal(raw, &seconds); err != nil {
		return 0
	}
	if seconds > tokenMaxTTL.Seconds() {
		return tokenMaxTTL
	}
	ttl := time.Duration(seconds*float64(time.Second)) - tokenExpiryMargin
	if ttl <= 0 {
		return 0
	}
	if ttl > tokenMaxTTL {
		ttl = tokenMaxTTL
	}
	return ttl
}

// printableASCII reports whether s holds only visible ASCII characters, the
// only ones a bearer token may contain in a header.
func printableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
