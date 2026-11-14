package configkeys

import (
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/confighandler"
)

func Test_NewRegistry(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(reg.Keys()) != 10 {
		t.Errorf("Wrong match. expect: 10 keys, got: %d", len(reg.Keys()))
	}
}

// Test_goldenKinds pins the kind, the type and the length limit of every key. A key name is
// never moved to another kind: rows and history that exist under the old meaning would be
// read as the new one. Declare a new key name instead, and update this table in the same change.
func Test_goldenKinds(t *testing.T) {
	expect := []struct {
		name   string
		kind   confighandler.Kind
		typ    confighandler.Type
		maxLen int
		group  string
		atomic bool
	}{
		{"llm.openai_compat.api_key", confighandler.KindSecret, confighandler.TypeString, 512, "", false},
		{"llm.gemini.api_key", confighandler.KindSecret, confighandler.TypeString, 512, "", false},
		{"mcp_oauth.github.client_secret", confighandler.KindSecret, confighandler.TypeString, 512, "mcp_oauth.github", true},
		{"mcp_oauth.github.client_id", confighandler.KindParam, confighandler.TypeString, 256, "mcp_oauth.github", true},
		{"mcp_oauth.linear.client_secret", confighandler.KindSecret, confighandler.TypeString, 512, "mcp_oauth.linear", true},
		{"mcp_oauth.linear.client_id", confighandler.KindParam, confighandler.TypeString, 256, "mcp_oauth.linear", true},
		{"mcp_oauth.asana.client_secret", confighandler.KindSecret, confighandler.TypeString, 512, "mcp_oauth.asana", true},
		{"mcp_oauth.asana.client_id", confighandler.KindParam, confighandler.TypeString, 256, "mcp_oauth.asana", true},
		{"mcp_oauth.hubspot.client_secret", confighandler.KindSecret, confighandler.TypeString, 512, "mcp_oauth.hubspot", true},
		{"mcp_oauth.hubspot.client_id", confighandler.KindParam, confighandler.TypeString, 256, "mcp_oauth.hubspot", true},
	}

	keys := Keys()
	if len(keys) != len(expect) {
		t.Fatalf("Wrong match. expect: %d keys, got: %d", len(expect), len(keys))
	}
	for i, e := range expect {
		k := keys[i]
		if k.Name != e.name || k.Kind != e.kind || k.Type != e.typ || k.MaxLen != e.maxLen {
			t.Errorf("Wrong match. key %d expect: %+v, got: %+v", i, e, k)
		}
		if k.Default != nil {
			t.Errorf("Wrong match. %s must have no default", k.Name)
		}
		if k.Env != "" {
			t.Errorf("Wrong match. %s expect: no environment name, got: %q", k.Name, k.Env)
		}
		if k.Group != e.group || k.Atomic != e.atomic {
			t.Errorf("Wrong match. key %d expect: group %q atomic %v, got: %q %v", i, e.group, e.atomic, k.Group, k.Atomic)
		}
		if k.Sensitive {
			t.Errorf("Wrong match. %s must not be sensitive", k.Name)
		}
	}
}

// Every key has a one line description for operators, and it never names a value.
func Test_descriptions(t *testing.T) {
	for _, k := range Keys() {
		if k.Description == "" {
			t.Errorf("Wrong match. %s has no description", k.Name)
		}
		if strings.ContainsAny(k.Description, "\r\n") || len(k.Description) > 300 {
			t.Errorf("Wrong match. %s: the description must be one short line, got: %q", k.Name, k.Description)
		}
	}
}

func Test_tableNames(t *testing.T) {
	if Table != "ai_configs" || HistoryTable != "ai_config_histories" {
		t.Errorf("Wrong match. the table names are part of the stored secret contract: %s %s", Table, HistoryTable)
	}
}

// Every vendor has an atomic group whose anchor is the client_secret, and the helpers return the
// declared names.
func Test_mcpOAuthGroups(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if got := MCPOAuthVendors(); len(got) != 4 {
		t.Fatalf("Wrong match. expect: 4 vendors, got: %v", got)
	}

	for _, vendor := range MCPOAuthVendors() {
		secretKey, ok1 := MCPOAuthClientSecretKey(vendor)
		idKey, ok2 := MCPOAuthClientIDKey(vendor)
		if !ok1 || !ok2 {
			t.Fatalf("Wrong match. %s: expect declared keys", vendor)
		}

		sk, ok := reg.Lookup(secretKey)
		if !ok || sk.Kind != confighandler.KindSecret || !sk.Atomic {
			t.Errorf("Wrong match. %s: the client_secret must be an atomic secret, got: %+v", vendor, sk)
		}
		ik, ok := reg.Lookup(idKey)
		if !ok || ik.Kind != confighandler.KindParam || ik.Group != sk.Group {
			t.Errorf("Wrong match. %s: the client_id must be in the group of the client_secret, got: %+v", vendor, ik)
		}
	}

	if _, ok := MCPOAuthClientSecretKey("unknown"); ok {
		t.Errorf("Wrong match. expect: unknown vendor not found")
	}
	if _, ok := MCPOAuthClientIDKey("unknown"); ok {
		t.Errorf("Wrong match. expect: unknown vendor not found")
	}
}

