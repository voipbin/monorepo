// Package configkeys declares the configuration keys of bin-ai-manager.
//
// Two kinds of keys are declared.
//
// The platform LLM API keys are independent keys on purpose: which of them a call uses is decided
// by the provider of the call (see aiconfig.KeyForBaseURL), and they are rotated separately.
//
// The MCP OAuth app of each vendor is a group of a client_secret and a client_id.
//
// The values are read from environment variables only (the Env constants below name them and are
// the single list of names). The stored settings of these keys stay in the registry for the admin
// API and ai-control, but no read path uses them, and no key declares an environment name
// (Key.Env stays empty: a handler with an environment lookup would resolve the database first).
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

// Names of the environment variables that hold the platform keys. The OpenAI variable is shared
// with bin-pipecat-manager, and the Google variable too.
const (
	// EnvOpenAIAPIKey is the API key of the OpenAI compatible provider.
	EnvOpenAIAPIKey = "OPENAI_API_KEY"

	// EnvGoogleAPIKey is the Google API key of the Gemini endpoint.
	EnvGoogleAPIKey = "GOOGLE_API_KEY"
)

// Names of the environment variables of the MCP OAuth app of each vendor.
const (
	EnvGitHubMCPOAuthClientID      = "GITHUB_MCP_OAUTH_CLIENT_ID"
	EnvGitHubMCPOAuthClientSecret  = "GITHUB_MCP_OAUTH_CLIENT_SECRET"
	EnvLinearMCPOAuthClientID      = "LINEAR_MCP_OAUTH_CLIENT_ID"
	EnvLinearMCPOAuthClientSecret  = "LINEAR_MCP_OAUTH_CLIENT_SECRET"
	EnvAsanaMCPOAuthClientID       = "ASANA_MCP_OAUTH_CLIENT_ID"
	EnvAsanaMCPOAuthClientSecret   = "ASANA_MCP_OAUTH_CLIENT_SECRET"
	EnvHubSpotMCPOAuthClientID     = "HUBSPOT_MCP_OAUTH_CLIENT_ID"
	EnvHubSpotMCPOAuthClientSecret = "HUBSPOT_MCP_OAUTH_CLIENT_SECRET"
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

// MCPOAuthEnvNames returns the names of the environment variables of the client_id and the
// client_secret of the vendor. An unknown vendor gives ("", "", false).
func MCPOAuthEnvNames(vendor string) (idName, secretName string, ok bool) {
	switch vendor {
	case VendorGitHub:
		return EnvGitHubMCPOAuthClientID, EnvGitHubMCPOAuthClientSecret, true
	case VendorLinear:
		return EnvLinearMCPOAuthClientID, EnvLinearMCPOAuthClientSecret, true
	case VendorAsana:
		return EnvAsanaMCPOAuthClientID, EnvAsanaMCPOAuthClientSecret, true
	case VendorHubSpot:
		return EnvHubSpotMCPOAuthClientID, EnvHubSpotMCPOAuthClientSecret, true
	default:
		return "", "", false
	}
}

// EnvNames lists every variable that holds a platform key, in the order of the keys.
func EnvNames() []string {
	names := []string{EnvOpenAIAPIKey, EnvGoogleAPIKey}
	for _, vendor := range mcpOAuthVendors {
		idName, secretName, _ := MCPOAuthEnvNames(vendor)
		names = append(names, secretName, idName)
	}

	return names
}

// EnvNameOfKey returns the name of the environment variable that is read in place of the stored
// key. A name that is not a declared key gives ("", false).
func EnvNameOfKey(key string) (string, bool) {
	switch key {
	case KeyOpenAICompatAPIKey:
		return EnvOpenAIAPIKey, true
	case KeyGeminiAPIKey:
		return EnvGoogleAPIKey, true
	}

	for _, vendor := range mcpOAuthVendors {
		idName, secretName, _ := MCPOAuthEnvNames(vendor)
		if k, _ := MCPOAuthClientIDKey(vendor); k == key {
			return idName, true
		}
		if k, _ := MCPOAuthClientSecretKey(vendor); k == key {
			return secretName, true
		}
	}

	return "", false
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
		{Name: secretName, Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: maxSecretLen, Group: group, Atomic: true, Description: "OAuth client_secret of the " + label + " app that connects customer MCP servers. Stored settings are not used: the service reads the environment variable of the " + label + " app."},
		{Name: idName, Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: maxClientIDLen, Group: group, Atomic: true, Description: "OAuth client_id of the " + label + " app that connects customer MCP servers. Stored settings are not used: the service reads the environment variable."},
	}
}

// Keys returns the declarations of the stored settings of the admin API and ai-control. No key
// declares an environment name (Key.Env stays empty); the variables are the Env constants above.
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
