package mcpoauthhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// Test_GetValidAccessToken_NoExpiryReturnsAsIs verifies design §8 step 2:
// a nil AccessTokenExpiresAt (e.g. GitHub, which never reports one) means
// the decrypted token is returned as-is, with no refresh call and no
// McpServerUpdate.
func Test_GetValidAccessToken_NoExpiryReturnsAsIs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	ct, nonce, ver, err := h.crypto.Encrypt("gho_livetoken")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}

	m := &mcpserver.McpServer{
		OAuthVendor:           VendorGitHub,
		AccessTokenCiphertext: ct,
		AccessTokenNonce:      nonce,
		KeyVersion:            ver,
		AccessTokenExpiresAt:  nil,
	}

	// No McpServerUpdate expectation set -- gomock fails the test if one
	// is called unexpectedly.
	token, err := h.GetValidAccessToken(context.Background(), m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "gho_livetoken" {
		t.Fatalf("unexpected token: %q", token)
	}
}

// Test_GetValidAccessToken_StillValidReturnsAsIs verifies design §8 step
// 2's margin check: a token expiring well in the future is returned
// as-is, no refresh triggered.
func Test_GetValidAccessToken_StillValidReturnsAsIs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	ct, nonce, ver, err := h.crypto.Encrypt("linear-access-token")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}

	future := time.Now().Add(1 * time.Hour)
	m := &mcpserver.McpServer{
		OAuthVendor:           VendorLinear,
		AccessTokenCiphertext: ct,
		AccessTokenNonce:      nonce,
		KeyVersion:            ver,
		AccessTokenExpiresAt:  &future,
	}

	token, err := h.GetValidAccessToken(context.Background(), m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "linear-access-token" {
		t.Fatalf("unexpected token: %q", token)
	}
}

// Test_GetValidAccessToken_ExpiredNoRefreshTokenFails verifies design §8
// step 4: an expired token with no refresh token available returns an
// error rather than the stale token, so the caller (mcptoolhandler) fails
// the tool call rather than silently sending an expired Bearer header.
func Test_GetValidAccessToken_ExpiredNoRefreshTokenFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	ct, nonce, ver, err := h.crypto.Encrypt("stale-token")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}

	past := time.Now().Add(-1 * time.Hour)
	m := &mcpserver.McpServer{
		OAuthVendor:           VendorGitHub,
		AccessTokenCiphertext: ct,
		AccessTokenNonce:      nonce,
		KeyVersion:            ver,
		AccessTokenExpiresAt:  &past,
		// no RefreshTokenCiphertext
	}

	_, err = h.GetValidAccessToken(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error for expired token with no refresh token")
	}
}