// MCPOAuthVendors returns a copy.
func Test_MCPOAuthVendors_copy(t *testing.T) {
	v := MCPOAuthVendors()
	v[0] = "changed"
	if MCPOAuthVendors()[0] != VendorGitHub {
		t.Errorf("Wrong match. expect: the list is not shared")
	}
}

// The environment variable names are the ones of the deployment definition, and the two platform
// keys keep their shared names.
func Test_envNames(t *testing.T) {
	if EnvOpenAIAPIKey != "OPENAI_API_KEY" || EnvGoogleAPIKey != "GOOGLE_API_KEY" {
		t.Errorf("Wrong match. expect: OPENAI_API_KEY and GOOGLE_API_KEY, got: %q %q", EnvOpenAIAPIKey, EnvGoogleAPIKey)
	}

	expect := map[string][2]string{
		VendorGitHub:  {"GITHUB_MCP_OAUTH_CLIENT_ID", "GITHUB_MCP_OAUTH_CLIENT_SECRET"},
		VendorLinear:  {"LINEAR_MCP_OAUTH_CLIENT_ID", "LINEAR_MCP_OAUTH_CLIENT_SECRET"},
		VendorAsana:   {"ASANA_MCP_OAUTH_CLIENT_ID", "ASANA_MCP_OAUTH_CLIENT_SECRET"},
		VendorHubSpot: {"HUBSPOT_MCP_OAUTH_CLIENT_ID", "HUBSPOT_MCP_OAUTH_CLIENT_SECRET"},
	}
	for _, vendor := range MCPOAuthVendors() {
		idName, secretName, ok := MCPOAuthEnvNames(vendor)
		if !ok || idName != expect[vendor][0] || secretName != expect[vendor][1] {
			t.Errorf("Wrong match. %s expect: %v, got: %q %q %v", vendor, expect[vendor], idName, secretName, ok)
		}
	}

	if idName, secretName, ok := MCPOAuthEnvNames("unknown"); ok || idName != "" || secretName != "" {
		t.Errorf("Wrong match. expect: unknown vendor not found, got: %q %q %v", idName, secretName, ok)
	}
}

// No key declares an environment name: a handler with a lookup would resolve the database first,
// and the admin listing must not suggest that the handler reads the environment.
func Test_keysDeclareNoEnv(t *testing.T) {
	for _, k := range Keys() {
		if k.Env != "" {
			t.Errorf("Wrong match. %s expect: no Env, got: %q", k.Name, k.Env)
		}
	}
}

// EnvNameOfKey names the variable that replaces a stored key: the one a read uses.
func Test_EnvNameOfKey(t *testing.T) {
	tests := []struct {
		key    string
		expect string
		ok     bool
	}{
		{KeyOpenAICompatAPIKey, "OPENAI_API_KEY", true},
		{KeyGeminiAPIKey, "GOOGLE_API_KEY", true},
		{"mcp_oauth.github.client_id", "GITHUB_MCP_OAUTH_CLIENT_ID", true},
		{"mcp_oauth.github.client_secret", "GITHUB_MCP_OAUTH_CLIENT_SECRET", true},
		{"mcp_oauth.asana.client_secret", "ASANA_MCP_OAUTH_CLIENT_SECRET", true},
		{"mcp_oauth.unknown.client_id", "", false},
		{"not.a.key", "", false},
	}
	for _, tt := range tests {
		got, ok := EnvNameOfKey(tt.key)
		if got != tt.expect || ok != tt.ok {
			t.Errorf("Wrong match. %s expect: %q %v, got: %q %v", tt.key, tt.expect, tt.ok, got, ok)
		}
	}

	// every declared key has a variable
	for _, k := range Keys() {
		if _, ok := EnvNameOfKey(k.Name); !ok {
			t.Errorf("Wrong match. %s expect: a variable name", k.Name)
		}
	}
}
