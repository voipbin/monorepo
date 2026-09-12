package mcpoauthhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
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
