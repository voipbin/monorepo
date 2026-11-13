package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler/storetest"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/internal/config"
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

// The adapter reads the real config handler: no row and an environment pair gives the environment
// pair, a stored pair gives the stored pair, and a stored client_secret without a client_id is an
// error that does not fall back to the environment.
func Test_mcpOAuthCredentials(t *testing.T) {
	t.Setenv("MCP_OAUTH_ASANA_CLIENT_ID", "env-id")
	t.Setenv("MCP_OAUTH_ASANA_CLIENT_SECRET", "env-secret")

	h, err := aiconfig.New(&config.Config{ConfigEncryptionKeys: testMasterKeyList()}, storetest.NewMemory())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	fn := mcpOAuthCredentials(h)
	ctx := context.Background()

	got, err := fn(ctx, mcpoauthhandler.VendorAsana)
	if err != nil || got != (mcpoauthhandler.Credentials{ClientID: "env-id", ClientSecret: "env-secret"}) {
		t.Errorf("Wrong match. expect: the environment pair, got: %+v %v", got, err)
	}

	// Not configured: an empty pair and no error.
	got, err = fn(ctx, mcpoauthhandler.VendorGitHub)
	if err != nil || got != (mcpoauthhandler.Credentials{}) {
		t.Errorf("Wrong match. expect: an empty pair, got: %+v %v", got, err)
	}

	actor := uuid.Must(uuid.NewV4())
	if _, err := h.SetSecret(ctx, actor, "mcp_oauth.asana.client_secret", "db-secret"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	_, err = fn(ctx, mcpoauthhandler.VendorAsana)
	if !errors.Is(err, aiconfig.ErrMCPOAuthIncomplete) {
		t.Errorf("Wrong match. expect: ErrMCPOAuthIncomplete, got: %v", err)
	}

	if _, err := h.Set(ctx, actor, "mcp_oauth.asana.client_id", "db-id"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	got, err = fn(ctx, mcpoauthhandler.VendorAsana)
	if err != nil || got != (mcpoauthhandler.Credentials{ClientID: "db-id", ClientSecret: "db-secret"}) {
		t.Errorf("Wrong match. expect: the stored pair, got: %+v %v", got, err)
	}

	// An unknown vendor is an error, not an empty pair.
	if _, err := fn(ctx, "not-a-vendor"); err == nil {
		t.Error("Wrong match. expect: an error for an unknown vendor")
	}
}

// A read error is passed on as ErrKeyUnavailable and never turned into the environment pair.
func Test_mcpOAuthCredentials_readErrorIsPassedOn(t *testing.T) {
	t.Setenv("MCP_OAUTH_GITHUB_CLIENT_ID", "env-id")
	t.Setenv("MCP_OAUTH_GITHUB_CLIENT_SECRET", "env-secret")

	store := storetest.NewMemory()
	store.GetErr = errors.New("database is down")
	h, err := aiconfig.New(&config.Config{ConfigEncryptionKeys: testMasterKeyList()}, store)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, err := mcpOAuthCredentials(h)(context.Background(), mcpoauthhandler.VendorGitHub)
	if !errors.Is(err, aiconfig.ErrKeyUnavailable) || got != (mcpoauthhandler.Credentials{}) {
		t.Errorf("Wrong match. expect: ErrKeyUnavailable and an empty pair, got: %+v %v", got, err)
	}
}
