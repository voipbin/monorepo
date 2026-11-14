package configkeys

import (
	"reflect"
	"testing"
)

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

// EnvNames lists the variable of every platform key once, in declaration order.
func Test_EnvNames(t *testing.T) {
	expect := []string{
		"OPENAI_API_KEY", "GOOGLE_API_KEY",
		"GITHUB_MCP_OAUTH_CLIENT_SECRET", "GITHUB_MCP_OAUTH_CLIENT_ID",
		"LINEAR_MCP_OAUTH_CLIENT_SECRET", "LINEAR_MCP_OAUTH_CLIENT_ID",
		"ASANA_MCP_OAUTH_CLIENT_SECRET", "ASANA_MCP_OAUTH_CLIENT_ID",
		"HUBSPOT_MCP_OAUTH_CLIENT_SECRET", "HUBSPOT_MCP_OAUTH_CLIENT_ID",
	}
	if got := EnvNames(); !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, got)
	}
}
