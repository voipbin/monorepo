package mcpoauthhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
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

	mockDB.EXPECT().McpServerUpdate(gomock.Any(), serverID, gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID, fields map[mcpserver.Field]any) error {
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
