package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpserver"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Test_McpServerCreate_Get_SecretRoundTrip pins the real create->get round
// trip through the actual SQL layer (not a mocked DBHandler) for both a
// server with no secret and one with a real secret. This is the exact layer
// a prior PR review round found broken: PrepareFields' auto-detect
// conversion JSON-marshaled a raw []byte blob column (rather than passing
// it through), turning a nil SecretCiphertext into the literal 4-byte
// string "null" and a non-nil one into a quoted base64 JSON string -- both
// silently wrong, and invisible to any test that only inspects the value
// handed to a mocked db.McpServerCreate/Update before PrepareFields runs.
func Test_McpServerCreate_Get_SecretRoundTrip(t *testing.T) {
	h := NewHandler(dbTest, nil)

	tests := []struct {
		name string

		secretCiphertext []byte
		secretNonce      []byte
		keyVersion       int

		wantHasSecret        bool
		wantSecretCiphertext []byte
	}{
		{
			name:                 "no secret at creation",
			secretCiphertext:     nil,
			secretNonce:          nil,
			keyVersion:           0,
			wantHasSecret:        false,
			wantSecretCiphertext: nil,
		},
		{
			name:                 "a real secret at creation round-trips as raw bytes, not JSON",
			secretCiphertext:     []byte{0x01, 0x02, 0x03, 0x04, 0xff},
			secretNonce:          []byte{0x0a, 0x0b, 0x0c},
			keyVersion:           1,
			wantHasSecret:        true,
			wantSecretCiphertext: []byte{0x01, 0x02, 0x03, 0x04, 0xff},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.Must(uuid.NewV4())
			customerID := uuid.Must(uuid.NewV4())

			m := &mcpserver.McpServer{
				Identity:         commonidentity.Identity{ID: id, CustomerID: customerID},
				Name:             "test-server",
				URL:              "https://mcp.example.com/",
				Status:           mcpserver.StatusActive,
				SecretCiphertext: tt.secretCiphertext,
				SecretNonce:      tt.secretNonce,
				KeyVersion:       tt.keyVersion,
			}

			if err := h.McpServerCreate(context.Background(), m); err != nil {
				t.Fatalf("McpServerCreate failed: %v", err)
			}

			got, err := h.McpServerGet(context.Background(), id)
			if err != nil {
				t.Fatalf("McpServerGet failed: %v", err)
			}

			if got.HasSecret != tt.wantHasSecret {
				t.Errorf("expected HasSecret=%v, got %v", tt.wantHasSecret, got.HasSecret)
			}
			if string(got.SecretCiphertext) != string(tt.wantSecretCiphertext) {
				t.Errorf("expected SecretCiphertext=%v, got %v (must be raw bytes, never a JSON-marshaled value)", tt.wantSecretCiphertext, got.SecretCiphertext)
			}
		})
	}
}

// Test_McpServerUpdate_ClearSecret_RoundTrip pins the real update round trip
// for the explicit-clear-secret transition (design §10: a PUT with secret
// pointing at "" must zero the stored ciphertext/nonce/version, not leave a
// stale or corrupted value in the blob columns).
func Test_McpServerUpdate_ClearSecret_RoundTrip(t *testing.T) {
	h := NewHandler(dbTest, nil)

	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	m := &mcpserver.McpServer{
		Identity:         commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:             "test-server-to-clear",
		URL:              "https://mcp.example.com/",
		Status:           mcpserver.StatusActive,
		SecretCiphertext: []byte{0xaa, 0xbb, 0xcc},
		SecretNonce:      []byte{0x01, 0x02, 0x03},
		KeyVersion:       1,
	}
	if err := h.McpServerCreate(context.Background(), m); err != nil {
		t.Fatalf("McpServerCreate failed: %v", err)
	}

	before, err := h.McpServerGet(context.Background(), id)
	if err != nil {
		t.Fatalf("McpServerGet (before) failed: %v", err)
	}
	if !before.HasSecret {
		t.Fatalf("expected HasSecret=true before clearing, got false")
	}

	// Mirrors mcpServerHandler.Update's explicit-clear-secret branch.
	if err := h.McpServerUpdate(context.Background(), id, map[mcpserver.Field]any{
		mcpserver.FieldSecretCiphertext: []byte(nil),
		mcpserver.FieldSecretNonce:      []byte(nil),
		mcpserver.FieldKeyVersion:       0,
	}); err != nil {
		t.Fatalf("McpServerUpdate (clear) failed: %v", err)
	}

	after, err := h.McpServerGet(context.Background(), id)
	if err != nil {
		t.Fatalf("McpServerGet (after) failed: %v", err)
	}
	if after.HasSecret {
		t.Errorf("expected HasSecret=false after clearing, got true (SecretCiphertext=%v)", after.SecretCiphertext)
	}
	if len(after.SecretCiphertext) != 0 {
		t.Errorf("expected an empty SecretCiphertext after clearing, got %v", after.SecretCiphertext)
	}
}
