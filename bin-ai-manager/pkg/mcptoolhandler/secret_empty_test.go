package mcptoolhandler

import (
	"context"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

func Test_NewMcpToolHandler_errorHoldsNoKeyText(t *testing.T) {
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
			res, err := NewMcpToolHandler(nil, tt.keys, 0, nil, mcpserver.DefaultCarrierTable(), nil)
			if err == nil || res != nil {
				t.Fatalf("expected an error and a nil handler, got %v / %v", res, err)
			}
			if strings.Contains(err.Error(), "SENTINEL") {
				t.Errorf("the error holds the key list text: %v", err)
			}
		})
	}
}

func Test_NewMcpToolHandler_blankEntriesMeanNoKey(t *testing.T) {
	for _, keys := range []string{"", ","} {
		res, err := NewMcpToolHandler(nil, keys, 0, nil, mcpserver.DefaultCarrierTable(), nil)
		if err != nil || res == nil {
			t.Errorf("a list of blank entries (%q) must build a handler without a key, got %v / %v", keys, res, err)
		}
	}
}

// Test_buildAuthHeader_emptyColumns verifies that a row whose secret columns are empty is an error,
// not a panic.
func Test_buildAuthHeader_emptyColumns(t *testing.T) {
	crypto, err := mcpserverhandler.NewSecretCrypto(coverageTestKeys)
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}

	for _, authType := range []mcpserver.AuthType{mcpserver.AuthTypeBearer, mcpserver.AuthTypeAPIKey} {
		t.Run(string(authType), func(t *testing.T) {
			h := &mcpToolHandler{crypto: crypto}

			_, _, err := h.buildAuthHeader(context.Background(), &mcpserver.McpServer{AuthType: authType, KeyVersion: 1})
			if err == nil || !strings.Contains(err.Error(), "could not decrypt secret") {
				t.Errorf("expected the decrypt error, got %v", err)
			}
		})
	}
}

// Test_buildAuthHeader_noKey verifies that a handler without a key fails to read a saved secret.
func Test_buildAuthHeader_noKey(t *testing.T) {
	withKey, err := mcpserverhandler.NewSecretCrypto(coverageTestKeys)
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}
	ct, nonce, version, err := mcpserverhandler.EncryptSecret(withKey, "fake-secret-value")
	if err != nil {
		t.Fatalf("could not encrypt. err: %v", err)
	}

	noKey, err := mcpserverhandler.NewSecretCrypto("")
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}

	for _, authType := range []mcpserver.AuthType{mcpserver.AuthTypeBearer, mcpserver.AuthTypeAPIKey} {
		t.Run(string(authType), func(t *testing.T) {
			h := &mcpToolHandler{crypto: noKey}

			_, _, err := h.buildAuthHeader(context.Background(), &mcpserver.McpServer{AuthType: authType, SecretCiphertext: ct, SecretNonce: nonce, KeyVersion: version})
			if err == nil || !strings.Contains(err.Error(), "could not decrypt secret") {
				t.Errorf("expected the decrypt error, got %v", err)
			}
		})
	}
}
