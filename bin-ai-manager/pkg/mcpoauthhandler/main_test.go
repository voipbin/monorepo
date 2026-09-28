package mcpoauthhandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// newTestHandler builds an *mcpOAuthHandler wired to mocks, with a real
// SecretCrypto (not exercised by Start/CallbackExists, only relevant for
// Complete's token-encryption path).
func newTestHandler(t *testing.T, db dbhandler.DBHandler) *mcpOAuthHandler {
	t.Helper()
	h, err := NewMcpOAuthHandler(
		db,
		"1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=",
		"github-client-id", "github-client-secret",
		"linear-client-id", "linear-client-secret",
	)
	if err != nil {
		t.Fatalf("could not build test mcpOAuthHandler: %v", err)
	}
	return h.(*mcpOAuthHandler)
}

// Test_Start_InvalidVendor pins that Start rejects an unknown vendor before
// touching the DB at all.
func Test_Start_InvalidVendor(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	customerID := uuid.Must(uuid.NewV4())

	_, _, err := h.Start(context.Background(), customerID, "not-a-real-vendor", nil)
	if err == nil {
		t.Fatal("expected an error for an unknown vendor, got nil")
	}
}

// Test_Start_OwnershipVerification pins design §9's IDOR fix: when a bare
// mcp_server_id is supplied (reconnect flow), Start must verify the row
// belongs to the requesting customer BEFORE creating any state row, and
// must return the SAME not-found error whether the row does not exist at
// all or belongs to a different customer (no enumeration signal).
func Test_Start_OwnershipVerification(t *testing.T) {
	otherCustomerID := uuid.Must(uuid.NewV4())
	requestingCustomerID := uuid.Must(uuid.NewV4())
	serverID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name         string
		getReturn    *mcpserver.McpServer
		getErr       error
		wantStateNew bool
	}{
		{
			name:      "mcp server not found at all",
			getReturn: nil,
			getErr:    dbhandler.ErrNotFound,
		},
		{
			name: "mcp server exists but belongs to a different customer",
			getReturn: &mcpserver.McpServer{
				Identity: identity.Identity{ID: serverID, CustomerID: otherCustomerID},
			},
			getErr: nil,
		},
		{
			// A deleted server is treated exactly like a missing one:
			// reconnecting would write fresh OAuth tokens onto a row the
			// customer has already revoked, and the delete path zeroes
			// precisely those token columns.
			name: "mcp server is owned but soft-deleted",
			getReturn: &mcpserver.McpServer{
				Identity: identity.Identity{ID: serverID, CustomerID: requestingCustomerID},
				TMDelete: func() *time.Time { ts := time.Now(); return &ts }(),
			},
			getErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandler(t, mockDB)

			mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(tt.getReturn, tt.getErr)
			// Ownership check must fail BEFORE any state row is created.
			mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Times(0)

			_, _, err := h.Start(context.Background(), requestingCustomerID, VendorGitHub, &serverID)
			if err == nil {
				t.Fatal("expected an ownership-check error, got nil")
			}
		})
	}
}

// Test_Start_OwnershipVerification_Success pins that Start proceeds to
// create a state row once ownership is confirmed.
func Test_Start_OwnershipVerification_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	customerID := uuid.Must(uuid.NewV4())
	serverID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(&mcpserver.McpServer{
		Identity: identity.Identity{ID: serverID, CustomerID: customerID},
	}, nil)
	mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Return(nil)

	authorizeURL, linkToken, err := h.Start(context.Background(), customerID, VendorGitHub, &serverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authorizeURL == "" {
		t.Error("expected a non-empty authorize_url")
	}
	if linkToken == "" {
		t.Error("expected a non-empty link_token")
	}
}