// Test_GetValidAccessToken_RefreshesAndPersists verifies design §8 step
// 3: an expired token WITH a refresh token triggers a real POST to the
// vendor's token endpoint, and the new access/refresh token pair (OAuth
// 2.1 rotation) is re-encrypted and persisted via McpServerUpdate BEFORE
// the new access token is returned.
func Test_GetValidAccessToken_RefreshesAndPersists(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	var gotGrantType, gotRefreshToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrantType = r.Form.Get("grant_type")
		gotRefreshToken = r.Form.Get("refresh_token")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	h.vendors[VendorLinear] = vendorConfig{
		AuthorizeURL: "https://linear.app/oauth/authorize",
		TokenURL:     srv.URL,
		ClientID:     "linear-client-id",
		ClientSecret: "linear-client-secret",
	}
	h.httpClient = &http.Client{Timeout: 2 * time.Second}

	accessCT, accessNonce, ver, err := h.crypto.Encrypt("stale-access-token")
	if err != nil {
		t.Fatalf("could not encrypt access token: %v", err)
	}
	refreshCT, refreshNonce, _, err := h.crypto.Encrypt("old-refresh-token")
	if err != nil {
		t.Fatalf("could not encrypt refresh token: %v", err)
	}

	serverID := uuid.Must(uuid.NewV4())
	past := time.Now().Add(-1 * time.Hour)
	m := &mcpserver.McpServer{
		OAuthVendor:            VendorLinear,
		AccessTokenCiphertext:  accessCT,
		AccessTokenNonce:       accessNonce,
		KeyVersion:             ver,
		AccessTokenExpiresAt:   &past,
		RefreshTokenCiphertext: refreshCT,
		RefreshTokenNonce:      refreshNonce,
	}
	m.ID = serverID

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(func() *time.Time { n := time.Now(); return &n }()).AnyTimes()
	h.utilHandler = mockUtil

	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), serverID, m.RefreshTokenCiphertext, gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID, _ []byte, fields map[mcpserver.Field]any) error {
			// The persisted access token ciphertext must decrypt back
			// to the NEW token, not the stale one -- proves the update
			// actually happened before the function returns the token.
			ct, ok := fields[mcpserver.FieldAccessTokenCiphertext].([]byte)
			if !ok {
				t.Fatalf("FieldAccessTokenCiphertext missing or wrong type")
			}
			nonce, ok := fields[mcpserver.FieldAccessTokenNonce].([]byte)
			if !ok {
				t.Fatalf("FieldAccessTokenNonce missing or wrong type")
			}
			kv, ok := fields[mcpserver.FieldKeyVersion].(int)
			if !ok {
				t.Fatalf("FieldKeyVersion missing or wrong type")
			}
			decrypted, err := h.crypto.Decrypt(ct, nonce, kv)
			if err != nil {
				t.Fatalf("could not decrypt persisted access token: %v", err)
			}
			if decrypted != "new-access-token" {
				t.Fatalf("persisted access token = %q, want new-access-token", decrypted)
			}
			return nil
		},
	)

	token, err := h.GetValidAccessToken(context.Background(), m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "new-access-token" {
		t.Fatalf("returned token = %q, want new-access-token", token)
	}
	if gotGrantType != "refresh_token" {
		t.Fatalf("grant_type sent to vendor = %q, want refresh_token", gotGrantType)
	}
	if gotRefreshToken != "old-refresh-token" {
		t.Fatalf("refresh_token sent to vendor = %q, want old-refresh-token", gotRefreshToken)
	}
}

// Test_GetValidAccessToken_RefreshFailureSetsBackoff verifies design §8
// step 5 (Round 1 review fix): a failed refresh call sets a backoff so
// the immediately-following call does NOT re-attempt the vendor POST.
func Test_GetValidAccessToken_RefreshFailureSetsBackoff(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
	}))
	defer srv.Close()

	h.vendors[VendorLinear] = vendorConfig{
		TokenURL:     srv.URL,
		ClientID:     "linear-client-id",
		ClientSecret: "linear-client-secret",
	}
	h.httpClient = &http.Client{Timeout: 2 * time.Second}

	accessCT, accessNonce, ver, err := h.crypto.Encrypt("stale-access-token")
	if err != nil {
		t.Fatalf("could not encrypt access token: %v", err)
	}
	refreshCT, refreshNonce, _, err := h.crypto.Encrypt("revoked-refresh-token")
	if err != nil {
		t.Fatalf("could not encrypt refresh token: %v", err)
	}

	serverID := uuid.Must(uuid.NewV4())
	past := time.Now().Add(-1 * time.Hour)
	m := &mcpserver.McpServer{
		OAuthVendor:            VendorLinear,
		AccessTokenCiphertext:  accessCT,
		AccessTokenNonce:       accessNonce,
		KeyVersion:             ver,
		AccessTokenExpiresAt:   &past,
		RefreshTokenCiphertext: refreshCT,
		RefreshTokenNonce:      refreshNonce,
	}
	m.ID = serverID

	// First call: attempts the refresh, fails, sets backoff.
	_, err = h.GetValidAccessToken(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error on first (failing) refresh attempt")
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 vendor call, got %d", callCount)
	}

	// Second call, immediately after: must NOT hit the vendor again.
	_, err = h.GetValidAccessToken(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error on second (backoff) attempt")
	}
	if callCount != 1 {
		t.Fatalf("expected still exactly 1 vendor call (backoff should skip retry), got %d", callCount)
	}
}

