package main

import (
	"context"

	"monorepo/bin-ai-manager/internal/aiconfig"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
)

// mcpOAuthCredentials adapts aiconfig.MCPOAuthClient to the function the MCP OAuth handler calls
// each time a request needs the OAuth app of a vendor. It adds nothing: the pair comes from the
// environment variables of the vendor, and an error is passed on as it is.
func mcpOAuthCredentials() mcpoauthhandler.CredentialsFunc {
	return func(_ context.Context, vendor string) (mcpoauthhandler.Credentials, error) {
		id, secret, err := aiconfig.MCPOAuthClient(vendor)
		if err != nil {
			return mcpoauthhandler.Credentials{}, err
		}

		return mcpoauthhandler.Credentials{ClientID: id, ClientSecret: secret}, nil
	}
}
