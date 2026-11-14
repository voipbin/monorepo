package main

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/configkeys"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
)

// The vendor identifiers of the config registry are the ones the handler stores and looks up.
func Test_mcpOAuthVendorsMatchTheHandler(t *testing.T) {
	expect := []string{
		mcpoauthhandler.VendorGitHub,
		mcpoauthhandler.VendorLinear,
		mcpoauthhandler.VendorAsana,
		mcpoauthhandler.VendorHubSpot,
	}

	got := configkeys.MCPOAuthVendors()
	if len(got) != len(expect) {
		t.Fatalf("Wrong match. expect: %v, got: %v", expect, got)
	}
	for i := range expect {
		if got[i] != expect[i] {
			t.Errorf("Wrong match. vendor %d expect: %q, got: %q", i, expect[i], got[i])
		}
	}
}

// The adapter reads the environment of the vendor: no variable gives an empty pair, a pair gives
// the pair, a secret without an id is an error, and an id alone is the empty pair.
func Test_mcpOAuthCredentials(t *testing.T) {
	for _, vendor := range configkeys.MCPOAuthVendors() {
		idName, secretName, _ := configkeys.MCPOAuthEnvNames(vendor)
		t.Setenv(idName, "")
		t.Setenv(secretName, "")
	}

	fn := mcpOAuthCredentials()
	ctx := context.Background()

	// Not configured: an empty pair and no error.
	got, err := fn(ctx, mcpoauthhandler.VendorGitHub)
	if err != nil || got != (mcpoauthhandler.Credentials{}) {
		t.Errorf("Wrong match. expect: an empty pair, got: %+v %v", got, err)
	}

	t.Setenv("ASANA_MCP_OAUTH_CLIENT_SECRET", "env-secret")
	_, err = fn(ctx, mcpoauthhandler.VendorAsana)
	if !errors.Is(err, aiconfig.ErrMCPOAuthIncomplete) {
		t.Errorf("Wrong match. expect: ErrMCPOAuthIncomplete, got: %v", err)
	}

	t.Setenv("ASANA_MCP_OAUTH_CLIENT_ID", "env-id")
	got, err = fn(ctx, mcpoauthhandler.VendorAsana)
	if err != nil || got != (mcpoauthhandler.Credentials{ClientID: "env-id", ClientSecret: "env-secret"}) {
		t.Errorf("Wrong match. expect: the pair, got: %+v %v", got, err)
	}

	t.Setenv("HUBSPOT_MCP_OAUTH_CLIENT_ID", "env-id-alone")
	got, err = fn(ctx, mcpoauthhandler.VendorHubSpot)
	if err != nil || got != (mcpoauthhandler.Credentials{}) {
		t.Errorf("Wrong match. expect: the empty pair for an id alone, got: %+v %v", got, err)
	}

	// An unknown vendor is an error, not an empty pair.
	if _, err := fn(ctx, "not-a-vendor"); err == nil {
		t.Error("Wrong match. expect: an error for an unknown vendor")
	}
}