// rotatingVendor is a fake token endpoint that rotates the refresh token on
// every successful refresh and rejects a refresh token it has already seen,
// as GitHub and Linear do. delay holds each answer, to widen races.
type rotatingVendor struct {
	mu      sync.Mutex
	live    string
	seen    map[string]bool
	calls   int
	reused  int
	delay   time.Duration
	release chan struct{}
}

func newRotatingVendor(initial string) *rotatingVendor {
	return &rotatingVendor{live: initial, seen: map[string]bool{}}
}

func (v *rotatingVendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	presented := r.PostForm.Get("refresh_token")

	if v.release != nil {
		<-v.release
	}
	time.Sleep(v.delay)

	v.mu.Lock()
	v.calls++
	if v.seen[presented] || presented != v.live {
		v.reused++
		v.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		return
	}
	v.seen[presented] = true
	v.live = fmt.Sprintf("refresh-%d", v.calls)
	next := v.live
	v.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  fmt.Sprintf("access-%d", v.calls),
		"refresh_token": next,
		"token_type":    "Bearer",
		"expires_in":    3600,
	})
}

func (v *rotatingVendor) counts() (calls int, reused int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls, v.reused
}

// expiredOAuthServer builds a handler pointed at vendor and a server row
// whose access token has expired and whose refresh token is refreshToken.
func expiredOAuthServer(t *testing.T, mockDB *dbhandler.MockDBHandler, vendor http.Handler, refreshToken string) (*mcpOAuthHandler, *mcpserver.McpServer) {
	t.Helper()

	h := newTestHandler(t, mockDB)
	srv := httptest.NewServer(vendor)
	t.Cleanup(srv.Close)

	h.vendors[VendorLinear] = vendorConfig{
		AuthorizeURL: "https://linear.app/oauth/authorize",
		TokenURL:     srv.URL,
		ClientID:     "linear-client-id",
		ClientSecret: "linear-client-secret",
	}
	h.httpClient = &http.Client{Timeout: 5 * time.Second}

	accessCT, accessNonce, ver, err := h.crypto.Encrypt("stale-access-token")
	if err != nil {
		t.Fatalf("could not encrypt access token: %v", err)
	}
	refreshCT, refreshNonce, _, err := h.crypto.Encrypt(refreshToken)
	if err != nil {
		t.Fatalf("could not encrypt refresh token: %v", err)
	}

	past := time.Now().Add(-1 * time.Hour)
	m := &mcpserver.McpServer{
		OAuthVendor:            VendorLinear,
		AccessTokenCiphertext:  accessCT,
		AccessTokenNonce:       accessNonce,
		KeyVersion:             ver,
		AccessTokenExpiresAt:   &past,
		RefreshTokenCiphertext: refreshCT,
		RefreshTokenNonce:      refreshNonce,
	}
	m.ID = uuid.Must(uuid.NewV4())

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().DoAndReturn(func() *time.Time { n := time.Now(); return &n }).AnyTimes()
	h.utilHandler = mockUtil

	return h, m
}

