package dbhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpserver"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// liveServerWithCredentials creates a server carrying every credential column
// the delete path is supposed to zero, through the real SQL layer.
func liveServerWithCredentials(t *testing.T, h DBHandler, id, customerID uuid.UUID) {
	t.Helper()

	m := &mcpserver.McpServer{
		Identity:         commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:             "test-server-credentials",
		URL:              "https://mcp.example.com/",
		Status:           mcpserver.StatusActive,
		SecretCiphertext: []byte{0xaa, 0xbb, 0xcc},
		SecretNonce:      []byte{0x01, 0x02, 0x03},
		KeyVersion:       1,
	}
	if err := h.McpServerCreate(context.Background(), m); err != nil {
		t.Fatalf("McpServerCreate failed: %v", err)
	}

	// The OAuth columns are written by the OAuth flow, not by Create, so set
	// them the same way it does. Without them the delete-zeroing assertions
	// below would pass on a server that never held a token.
	if err := h.McpServerUpdate(context.Background(), id, map[mcpserver.Field]any{
		mcpserver.FieldOAuthVendor:            "github",
		mcpserver.FieldAccessTokenCiphertext:  []byte{0x11, 0x22},
		mcpserver.FieldAccessTokenNonce:       []byte{0x33, 0x44},
		mcpserver.FieldRefreshTokenCiphertext: []byte{0x55, 0x66},
		mcpserver.FieldRefreshTokenNonce:      []byte{0x77, 0x88},
	}); err != nil {
		t.Fatalf("McpServerUpdate (seed oauth columns) failed: %v", err)
	}
}

// Test_McpServerDelete_ZeroesCredentials covers D4 through the real SQL layer.
// Revoking access must not leave decryptable material behind: before this fix
// McpServerDelete set only the timestamps, so a deleted server's secret and
// OAuth tokens stayed in the row indefinitely.
//
// It must be the SAME statement as the delete. A follow-up McpServerUpdate
// would be refused by that method's own `tm_delete IS NULL` predicate, leaving
// the credentials behind exactly when they matter most -- which is why this
// test asserts through Get rather than trusting the field map.
func Test_McpServerDelete_ZeroesCredentials(t *testing.T) {
	h := NewHandler(dbTest, nil)

	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	liveServerWithCredentials(t, h, id, customerID)

	before, err := h.McpServerGet(context.Background(), id)
	if err != nil {
		t.Fatalf("McpServerGet (before) failed: %v", err)
	}
	if !before.HasSecret {
		t.Fatalf("expected HasSecret=true before delete, got false")
	}
	if len(before.AccessTokenCiphertext) == 0 || len(before.RefreshTokenCiphertext) == 0 {
		t.Fatalf("expected the oauth token columns to be seeded before delete")
	}

	if err := h.McpServerDelete(context.Background(), id); err != nil {
		t.Fatalf("McpServerDelete failed: %v", err)
	}

	// Get deliberately still returns a soft-deleted row, which is what lets us
	// prove the credentials are gone rather than merely unreachable.
	after, err := h.McpServerGet(context.Background(), id)
	if err != nil {
		t.Fatalf("McpServerGet (after) failed: %v", err)
	}
	if after.TMDelete == nil {
		t.Errorf("expected tm_delete to be set after delete")
	}
	if len(after.SecretCiphertext) != 0 {
		t.Errorf("secret_ciphertext must be zeroed on delete, got %v", after.SecretCiphertext)
	}
	if len(after.SecretNonce) != 0 {
		t.Errorf("secret_nonce must be zeroed on delete, got %v", after.SecretNonce)
	}
	if after.KeyVersion != 0 {
		t.Errorf("key_version must be reset on delete, got %d", after.KeyVersion)
	}
	if len(after.AccessTokenCiphertext) != 0 {
		t.Errorf("access_token_ciphertext must be zeroed on delete, got %v", after.AccessTokenCiphertext)
	}
	if len(after.AccessTokenNonce) != 0 {
		t.Errorf("access_token_nonce must be zeroed on delete, got %v", after.AccessTokenNonce)
	}
	if len(after.RefreshTokenCiphertext) != 0 {
		t.Errorf("refresh_token_ciphertext must be zeroed on delete, got %v", after.RefreshTokenCiphertext)
	}
	if len(after.RefreshTokenNonce) != 0 {
		t.Errorf("refresh_token_nonce must be zeroed on delete, got %v", after.RefreshTokenNonce)
	}
	if after.HasSecret {
		t.Errorf("expected HasSecret=false after delete, got true")
	}

	// oauth_vendor is metadata, not a secret, and is deliberately retained so
	// the audit trail still shows which vendor the server had been linked to.
	if after.OAuthVendor != "github" {
		t.Errorf("oauth_vendor is metadata and must be retained, got %q", after.OAuthVendor)
	}
}

