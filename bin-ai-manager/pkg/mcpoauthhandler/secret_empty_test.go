package mcpoauthhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

func Test_NewMcpOAuthHandler_errorHoldsNoKeyText(t *testing.T) {
	tests := []struct {
		name string
		keys string
	}{
		{"entry without a colon", "SENTINELNOCOLON"},
		{"version is not a number", "SENTINELVER:AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="},
		{"key is not base64", "1:SENTINELB64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewMcpOAuthHandler(nil, tt.keys, "", "", "", "", "", "", "", "")
			if err == nil || res != nil {
				t.Fatalf("expected an error and a nil handler, got %v / %v", res, err)
			}
			if strings.Contains(err.Error(), "SENTINEL") {
				t.Errorf("the error holds the key list text: %v", err)
			}
		})
	}
}

func Test_NewMcpOAuthHandler_blankEntriesMeanNoKey(t *testing.T) {
	for _, keys := range []string{"", ","} {
		res, err := NewMcpOAuthHandler(nil, keys, "", "", "", "", "", "", "", "")
		if err != nil || res == nil {
			t.Errorf("a list of blank entries (%q) must build a handler without a key, got %v / %v", keys, res, err)
		}
	}
}

// Test_GetValidAccessToken_NoKeyFails verifies that a handler without a key cannot read a saved token.
func Test_GetValidAccessToken_NoKeyFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	withKey := newTestHandler(t, dbhandler.NewMockDBHandler(mc))
	ct, nonce, ver, err := mcpserverhandler.EncryptSecret(withKey.crypto, "fake-token")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}

	noKeyHandler, err := NewMcpOAuthHandler(dbhandler.NewMockDBHandler(mc), "", "", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("could not build the handler: %v", err)
	}

	m := &mcpserver.McpServer{OAuthVendor: VendorGitHub, AccessTokenCiphertext: ct, AccessTokenNonce: nonce, KeyVersion: ver}
	_, err = noKeyHandler.GetValidAccessToken(context.Background(), m)
	if err == nil || !strings.Contains(err.Error(), "could not decrypt oauth access token") {
		t.Errorf("expected the access token decrypt error, got %v", err)
	}
}

// Test_GetValidAccessToken_emptyColumns verifies that a row with a missing column is an error, not a
// panic: both access token columns empty, and a refresh token with an empty nonce.
func Test_GetValidAccessToken_emptyColumns(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := newTestHandler(t, dbhandler.NewMockDBHandler(mc))
	ct, nonce, ver, err := mcpserverhandler.EncryptSecret(h.crypto, "fake-token")
	if err != nil {
		t.Fatalf("could not encrypt: %v", err)
	}
	past := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name   string
		server *mcpserver.McpServer
		expect string
	}{
		{
			name:   "access token columns empty",
			server: &mcpserver.McpServer{OAuthVendor: VendorGitHub, KeyVersion: ver},
			expect: "could not decrypt oauth access token",
		},
		{
			name: "refresh token nonce empty",
			server: &mcpserver.McpServer{
				OAuthVendor: VendorGitHub, AccessTokenCiphertext: ct, AccessTokenNonce: nonce, KeyVersion: ver,
				AccessTokenExpiresAt: &past, RefreshTokenCiphertext: ct,
			},
			expect: "could not decrypt oauth refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetValidAccessToken(context.Background(), tt.server)
			if err == nil || !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("expected %q, got %v", tt.expect, err)
			}
		})
	}
}