// Test_GetValidAccessToken_RefreshOutlivesCaller pins that a caller whose
// context ends while the vendor is answering is released at once, while the
// refresh runs on and its rotated token is still stored. The vendor has
// already invalidated the old refresh token by then, so dropping the answer
// or abandoning the write would force the customer to reconnect.
func Test_GetValidAccessToken_RefreshOutlivesCaller(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	vendor.release = make(chan struct{})
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")

	stored := make(chan map[mcpserver.Field]any, 1)
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).DoAndReturn(
		func(writeCtx context.Context, _ uuid.UUID, _ []byte, fields map[mcpserver.Field]any) error {
			if writeCtx.Err() != nil {
				t.Errorf("the rotated token write ran on a cancelled context: %v", writeCtx.Err())
			}
			if _, ok := writeCtx.Deadline(); !ok {
				t.Error("the rotated token write must still be bounded by a deadline")
			}
			stored <- fields
			return nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// The vendor does not answer until the caller has returned, so a caller
	// that waited for the exchange instead of its own context would never
	// return. Run it aside and fail cleanly rather than hang.
	returned := make(chan error, 1)
	go func() {
		_, err := h.GetValidAccessToken(ctx, m)
		returned <- err
	}()
	select {
	case err := <-returned:
		if err == nil || !strings.Contains(err.Error(), "gave up waiting") {
			t.Fatalf("expected the caller to give up at its deadline, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(vendor.release)
		t.Fatal("the caller was held past its own deadline waiting for the refresh")
	}

	close(vendor.release)
	select {
	case fields := <-stored:
		ct, _ := fields[mcpserver.FieldRefreshTokenCiphertext].([]byte)
		nonce, _ := fields[mcpserver.FieldRefreshTokenNonce].([]byte)
		got, err := h.crypto.Decrypt(ct, nonce, m.KeyVersion)
		if err != nil || got != "refresh-1" {
			t.Fatalf("stored refresh token = %q (err %v), want the rotated refresh-1", got, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the rotated token was never stored after the caller gave up")
	}
}

// Test_GetValidAccessToken_ConcurrentCallersShareOneRefresh pins that a burst
// of callers holding the same expired token spends the refresh token once.
// Without this every caller but one presents a spent refresh token, and a
// vendor that detects reuse revokes the grant.
func Test_GetValidAccessToken_ConcurrentCallersShareOneRefresh(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	vendor.delay = 100 * time.Millisecond
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).Return(nil).Times(1)

	const callers = 8
	var wg sync.WaitGroup
	tokens := make([]string, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tokens[i], errs[i] = h.GetValidAccessToken(context.Background(), m)
		}(i)
	}
	wg.Wait()

	for i := 0; i < callers; i++ {
		if errs[i] != nil || tokens[i] != "access-1" {
			t.Errorf("caller %d got %q, %v; want access-1", i, tokens[i], errs[i])
		}
	}
	if calls, reused := vendor.counts(); calls != 1 || reused != 0 {
		t.Fatalf("vendor saw %d refreshes and %d reused refresh tokens; want 1 and 0", calls, reused)
	}
}

// Test_GetValidAccessToken_LateCallerWithSpentTokenReusesResult pins that a
// caller arriving after a refresh finished, still holding the row it read
// before, is given that refresh's result instead of presenting the spent
// refresh token again.
func Test_GetValidAccessToken_LateCallerWithSpentTokenReusesResult(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).Return(nil).Times(1)

	for i := 0; i < 2; i++ {
		token, err := h.GetValidAccessToken(context.Background(), m)
		if err != nil || token != "access-1" {
			t.Fatalf("call %d got %q, %v; want access-1", i, token, err)
		}
	}
	if calls, reused := vendor.counts(); calls != 1 || reused != 0 {
		t.Fatalf("vendor saw %d refreshes and %d reused refresh tokens; want 1 and 0", calls, reused)
	}
}

// Test_GetValidAccessToken_FailedRefreshIsNotReused pins that a failed
// exchange is not handed to later callers: the backoff decides when a retry
// is allowed, and once it lapses the next caller starts a new exchange
// rather than receiving the old failure.
func Test_GetValidAccessToken_FailedRefreshIsNotReused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("some-other-token")
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")

	if _, err := h.GetValidAccessToken(context.Background(), m); err == nil || !strings.Contains(err.Error(), "could not refresh") {
		t.Fatalf("expected the vendor refusal, got: %v", err)
	}
	if _, err := h.GetValidAccessToken(context.Background(), m); err == nil || !strings.Contains(err.Error(), "backoff") {
		t.Fatalf("expected the second call to hit the backoff, got: %v", err)
	}
	if calls, _ := vendor.counts(); calls != 1 {
		t.Fatalf("vendor saw %d refreshes; want 1", calls)
	}

	// Let the backoff lapse. The next caller must reach the vendor again.
	refreshBackoffMu.Lock()
	delete(refreshBackoff, m.ID.String())
	refreshBackoffMu.Unlock()

	if _, err := h.GetValidAccessToken(context.Background(), m); err == nil || !strings.Contains(err.Error(), "could not refresh") {
		t.Fatalf("expected a fresh vendor refusal after the backoff, got: %v", err)
	}
	if calls, _ := vendor.counts(); calls != 2 {
		t.Fatalf("vendor saw %d refreshes; want 2, a failed exchange must not be reused", calls)
	}
}

// Test_GetValidAccessToken_RowMovedOnIsNotOverwritten pins that a refresh
// finishing after its row changed (reconnected, possibly to another account;
// downgraded out of OAuth; or already rotated) stores nothing, and that the
// caller still receives the token it was refreshed for without the server
// being put into backoff.
func Test_GetValidAccessToken_RowMovedOnIsNotOverwritten(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).Return(dbhandler.ErrNotFound)

	token, err := h.GetValidAccessToken(context.Background(), m)
	if err != nil || token != "access-1" {
		t.Fatalf("got %q, %v; want access-1", token, err)
	}
	if h.isInRefreshBackoff(m.ID.String()) {
		t.Fatal("a row that moved on is not a refresh failure and must not back off")
	}
}