// Test_CallbackExists_StateExpiry pins the thin, no-mutation existence
// check's expiry semantics: not-found and expired both report false, an
// unexpired row reports true. Never deletes and never errors solely
// because the row is expired.
func Test_CallbackExists_StateExpiry(t *testing.T) {
	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	tests := []struct {
		name      string
		getReturn *mcpoauthstate.McpOAuthState
		getErr    error
		want      bool
	}{
		{
			name:   "state not found",
			getErr: dbhandler.ErrNotFound,
			want:   false,
		},
		{
			name: "state expired",
			getReturn: &mcpoauthstate.McpOAuthState{
				State:    state,
				TMExpire: &past,
			},
			want: false,
		},
		{
			name: "state still valid",
			getReturn: &mcpoauthstate.McpOAuthState{
				State:    state,
				TMExpire: &future,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := newTestHandler(t, mockDB)
			h.utilHandler = mockUtil

			mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(tt.getReturn, tt.getErr)
			if tt.getErr == nil {
				mockUtil.EXPECT().TimeNow().Return(&now)
			}
			// This is a strictly read-only check -- it must never delete the row.
			mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), gomock.Any()).Times(0)

			got, err := h.CallbackExists(context.Background(), state)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// Test_Complete_StateValidation pins Complete's anti-enumeration posture
// (design §7a): not-found, expired, and customer-mismatch all surface the
// SAME generic errStateNotFoundOrUsed, never a more specific error.
func Test_Complete_StateValidation(t *testing.T) {
	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	callingCustomerID := uuid.Must(uuid.NewV4())
	otherCustomerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		getReturn *mcpoauthstate.McpOAuthState
		getErr    error
	}{
		{
			name:   "state not found",
			getErr: dbhandler.ErrNotFound,
		},
		{
			name: "state expired",
			getReturn: &mcpoauthstate.McpOAuthState{
				State:      state,
				CustomerID: callingCustomerID,
				Vendor:     VendorGitHub,
				TMExpire:   &past,
			},
		},
		{
			name: "state belongs to a different customer",
			getReturn: &mcpoauthstate.McpOAuthState{
				State:      state,
				CustomerID: otherCustomerID,
				Vendor:     VendorGitHub,
				TMExpire:   &future,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := newTestHandler(t, mockDB)
			h.utilHandler = mockUtil

			mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(tt.getReturn, tt.getErr)
			if tt.getErr == nil {
				mockUtil.EXPECT().TimeNow().Return(&now)
			}
			// A rejected Complete must never delete the state row (so a
			// subsequent legitimate attempt, if any, is unaffected) and
			// must never reach the vendor token exchange.
			mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), gomock.Any()).Times(0)

			_, err := h.Complete(context.Background(), callingCustomerID, state, "some-code")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if err != errStateNotFoundOrUsed {
				t.Errorf("expected the generic errStateNotFoundOrUsed (anti-enumeration), got: %v", err)
			}
		})
	}
}

// Test_Complete_SingleUse pins that a valid, owned, unexpired state row is
// deleted BEFORE the vendor token exchange (single-use enforcement,
// design §7a) -- verified here via an httpClient stub that always fails,
// so we can assert the delete already happened by the time the exchange
// is attempted.
func Test_Complete_SingleUse(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockDB)
	h.utilHandler = mockUtil

	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	customerID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(&mcpoauthstate.McpOAuthState{
		State:        state,
		CustomerID:   customerID,
		Vendor:       VendorGitHub,
		PKCEVerifier: "verifier",
		TMExpire:     &future,
	}, nil)
	mockUtil.EXPECT().TimeNow().Return(&now)

	deleted := false
	mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), state).DoAndReturn(
		func(_ context.Context, _ string) error {
			deleted = true
			return nil
		},
	)

	// exchangeCode will fail (no real network target), which is fine --
	// this test only asserts ordering, not the full happy path.
	_, err := h.Complete(context.Background(), customerID, state, "some-code")
	if err == nil {
		t.Fatal("expected an error from the unreachable vendor token endpoint")
	}
	if !deleted {
		t.Error("expected the state row to be deleted before attempting the token exchange")
	}
}

// Test_Complete_ReconnectRefusesDeletedServer covers D3 on the callback side.
// Start verified ownership before the state row was created, but the customer
// can delete the server while the vendor's consent screen is up. Trusting
// Start's verdict would write fresh OAuth tokens onto a revoked row.
//
// The refusal must precede BOTH the single-use state delete and the token
// exchange: gomock's Times(0) on the delete is the assertion that nothing was
// burned on the way to refusing.
func Test_Complete_ReconnectRefusesDeletedServer(t *testing.T) {
	serverID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	otherCustomerID := uuid.Must(uuid.NewV4())
	ts := time.Now()

	tests := []struct {
		name      string
		getReturn *mcpserver.McpServer
	}{
		{
			name: "server soft-deleted between start and callback",
			getReturn: &mcpserver.McpServer{
				Identity: identity.Identity{ID: serverID, CustomerID: customerID},
				TMDelete: &ts,
			},
		},
		{
			// Re-checked rather than assumed: the row could also have been
			// reassigned, and Start's verdict is stale by this point.
			name: "server reassigned to another customer",
			getReturn: &mcpserver.McpServer{
				Identity: identity.Identity{ID: serverID, CustomerID: otherCustomerID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := newTestHandler(t, mockDB)
			h.utilHandler = mockUtil

			state := "test-state-token"
			now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
			future := now.Add(time.Minute)

			mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(&mcpoauthstate.McpOAuthState{
				State:        state,
				CustomerID:   customerID,
				McpServerID:  &serverID,
				Vendor:       VendorGitHub,
				PKCEVerifier: "verifier",
				TMExpire:     &future,
			}, nil)
			mockUtil.EXPECT().TimeNow().Return(&now)

			mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(tt.getReturn, nil)

			// Neither may happen: refusing after either would have burned the
			// single-use state row or minted tokens for an unusable server.
			mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), gomock.Any()).Times(0)
			mockDB.EXPECT().McpServerUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			if _, err := h.Complete(context.Background(), customerID, state, "some-code"); err == nil {
				t.Fatal("expected the reconnect to be refused, got nil")
			}
		})
	}
}

