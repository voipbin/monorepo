// Package configkeys names the environment variables of the platform keys of bin-ai-manager.
//
// The platform LLM API keys are independent variables on purpose: which of them a call uses is
// decided by the provider of the call (see aiconfig.KeyForBaseURL), and they are rotated
// separately. The MCP OAuth app of each vendor is a pair of a client_secret and a client_id.
//
// The values are read from environment variables only (the Env constants below name them and are
// the single list of names).
package configkeys

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

// mcpOAuthVendors is the declared vendor list, in declaration order.
var mcpOAuthVendors = []string{VendorGitHub, VendorLinear, VendorAsana, VendorHubSpot}

// MCPOAuthVendors returns the vendors that have a declared OAuth app, in declaration order.
func MCPOAuthVendors() []string {
	return append([]string(nil), mcpOAuthVendors...)
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

// EnvNames lists every variable that holds a platform key, in declaration order.
func EnvNames() []string {
	names := []string{EnvOpenAIAPIKey, EnvGoogleAPIKey}
	for _, vendor := range mcpOAuthVendors {
		idName, secretName, _ := MCPOAuthEnvNames(vendor)
		names = append(names, secretName, idName)
	}

	return names
}