// Test_GetValidAccessToken_PersistFailureBacksOff pins that when the vendor
// rotated the refresh token but storing it failed, the server backs off, so
// the next caller does not present the spent refresh token (which a vendor
// that detects reuse answers by revoking the grant). The refreshed access
// token is still returned, and callers holding the same row reuse it.
func Test_GetValidAccessToken_PersistFailureBacksOff(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).Return(fmt.Errorf("connection reset")).Times(1)

	for i := 0; i < 2; i++ {
		token, err := h.GetValidAccessToken(context.Background(), m)
		if err != nil || token != "access-1" {
			t.Fatalf("call %d got %q, %v; want access-1", i, token, err)
		}
	}
	if !h.isInRefreshBackoff(m.ID.String()) {
		t.Fatal("a rotated token that could not be stored must put the server into backoff")
	}
	if calls, reused := vendor.counts(); calls != 1 || reused != 0 {
		t.Fatalf("vendor saw %d refreshes and %d reused refresh tokens; want 1 and 0", calls, reused)
	}
}

// Test_GetValidAccessToken_SameTokenDifferentCiphertextShares pins that two
// rows holding the same refresh token under different ciphertexts (each
// encryption draws a fresh nonce) share one exchange instead of each
// spending the token.
func Test_GetValidAccessToken_SameTokenDifferentCiphertextShares(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	vendor.delay = 100 * time.Millisecond
	h, m1 := expiredOAuthServer(t, mockDB, vendor, "refresh-0")

	m2 := *m1
	ct, nonce, _, err := h.crypto.Encrypt("refresh-0")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}
	m2.RefreshTokenCiphertext, m2.RefreshTokenNonce = ct, nonce
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m1.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	var wg sync.WaitGroup
	for _, row := range []*mcpserver.McpServer{m1, &m2} {
		wg.Add(1)
		go func(row *mcpserver.McpServer) {
			defer wg.Done()
			if token, err := h.GetValidAccessToken(context.Background(), row); err != nil || token != "access-1" {
				t.Errorf("got %q, %v; want access-1", token, err)
			}
		}(row)
	}
	wg.Wait()

	if calls, reused := vendor.counts(); calls != 1 || reused != 0 {
		t.Fatalf("vendor saw %d refreshes and %d reused refresh tokens; want 1 and 0", calls, reused)
	}
}