// Test_Complete_ReconnectSuccessPersistsTokens is the success-side control for
// the reconnect gate added in Test_Complete_ReconnectRefusesDeletedServer.
// Every other Complete test asserts a refusal or an exchange failure, so a
// gate that refused EVERY reconnect -- or a reconnect that wrote no tokens at
// all -- would ship green.
//
// The vendor token endpoint is stubbed with an httptest server (same approach
// as access_token_test.go's refresh tests) so the exchange actually succeeds
// and the reconnect runs end to end. The assertion inspects the real fields
// map handed to McpServerUpdate: both token ciphertext columns must be present
// and non-empty, and the access token must decrypt back to what the vendor
// returned.
func Test_Complete_ReconnectSuccessPersistsTokens(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockDB)
	h.utilHandler = mockUtil

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "reconnect-access-token",
			"refresh_token": "reconnect-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	h.vendors[VendorLinear] = vendorConfig{
		MCPServerURL: "https://mcp.linear.app/mcp",
		AuthorizeURL: "https://linear.app/oauth/authorize",
		TokenURL:     srv.URL,
		ClientID:     "linear-client-id",
		ClientSecret: "linear-client-secret",
	}
	h.httpClient = &http.Client{Timeout: 2 * time.Second}

	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	customerID := uuid.Must(uuid.NewV4())
	serverID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(&mcpoauthstate.McpOAuthState{
		State:        state,
		CustomerID:   customerID,
		McpServerID:  &serverID,
		Vendor:       VendorLinear,
		PKCEVerifier: "verifier",
		TMExpire:     &future,
	}, nil)
	mockUtil.EXPECT().TimeNow().Return(&now)

	// An owned, live server: the reconnect gate must let this through.
	existing := &mcpserver.McpServer{
		Identity: identity.Identity{ID: serverID, CustomerID: customerID},
	}
	reconnected := &mcpserver.McpServer{
		Identity:    identity.Identity{ID: serverID, CustomerID: customerID},
		AuthType:    mcpserver.AuthTypeOAuth,
		OAuthVendor: VendorLinear,
	}
	// First Get is the gate's ownership re-check, second is the post-update
	// read-back Complete returns.
	gomock.InOrder(
		mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(existing, nil),
		mockDB.EXPECT().McpServerGet(gomock.Any(), serverID).Return(reconnected, nil),
	)

	mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), state).Return(nil)

	updateCalled := false
	mockDB.EXPECT().McpServerUpdate(gomock.Any(), serverID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, fields map[mcpserver.Field]any) error {
			updateCalled = true

			// Every token column the reconnect is supposed to persist must be
			// present and non-empty. An empty (or partially populated) fields
			// map means the reconnect wrote no tokens -- the server would be
			// marked OAuth-connected while carrying nothing to authenticate
			// with.
			accessCT, _ := fields[mcpserver.FieldAccessTokenCiphertext].([]byte)
			if len(accessCT) == 0 {
				t.Errorf("FieldAccessTokenCiphertext missing, wrong type, or empty: %#v", fields[mcpserver.FieldAccessTokenCiphertext])
			}
			accessNonce, _ := fields[mcpserver.FieldAccessTokenNonce].([]byte)
			if len(accessNonce) == 0 {
				t.Errorf("FieldAccessTokenNonce missing, wrong type, or empty: %#v", fields[mcpserver.FieldAccessTokenNonce])
			}
			if refreshCT, _ := fields[mcpserver.FieldRefreshTokenCiphertext].([]byte); len(refreshCT) == 0 {
				t.Errorf("FieldRefreshTokenCiphertext missing, wrong type, or empty: %#v", fields[mcpserver.FieldRefreshTokenCiphertext])
			}
			if refreshNonce, _ := fields[mcpserver.FieldRefreshTokenNonce].([]byte); len(refreshNonce) == 0 {
				t.Errorf("FieldRefreshTokenNonce missing, wrong type, or empty: %#v", fields[mcpserver.FieldRefreshTokenNonce])
			}
			kv, kvOK := fields[mcpserver.FieldKeyVersion].(int)
			if !kvOK {
				t.Errorf("FieldKeyVersion missing or wrong type: %#v", fields[mcpserver.FieldKeyVersion])
			}

			// The persisted ciphertext must be the token the vendor actually
			// returned -- not a leftover, not an empty column.
			if kvOK && len(accessCT) > 0 && len(accessNonce) > 0 {
				decrypted, err := h.crypto.Decrypt(accessCT, accessNonce, kv)
				if err != nil {
					t.Errorf("could not decrypt persisted access token: %v", err)
				} else if decrypted != "reconnect-access-token" {
					t.Errorf("persisted access token = %q, want reconnect-access-token", decrypted)
				}
			}

			if fields[mcpserver.FieldAuthType] != mcpserver.AuthTypeOAuth {
				t.Errorf("FieldAuthType = %#v, want %v", fields[mcpserver.FieldAuthType], mcpserver.AuthTypeOAuth)
			}
			if fields[mcpserver.FieldOAuthVendor] != VendorLinear {
				t.Errorf("FieldOAuthVendor = %#v, want %v", fields[mcpserver.FieldOAuthVendor], VendorLinear)
			}
			return nil
		},
	)

	// A reconnect must never create a second server row.
	mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Times(0)

	res, err := h.Complete(context.Background(), customerID, state, "some-code")
	if err != nil {
		t.Fatalf("expected the reconnect to succeed, got: %v", err)
	}
	if !updateCalled {
		t.Fatal("expected McpServerUpdate to be called on a successful reconnect")
	}
	if res == nil || res.ID != serverID {
		t.Fatalf("expected the reconnected server %s, got: %#v", serverID, res)
	}
}

