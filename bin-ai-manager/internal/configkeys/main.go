// Package configkeys declares the configuration keys of bin-ai-manager.
//
// Two kinds of keys are declared.
//
// The platform LLM API keys are stored in the database only. They are independent keys on purpose:
// which of them a call uses is decided by the provider of the call (see aiconfig.KeyForBaseURL),
// and they are rotated separately.
//
// The MCP OAuth app of each vendor is an atomic group of a client_secret (the anchor, a secret) and
// a client_id (a plain parameter). A group is read from the database only, and only when its
// client_secret has a row: without it the app is not configured. No key declares an environment
// name.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "ai_configs"
	HistoryTable = "ai_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To make a key secret, declare a new key name.
const (
	// KeyOpenAICompatAPIKey is the key of an OpenAI compatible provider (OpenAI, Grok and so on).
	// The name is provider neutral because one key serves all of them.
	KeyOpenAICompatAPIKey = "llm.openai_compat.api_key"

	// KeyGeminiAPIKey is the Google API key of the Gemini endpoint.
	KeyGeminiAPIKey = "llm.gemini.api_key"
)

// Vendors of the MCP OAuth flow. The strings are the ones stored in McpServer.OAuthVendor and must
// equal the vendor identifiers of pkg/mcpoauthhandler (a test pins this).
const (
	VendorGitHub  = "github"
	VendorLinear  = "linear"
	VendorAsana   = "asana"
	VendorHubSpot = "hubspot"
)

const (
	// maxSecretLen is the longest accepted API key in bytes.
	maxSecretLen = 512

	// maxClientIDLen is the longest accepted OAuth client_id in bytes.
	maxClientIDLen = 256
)

// mcpOAuthVendors is the declared vendor list, in the order of Keys().
var mcpOAuthVendors = []string{VendorGitHub, VendorLinear, VendorAsana, VendorHubSpot}

// MCPOAuthVendors returns the vendors that have a declared OAuth app, in declaration order.
func MCPOAuthVendors() []string {
	return append([]string(nil), mcpOAuthVendors...)
}

func isMCPOAuthVendor(vendor string) bool {
	for _, v := range mcpOAuthVendors {
		if v == vendor {
			return true
		}
	}

	return false
}

// MCPOAuthClientSecretKey returns the name of the client_secret key (the anchor of the group) of the
// vendor. An unknown vendor gives ("", false).
func MCPOAuthClientSecretKey(vendor string) (string, bool) {
	if !isMCPOAuthVendor(vendor) {
		return "", false
	}

	return "mcp_oauth." + vendor + ".client_secret", true
}

// MCPOAuthClientIDKey returns the name of the client_id key of the vendor. An unknown vendor gives
// ("", false).
func MCPOAuthClientIDKey(vendor string) (string, bool) {
	if !isMCPOAuthVendor(vendor) {
		return "", false
	}

	return "mcp_oauth." + vendor + ".client_id", true
}

// mcpOAuthKeys declares the atomic group of one vendor: the client_secret (anchor) first, then the
// client_id. The client_id is not a secret: its stored value is readable in the history.
func mcpOAuthKeys(vendor, label string) []confighandler.Key {
	secretName, _ := MCPOAuthClientSecretKey(vendor)
	idName, _ := MCPOAuthClientIDKey(vendor)
	group := "mcp_oauth." + vendor

	return []confighandler.Key{
		{Name: secretName, Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: maxSecretLen, Group: group, Atomic: true, Description: "OAuth client_secret of the " + label + " app that connects customer MCP servers. The anchor of the " + label + " app: the app is configured only when this value and the client_id are stored. Write it after the client_id."},
		{Name: idName, Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: maxClientIDLen, Group: group, Atomic: true, Description: "OAuth client_id of the " + label + " app that connects customer MCP servers. Write it before the client_secret."},
	}
}

// Keys returns the declarations. Every key is stored in the database only: no key declares an
// environment name, so a key without a row is not configured. The MCP OAuth keys are an atomic
// group anchored on the client_secret: without a stored client_secret the client_id row is not read.
func Keys() []confighandler.Key {
	keys := []confighandler.Key{
		{Name: KeyOpenAICompatAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the OpenAI compatible provider that serves the conversation engine, and the summary, analysis and builder LLM calls when their base URL is not the Gemini endpoint.", MaxLen: maxSecretLen},
		{Name: KeyGeminiAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Google API key of the Gemini endpoint (audit and prompt proposal, and summary, analysis and builder when their base URL is the Gemini endpoint).", MaxLen: maxSecretLen},
	}

	keys = append(keys, mcpOAuthKeys(VendorGitHub, "GitHub")...)
	keys = append(keys, mcpOAuthKeys(VendorLinear, "Linear")...)
	keys = append(keys, mcpOAuthKeys(VendorAsana, "Asana")...)
	keys = append(keys, mcpOAuthKeys(VendorHubSpot, "HubSpot")...)

	return keys
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
