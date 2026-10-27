package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func newTestMcpServer() *McpServer {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &McpServer{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		Name:                  "tools",
		Detail:                "detail text",
		URL:                   "https://mcp.example.com/rpc",
		Status:                StatusActive,
		AuthType:              AuthTypeAPIKey,
		APIKeyHeader:          "X-Api-Key",
		SecretCiphertext:      []byte("cipher-secret"),
		SecretNonce:           []byte("nonce-secret"),
		KeyVersion:            3,
		HasSecret:             true,
		OAuthVendor:           "vendor-a",
		AccessTokenCiphertext: []byte("cipher-access"),
		RefreshTokenNonce:     []byte("nonce-refresh"),
		TMCreate:              &created,
	}
}

func Test_ConvertWebhookMessage_fields(t *testing.T) {
	m := newTestMcpServer().ConvertWebhookMessage()

	if m.ID.String() != "11111111-1111-1111-1111-111111111111" || m.CustomerID.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("identity not mapped: %v", m.Identity)
	}
	if m.Name != "tools" || m.Detail != "detail text" {
		t.Errorf("name/detail not mapped: %q %q", m.Name, m.Detail)
	}
	if m.URL != "https://mcp.example.com/rpc" || m.Status != StatusActive {
		t.Errorf("url/status not mapped: %q %q", m.URL, m.Status)
	}
	if m.AuthType != AuthTypeAPIKey || m.APIKeyHeader != "X-Api-Key" || m.OAuthVendor != "vendor-a" {
		t.Errorf("auth fields not mapped: %q %q %q", m.AuthType, m.APIKeyHeader, m.OAuthVendor)
	}
	if !m.HasSecret {
		t.Errorf("HasSecret not mapped")
	}
	if m.TMCreate == nil || !m.TMCreate.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("TMCreate not mapped: %v", m.TMCreate)
	}
	if m.TMUpdate != nil || m.TMDelete != nil {
		t.Errorf("expected nil TMUpdate/TMDelete")
	}
}

func Test_CreateWebhookEvent_noSecrets(t *testing.T) {
	data, err := newTestMcpServer().CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(data)
	for _, leaked := range []string{"cipher-secret", "nonce-secret", "cipher-access", "nonce-refresh", "key_version", "ciphertext"} {
		if strings.Contains(s, leaked) {
			t.Errorf("webhook payload contains %q: %s", leaked, s)
		}
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["has_secret"] != true || got["auth_type"] != "api_key" || got["status"] != "active" {
		t.Errorf("unexpected payload: %v", got)
	}
}

func Test_CreateWebhookEvent_authTypeNoneStaysPresent(t *testing.T) {
	data, err := (&McpServer{}).CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if v, ok := got["auth_type"]; !ok || v != "" {
		t.Errorf("auth_type should be present as empty string, got %v (present: %v)", v, ok)
	}
	if v, ok := got["has_secret"]; !ok || v != false {
		t.Errorf("has_secret should be present as false, got %v", v)
	}
}