// Test_Complete_UnconfiguredVendorIsNotAnInvalidArgument pins the reason and
// status of the vendor-disappeared path.
//
// It matters because the reason code is published. INVALID_MCP_OAUTH_VENDOR is
// documented as a 400 the customer fixes by choosing a supported vendor; this
// path is a 500 the customer cannot fix, since the vendor was removed from the
// platform catalog while their authorization was pending. Reusing the 400's
// reason here would put one reason code behind two HTTP statuses on a page that
// promises they map one to one, and a client branching on the reason would retry
// a request that can never succeed.
//
// The state row is deliberately valid in every other respect -- owned, live,
// unexpired -- so the only thing under test is the unknown vendor.
func Test_Complete_UnconfiguredVendorIsNotAnInvalidArgument(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockDB)
	h.utilHandler = mockUtil

	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	customerID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(&mcpoauthstate.McpOAuthState{
		State:        state,
		CustomerID:   customerID,
		Vendor:       "a-vendor-that-is-no-longer-configured",
		PKCEVerifier: "verifier",
		TMExpire:     &future,
	}, nil)
	mockUtil.EXPECT().TimeNow().Return(&now)

	// The state must NOT be consumed: the failure is ours, so the customer's
	// pending authorization is not silently burned on our behalf.
	_, err := h.Complete(context.Background(), customerID, state, "auth-code")
	if err == nil {
		t.Fatal("expected Complete to refuse a state naming an unconfigured vendor")
	}

	var ve *commonerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a VoipbinError, got %T: %v", err, err)
	}
	if ve.Status != commonerrors.StatusInternal {
		t.Errorf("expected status %v (HTTP 500), got %v", commonerrors.StatusInternal, ve.Status)
	}
	if ve.Reason != "MCP_OAUTH_VENDOR_UNAVAILABLE" {
		t.Errorf("expected reason %q, got %q", "MCP_OAUTH_VENDOR_UNAVAILABLE", ve.Reason)
	}
	if ve.Reason == "INVALID_MCP_OAUTH_VENDOR" {
		t.Error("this path must not reuse the 400 reason: one reason code cannot carry two HTTP statuses")
	}
}