// Test_tokenExchange_usable pins when a finished exchange may be handed out.
func Test_tokenExchange_usable(t *testing.T) {
	now := time.Now()
	at := func(d time.Duration) *time.Time { v := now.Add(d); return &v }

	tests := []struct {
		name string
		ex   tokenExchange
		want bool
	}{
		{name: "fresh success", ex: tokenExchange{finishedAt: at(-time.Minute), expiresAt: at(time.Hour)}, want: true},
		{name: "fresh success with no expiry", ex: tokenExchange{finishedAt: at(-time.Minute)}, want: true},
		{name: "failed", ex: tokenExchange{finishedAt: at(-time.Minute), err: fmt.Errorf("x")}, want: false},
		{name: "past retention", ex: tokenExchange{finishedAt: at(-exchangeRetention - time.Second), expiresAt: at(time.Hour)}, want: false},
		{name: "no expiry past retention", ex: tokenExchange{finishedAt: at(-exchangeRetention - time.Second)}, want: false},
		{name: "access token within the expiry margin", ex: tokenExchange{finishedAt: at(-time.Minute), expiresAt: at(accessTokenExpiryMargin / 2)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ex.usable(&now); got != tt.want {
				t.Fatalf("usable = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_GetValidAccessToken_NonRotatingVendorSurvivesKeyRotation pins that a
// refresh token the vendor did not rotate is re-encrypted under the key the
// row now claims. The row has one key_version for both tokens; writing back
// the old ciphertext beside an access token encrypted under a newer key left
// a refresh token that no longer decrypted, so the next refresh failed until
// the customer reconnected.
func Test_GetValidAccessToken_NonRotatingVendorSurvivesKeyRotation(t *testing.T) {
	const keyV1 = "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
	const keyV2 = "2:ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA="

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	// A vendor that answers every refresh with a new access token and no
	// refresh token, leaving the one it was given in place.
	var calls int
	vendor := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": fmt.Sprintf("access-%d", calls),
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	// The row was written under key 1.
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	oldCrypto, err := mcpserverhandler.NewSecretCrypto(keyV1)
	if err != nil {
		t.Fatalf("could not build crypto: %v", err)
	}
	m.AccessTokenCiphertext, m.AccessTokenNonce, m.KeyVersion, _ = oldCrypto.Encrypt("stale-access-token")
	m.RefreshTokenCiphertext, m.RefreshTokenNonce, _, _ = oldCrypto.Encrypt("refresh-0")

	// The operator then adds key 2, which becomes current.
	h.crypto, err = mcpserverhandler.NewSecretCrypto(keyV1 + "," + keyV2)
	if err != nil {
		t.Fatalf("could not build crypto: %v", err)
	}

	var stored map[mcpserver.Field]any
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ []byte, fields map[mcpserver.Field]any) error {
			stored = fields
			return nil
		},
	)

	if token, err := h.GetValidAccessToken(context.Background(), m); err != nil || token != "access-1" {
		t.Fatalf("got %q, %v; want access-1", token, err)
	}

	// The stored row must be self-consistent: both tokens decrypt under the
	// single key version it records, so the next refresh can read them.
	kv, _ := stored[mcpserver.FieldKeyVersion].(int)
	if kv != 2 {
		t.Fatalf("stored key version = %d, want the current key 2", kv)
	}
	rct, _ := stored[mcpserver.FieldRefreshTokenCiphertext].([]byte)
	rnonce, _ := stored[mcpserver.FieldRefreshTokenNonce].([]byte)
	got, err := h.crypto.Decrypt(rct, rnonce, kv)
	if err != nil || got != "refresh-0" {
		t.Fatalf("stored refresh token does not decrypt under the stored key version: %q, %v", got, err)
	}
}
