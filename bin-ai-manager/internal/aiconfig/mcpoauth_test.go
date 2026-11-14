package aiconfig

import (
	"errors"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/internal/configkeys"
)

const testVendor = configkeys.VendorLinear

// setPair sets the variables of the pair of a vendor. An empty string leaves the variable empty.
func setPair(t *testing.T, vendor, id, secret string) {
	t.Helper()

	clearKeyEnv(t)
	idName, secretName, ok := configkeys.MCPOAuthEnvNames(vendor)
	if !ok {
		t.Fatalf("Wrong match. expect: a known vendor, got: %q", vendor)
	}
	t.Setenv(idName, id)
	t.Setenv(secretName, secret)
}

func Test_MCPOAuthClient(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		secret string

		expectID     string
		expectSecret string
		expectErr    error
	}{
		{"both set", "synthetic-id", "synthetic-secret", "synthetic-id", "synthetic-secret", nil},
		{"both set with spaces around", " synthetic-id ", " synthetic-secret\n", "synthetic-id", "synthetic-secret", nil},
		{"neither is set", "", "", "", "", nil},
		{"secret without id is incomplete", "", "synthetic-secret", "", "", ErrMCPOAuthIncomplete},
		{"secret with a placeholder id is incomplete", "[[BIN_MANAGER__LINEAR_MCP_OAUTH_CLIENT_ID]]", "synthetic-secret", "", "", ErrMCPOAuthIncomplete},
		{"id alone is the empty pair", "synthetic-id", "", "", "", nil},
		{"placeholder secret is not configured", "synthetic-id", "[[BIN_MANAGER__LINEAR_MCP_OAUTH_CLIENT_SECRET]]", "", "", nil},
		{"both placeholders", "[[A]]", "[[B]]", "", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setPair(t, testVendor, tt.id, tt.secret)

			id, secret, err := MCPOAuthClient(testVendor)
			if (tt.expectErr == nil && err != nil) || (tt.expectErr != nil && !errors.Is(err, tt.expectErr)) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if id != tt.expectID || secret != tt.expectSecret {
				t.Errorf("Wrong match. expect: %q %q, got: %q %q", tt.expectID, tt.expectSecret, id, secret)
			}
			if err != nil && (strings.Contains(err.Error(), tt.secret) || strings.Contains(err.Error(), "synthetic")) {
				t.Errorf("Wrong match. the error holds a value: %v", err)
			}
		})
	}
}

// Each vendor reads its own variables and no other vendor's.
func Test_MCPOAuthClient_everyVendor(t *testing.T) {
	for _, vendor := range configkeys.MCPOAuthVendors() {
		t.Run(vendor, func(t *testing.T) {
			setPair(t, vendor, "id-of-"+vendor, "secret-of-"+vendor)

			id, secret, err := MCPOAuthClient(vendor)
			if err != nil || id != "id-of-"+vendor || secret != "secret-of-"+vendor {
				t.Errorf("Wrong match. expect: the pair of the vendor, got: %q %q %v", id, secret, err)
			}

			for _, other := range configkeys.MCPOAuthVendors() {
				if other == vendor {
					continue
				}
				if id, secret, err := MCPOAuthClient(other); err != nil || id != "" || secret != "" {
					t.Errorf("Wrong match. %s expect: the empty pair, got: %q %q %v", other, id, secret, err)
				}
			}
		})
	}
}

func Test_MCPOAuthClient_unknownVendor(t *testing.T) {
	clearKeyEnv(t)

	if id, secret, err := MCPOAuthClient("not-a-vendor"); err == nil || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: an error and an empty pair, got: %q %q %v", id, secret, err)
	}
}

// The old variable names are not read.
func Test_MCPOAuthClient_oldNamesAreIgnored(t *testing.T) {
	clearKeyEnv(t)
	t.Setenv("MCP_OAUTH_LINEAR_CLIENT_ID", "synthetic-old-id")
	t.Setenv("MCP_OAUTH_LINEAR_CLIENT_SECRET", "synthetic-old-secret")

	if id, secret, err := MCPOAuthClient(testVendor); err != nil || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: the empty pair, got: %q %q %v", id, secret, err)
	}
}