// Test_McpServerUpdate_RefusesDeletedRow covers D1/D2/D15 at the SQL layer: a
// soft-deleted server must be immutable and unresurrectable. Returning
// ErrNotFound rather than silently succeeding is the other half of the fix --
// the predicate alone would make the write a no-op while the caller still
// answered 200 and published an update event.
func Test_McpServerUpdate_RefusesDeletedRow(t *testing.T) {
	h := NewHandler(dbTest, nil)

	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	liveServerWithCredentials(t, h, id, customerID)

	if err := h.McpServerDelete(context.Background(), id); err != nil {
		t.Fatalf("McpServerDelete failed: %v", err)
	}

	err := h.McpServerUpdate(context.Background(), id, map[mcpserver.Field]any{
		mcpserver.FieldName:   "resurrected",
		mcpserver.FieldStatus: mcpserver.StatusActive,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound updating a soft-deleted server, got %v", err)
	}

	after, err := h.McpServerGet(context.Background(), id)
	if err != nil {
		t.Fatalf("McpServerGet failed: %v", err)
	}
	if after.Name == "resurrected" {
		t.Errorf("the refused update must not have written anything, got name %q", after.Name)
	}
	if after.TMDelete == nil {
		t.Errorf("the refused update must not have cleared tm_delete")
	}
}

// Test_McpServerUpdate_LiveRowStillUpdates is the negative control for the
// predicate above: a live row must keep updating normally. Without this, a
// predicate that refused everything would pass the test above.
func Test_McpServerUpdate_LiveRowStillUpdates(t *testing.T) {
	h := NewHandler(dbTest, nil)

	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	liveServerWithCredentials(t, h, id, customerID)

	if err := h.McpServerUpdate(context.Background(), id, map[mcpserver.Field]any{
		mcpserver.FieldName: "renamed",
	}); err != nil {
		t.Fatalf("updating a live server must succeed, got %v", err)
	}

	after, err := h.McpServerGet(context.Background(), id)
	if err != nil {
		t.Fatalf("McpServerGet failed: %v", err)
	}
	if after.Name != "renamed" {
		t.Errorf("expected the update to apply, got name %q", after.Name)
	}
}

// Test_McpServerDelete_RepeatReportsNotFound covers D16 at the SQL layer. The
// second delete matches no live row and says so; the HANDLER swallows that and
// still answers 200, because DELETE is idempotent by contract. Reporting it
// here is what lets the handler skip the duplicate deleted event instead of
// publishing one for a delete that did not happen.
func Test_McpServerDelete_RepeatReportsNotFound(t *testing.T) {
	h := NewHandler(dbTest, nil)

	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	liveServerWithCredentials(t, h, id, customerID)

	if err := h.McpServerDelete(context.Background(), id); err != nil {
		t.Fatalf("first McpServerDelete failed: %v", err)
	}
	if err := h.McpServerDelete(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on a repeat delete, got %v", err)
	}
}
